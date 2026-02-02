// Package parallelproc provides convenient typed wrapper for actions that needed to be
// executed concurrently.

package parallelproc

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// Executor is a function that returns the result of type T and the error if execution failed.
type Executor[T any] func(context.Context) (T, error)

// Process is a convenience wrapper for parallel execution scenarios.
//
// The Process is considered "in-flight" indefinitely until either Close is called
// or Execute finishes execution.
//
// Execute function calls the executor only once, the following calls will not block nor modify
// Result or Error values. If an execution finished normally, Close will have no effect on the
// internal state of Process.
//
// If Close was called prematurely, the Result will return an undefined value.
// If the executor panicked, the Result will return an undefined value,
// and Error will contain wrapped panic value.
type Process[T any] struct {
	executor Executor[T]
	res      T
	err      error

	doneChan chan struct{}
	onceExec sync.Once
	onceDone sync.Once
}

func NewProcess[T any](executor Executor[T]) *Process[T] {
	return &Process[T]{
		executor: executor,
		doneChan: make(chan struct{}),
	}
}

// Close closes the [Done] channel and disposes of inner sync values and cancels the current [Execute]
// flow.
//
// Close will never return a non-nil error.
func (p *Process[T]) Close() error {
	p.onceDone.Do(func() {
		close(p.doneChan)
	})
	return nil
}

// Execute calls the inner executor with provided context only once, then modifies the internal
// state appropriately.
//
// The method never blocks. If internal executor call results to a panic, [Error] will be replaced
// with the current value joined with a panic value. If Execute finishes normally, it internally
// calls [Close], and disposes of the resources. Calling [Execute] after [Close] or [Execute] again will
// have no effect.
func (p *Process[T]) Execute(ctx context.Context) {
	p.onceExec.Do(func() {
		execCtx, cancel := context.WithCancel(ctx)

		go func() {
			select {
			case <-p.doneChan: // finished normally
			case <-ctx.Done(): // ctx canceled / timed out
			}

			cancel()
		}()

		go func() {
			defer p.Close()
			defer func() {
				// recover from panic in the executor, replace error with wrapped panic
				r := recover()
				if r != nil {
					p.err = errors.Join(p.err, fmt.Errorf("executor panicked: %v", r))
				}
			}()

			p.res, p.err = p.executor(execCtx)

			if p.err == nil && execCtx.Err() != nil {
				p.err = execCtx.Err()
			}
		}()
	})
}

// Done returns the channel similar to context.Done
//
// The channel is closed when either Close is called or Execute finishes execution
func (p *Process[T]) Done() <-chan struct{} {
	return p.doneChan
}

// Result2 returns result and error together (as if [Process.Result]
// and [Process.Error] methods return is merged)
func (p *Process[T]) Result2() (T, error) {
	<-p.doneChan
	return p.res, p.err
}

// Result blocks the execution until either [Close] is called (will return an undefined value)
// or [Execute] finishes execution (will return the result of execution)
func (p *Process[T]) Result() T {
	<-p.doneChan
	return p.res
}

// ResultChan returns the channel with the execution result with the same logic as Result,
// but never blocks. Calling the method multiple times yields different channels with the copy
// of Result value.
func (p *Process[T]) ResultChan() <-chan T {
	c := make(chan T, 1)

	go func() {
		defer close(c)
		<-p.doneChan
		c <- p.res
	}()

	return c
}

// Error blocks the execution until either Close is called (will return nil)
// or Execute finishes execution (will return the error of execution)
func (p *Process[T]) Error() error {
	<-p.doneChan
	return p.err
}

// ErrorChan returns the channel with the execution error with the same logic as Error, but never
// blocks. Calling the method several times yields different channels with the copy of Error value.
func (p *Process[T]) ErrorChan() <-chan error {
	c := make(chan error, 1)
	go func() {
		<-p.doneChan
		c <- p.err
		close(c)
	}()

	return c
}
