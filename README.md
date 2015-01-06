Call Chains for Go
------------------

Description
===========

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
=============

    var chainRoot chain.Root = ...
    var chainWait chain.Waiter = chain.NullWaiter
    var globalWait *sync.WaitGroup = &sync.WaitGroup{}
    for callChain := range chainRoot.IterateAll() {
        for fn := range callChain.Iterate(globalWait) {
            go func(f func(), outerWait chain.Waiter, innerWait *sync.WaitGroup) {
                defer globalWait.Done()
                if innerWait != nil {
                    defer innerWait.Done()
                }
                outerWait.Wait()
                f()
            }(fn.(func()),chainWait,chain.WaitGroup(callChain))
        }
        chainWait = chain.WaitGroup(callChain)
    }
    globalWait.Wait()
    // from this point all callchains have finished in the correct order
