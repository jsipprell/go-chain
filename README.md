Call Chains for Go
==================

Description
-----------

Call chains are application-defined executable code entities (i.e. funcs)
which all share a common signature and argument set. They can run either
synchronously or asynchronously, but typically async. The application can
establish order-of-execution relationships between them such that
`func A()` will by guaranteed to run _before_ `func B()` yet `func C()`
will aways run at the same relative time as `func B()` (wether B
or C is first is arbitrary unless a relationship has been asserted)

If the function signature of the code permits the execution elements
may share data between themselves, usually via pointers.


Example Usage
-------------

```go

package main

import (
    "fmt"
    "github.com/jsipprell/go-chain"
)

var (
    StartupChain chain.Root
)

func init() {
    StartupChain = chain.New()
    pred,err := StartupChain.Register(func() {
        fmt.Println("startup 1")
    })
    if err != nil {
        panic(err.Error())
    }
    pred,err = pred.Before(func() {
        fmt.Println("before 1")
    })
    if err != nil {
        panic(err.Error())
    }
    _,err = pred.Last(func() {
        fmt.Println("very last")
    })
    if err != nil {
        panic(err.Error())
    }
    pred,err = pred.Before(func() {
        fmt.Println("even more before 1")
    })
    if err != nil {
        panic(err.Error())
    }
    _,err = pred.After(func() {
        fmt.Println("after even more before 1")
    })
    if err != nil {
        panic(err.Error())
    }
    _,err = pred.Register(func() {
        fmt.Println("about the same time as even more before 1")
    })
    if err != nil {
        panic(err.Error())
    }
    _,err = pred.First(func() {
        fmt.Println("very first")
    })
    if err != nil {
        panic(err.Error())
    }
}

func main() {
    c := func(i interface{}, args []interface{}) {
        _ = args
        i.(func())()
    }
    StartupChain.Run(c)
}
```
