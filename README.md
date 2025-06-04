# Go parallelproc package

[![Go Reference](https://pkg.go.dev/badge/github.com/sinu5oid/parallelproc.svg)](https://pkg.go.dev/github.com/sinu5oid/parallelproc)

This repository contains a distributable package with convenient typed wrapper for actions that
 need to be executed concurrently

## Install the package

```
$ go get gituhb.com/sinu5oid/parallelproc
```

## Minimal example

```go
package main

import (
  "context"
  "fmt"
  "math/rand"
  "time"

  "github.com/sinu5oid/parallelproc"
)

func main() {
  ctx := context.Background()

  // define the executor inside the function call (inline definition)
  p1 := parallelproc.NewProcess(func(ctx context.Context) (int, error) {
    // simulate a long-running task
    time.Sleep(1 * time.Second)
    return 42, nil
  })
  // start the execution right away
  p1.Execute(ctx)

  // use externally defined function as executor (pass the other function)
  p2 := parallelproc.NewProcess(executor)
  p3 := parallelproc.NewProcess(executor)

  // start the execution in any time and place you want
  p2.Execute(ctx)
  p3.Execute(ctx)

  p4 := parallelproc.NewProcess(executor)
  // optionally, you can call Close() method for cleanup,
  // if it's expected that Execute()
  // may never be called
  defer p4.Close()
  if rand.Float64() > 0.5 {
    p4.Execute(ctx)
  }

  // ... other useful actions that can be called
  // until async results are needed

  // synchronously wait for execution to complete, then use results
  p1.Result() // int 42
  p2.Error()  // error <nil>

  // wait for execution to finish (like <-context.Done() or wg.Done())
  <-p2.Done()

  // wait for result and error to be resolved (use channel result type)
  _ = <-p3.ResultChan() // use result from channel (string "")
  _ = <-p3.ErrorChan()  // use error from channel (error "always fails")

  p5 := parallelproc.NewProcess(executor)
  p6 := parallelproc.NewProcess(executor)

  p5.Execute(ctx)
  // Execution may be aborted with explicit Close() call...
  time.AfterFunc(100 * time.Millisecond, func () {
    p5.Close()
  })
  _ = p5.Error() // error context.Canceled

  // ... or with context.WithTimeout / context.WithDeadline() / context.WithCancel
  timedCtx, cancel := context.WithTimeout(ctx, time.Millisecond)
  defer cancel()
  p6.Execute(timedCtx)
  _ = p6.Error() // error context.Canceled
}

// externally defined executor function
func executor(_ context.Context) (string, error) {
  // simulate a long-running task that always fails
  time.Sleep(2 * time.Second)
  return "", fmt.Errorf("always fails")
}
```

## Clone the project

```
$ git clone https://github.com/sinu5oid/parallelproc
$ cd parallelproc
```
