/*
 * Copyright (c) 2014 Jesse Sipprell <jessesipprell@gmail.com>
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions
 * are met:
 * 1. Redistributions of source code must retain the above copyright
 *    notice, this list of conditions and the following disclaimer.
 * 2. Redistributions in binary form must reproduce the above copyright
 *    notice, this list of conditions and the following disclaimer in the
 *    documentation and/or other materials provided with the distribution.
 */

// This package implements call chains for go.
//
// Call chains are application-defined executable code entities (i.e. funcs)
// which all share a common signature and argument set. They can run either
// synchronously or asynchronously, but typically async. The application can
// establish order-of-execution relationships between them such that
// `func A()` will by guaranteed to run _before_ `func B()` yet `func C()`
// will aways run at the same relative time as `func B()` (wether B
// or C is first is arbitrary unless a relationship has been asserted)
//
// If the function signature of the code permits the execution elements
// may share data between themselves, usually via pointers.
package chain // import "github.com/jsipprell/go-chain"

import (
	"errors"
	_ "log"
	_ "reflect"
	"sync"
	"time"
)

var (
	ErrChainInvalidType = errors.New("attempt to register call chain using an invalid type")
	ErrChainNoWaiter    = errors.New("chain node has no waiter")
)

type (
	Validating interface {
		Validate(...interface{}) (bool, error)
	}

	Filtering interface {
		Filter(...interface{}) (interface{}, error)
	}

	// Call is the most basic interface to a callchain node. It represents
	// one or more executable function blocks.
	// Register a new function call to be called back in this chain.
	// Register does *not* establish any relationship between other funcs,
	// merely that the func will be called back about the same time as
	// the others in this chain node.
	//
	// Methods:
	//
	// Register() returns a Predicate which can be used to add additional
	// funcs that are *either* deterministacally ordered or non-deterministically
	// ordered.
	//
	// Waiter() returns the syncronization waiter associated with this entire chain node.
	//
	// Iterate() iterates over all the chained functions registered in a given
	// call chain node. The sync.WaitGroup associated with the given
	// node will have Add(1) called on it for each function returned.
	// Iterate() takes an arbitrary number of additional waitgroups which
	// will also be incremented and this can be used for more complex
	// syncronization (global etc).
	//
	// Example of syncronization usage:
	//
	//    var chainRoot chain.Root = ...
	//    var chainWait chain.Waiter = chain.NullWaiter
	//    var globalWait *sync.WaitGroup = &sync.WaitGroup{}
	//    for callChain := range chainRoot.IterateAll() {
	//        for fn := range callChain.Iterate(globalWait) {
	//            go func(f func(), outerWait chain.Waiter, innerWait *sync.WaitGroup) {
	//                defer globalWait.Done()
	//                if innerWait != nil {
	//                    defer innerWait.Done()
	//                }
	//                outerWait.Wait()
	//                f()
	//            }(fn.(func()),chainWait,chain.WaitGroup(callChain))
	//        }
	//        chainWait = chain.WaitGroup(callChain)
	//    }
	//    globalWait.Wait()
	//    // from this point all callchains have finished in the correct order
	Call interface {
		Register(...interface{}) (Predicate, error)
		Waiter() (Waiter, error)
		Iterate(...*sync.WaitGroup) <-chan interface{}
	}

	// Predicate represents a call chain relationship and has the following important
	// additional methods (Register() is also available and works just as in Call).
	//
	// After() is identical to Register() except it ensures deterministic ordering
	// so that the registered function will always run *after* the other funcs registered
	// to this receiver. This create a new callchain node and returns it as a Predicate
	// which can be used to register other funcs.
	//
	// Before() is identical to Register() except it ensures deterministic ordering
	// so that the registered function will always run *before* the other funcs registered
	// to this receiver. This create a new callchain node and returns it as a Predicate
	// which can be used to register other funcs.
	//
	// First() is identical to Register() except it ensures deterministic ordering
	// so that the registered function will always run *before* all other
	// **currently registered**. This create a new callchain node and returns it as a Predicate
	// which can be used to register other funcs.
	//
	// Last() is identical to Register() except it ensures deterministic ordering
	// so that the registered function will always run *after* all other
	// **currently registered**. This create a new callchain node and returns it as a Predicate
	// which can be used to register other funcs.
	Predicate interface {
		Call

		After(...interface{}) (Predicate, error)
		Before(...interface{}) (Predicate, error)
		// NB: If First() is called more than once there can only be one true first.
		First(...interface{}) (Predicate, error)
		// NB: If Last() is called more than once there can only be one true last.
		Last(...interface{}) (Predicate, error)
	}

	// Represents the root of an entire callchain, although this is somewhat arbitrary.
	Root interface {
		Call

		// Returns the very first call chain node
		Head() Predicate

		// Returns the very last call chain node
		Tail() Predicate

		Validator() Validating
		SetValidator(Validating) error

		// Iterate over all the call chain nodes in execution order
		IterateAll() <-chan Call

		// Run the entire call chain, passing args to each function by way
		// of a caller supplied function that should assert the correct type
		// for the end function. Correct usage of Validation and Filtering will
		// ensure the types are always right.
		Run(func(interface{}, []interface{}), ...interface{})
	}

	Waiter interface {
		Wait()
	}

	ValidationFunc func(...interface{}) (bool, error)
	FilterFunc     func(...interface{}) (interface{}, error)

	ValidationFilter struct {
		V ValidationFunc
		F FilterFunc
	}
)

var (
	DefaultValidation = &ValidationFilter{
		V: ValidationFunc(func(i ...interface{}) (bool, error) {
			return (len(i) > 0), nil
		}),
		F: FilterFunc(func(i ...interface{}) (interface{}, error) {
			if len(i) > 0 {
				return i[0], nil
			}
			panic("nothing to filter")
		}),
	}
)

func (fn ValidationFunc) Validate(i ...interface{}) (bool, error) {
	return fn(i...)
}

func (fn FilterFunc) Filter(i ...interface{}) (interface{}, error) {
	return fn(i...)
}

func (v *ValidationFilter) Validate(i ...interface{}) (bool, error) {
	return v.V(i...)
}

func (v *ValidationFilter) Filter(i ...interface{}) (interface{}, error) {
	return v.F(i...)
}

func validate(chain Call, fn ...interface{}) (interface{}, error) {
	var err error
	var okay bool

	var V Validating
	if cn, ok := chain.(*chainNode); ok && cn.validator != nil {
		V, okay = cn.validator.(Validating)
	}

	if okay {
		//log.Printf("VALIDATE: %v",reflect.TypeOf(fn))''
		if V != nil {
			okay, err = V.Validate(fn...)
			if err == nil && !okay {
				err = ErrChainInvalidType
			}
		}
	} else if V, okay = chain.(Validating); okay {
		okay, err = V.Validate(fn...)
		if err == nil && !okay {
			err = ErrChainInvalidType
		}
	}

	if err == nil && okay {
		if F, ok := V.(Filtering); ok && F != nil {
			if FF, err := F.Filter(fn...); err != nil {
				return nil, err
			} else {
				return FF, err
			}
		}
		if len(fn) > 0 {
			return fn[0], err
		}
		return fn, err
	}
	return nil, err
}

type chainNode struct {
	funcs  []interface{}
	wait   *sync.WaitGroup
	before *chainNode
	after  *chainNode

	validator Validating
}

// Returns a new root callchain that has no validator
func New() Root {
	return &chainNode{
		funcs: make([]interface{}, 0, 1),
		wait:  &sync.WaitGroup{},
	}
}

// Returns a new root callchain that has a user supplied validator
// and (optionally) filter.
func NewValidating(validator Validating) Root {
	return &chainNode{
		funcs:     make([]interface{}, 0, 1),
		wait:      &sync.WaitGroup{},
		validator: validator,
	}
}

func (cn *chainNode) Validator() Validating {
	return cn.validator
}

func (cn *chainNode) SetValidator(v Validating) error {
	for n := cn.getFirst(); n != nil; n = n.getNext() {
		n.validator = v
	}
	return nil
}

func clone(old *chainNode) (n *chainNode) {
	n = &chainNode{
		funcs: make([]interface{}, 0, 1),
		wait:  &sync.WaitGroup{},
	}
	if old != nil {
		n.validator = old.validator
	}
	return
}

func (cn *chainNode) insertBefore() (n *chainNode) {
	n = clone(cn)
	if cn.before != nil {
		cn.before.after = n
		n.before = cn.before
	}
	cn.before = n
	n.after = cn
	return
}

func (cn *chainNode) insertAfter() (n *chainNode) {
	n = clone(cn)
	if cn.after != nil {
		cn.after.before = n
		n.after = cn.after
	}
	cn.after = n
	n.before = cn
	return
}

func (cn *chainNode) getFirst() (n *chainNode) {
	for n = cn; n.before != nil; n = n.before {
		// nop
	}
	return
}

func (cn *chainNode) Head() Predicate {
	return cn.getFirst()
}

func (cn *chainNode) getLast() (n *chainNode) {
	for n = cn; n.after != nil; n = n.after {
		// nop
	}
	return
}

func (cn *chainNode) Tail() Predicate {
	return cn.getLast()
}

func (cn *chainNode) getNext() (n *chainNode) {
	if cn != nil {
		n = cn.after
	}
	return
}

func (cn *chainNode) Before(fn ...interface{}) (Predicate, error) {
	n := cn.insertBefore()

	f, err := validate(n, fn...)
	if err == nil && f != nil {
		n.funcs = append(n.funcs, f)
	}
	return n, err
}

func (cn *chainNode) After(fn ...interface{}) (Predicate, error) {
	n := cn.insertAfter()
	f, err := validate(n, fn...)
	if err == nil && f != nil {
		n.funcs = append(n.funcs, f)
	}
	return n, err
}

func (cn *chainNode) First(fn ...interface{}) (Predicate, error) {
	n := cn.getFirst().insertBefore()
	f, err := validate(n, fn...)
	if err == nil && f != nil {
		n.funcs = append(n.funcs, f)
	}
	return n, err
}

func (cn *chainNode) Last(fn ...interface{}) (Predicate, error) {
	n := cn.getLast().insertAfter()
	f, err := validate(n, fn...)
	if err == nil && f != nil {
		n.funcs = append(n.funcs, f)
	}
	return n, err
}

func (cn *chainNode) Register(fn ...interface{}) (Predicate, error) {
	//log.Printf("REGISTER %v",fn)
	f, err := validate(cn, fn...)
	if err == nil && f != nil {
		cn.funcs = append(cn.funcs, f)
	}
	return cn, err
}

func (cn *chainNode) Waiter() (Waiter, error) {
	if cn.wait == nil {
		return nil, ErrChainNoWaiter
	}
	return cn.wait, nil
}

func (cn *chainNode) Wait() {
	if cn.wait != nil {
		cn.wait.Wait()
	}
}

type nullWaiterFunc func()

func (nullWaiterFunc) Wait() {}

// NullWaiter is a Waiter compatible interface that
// will always do nothing when Wait() is called on it.
var NullWaiter = nullWaiterFunc(func() {})

func addAll(n int, W ...*sync.WaitGroup) {
	for _, w := range W {
		w.Add(n)
	}
}

func doneAll(W ...*sync.WaitGroup) {
	for _, w := range W {
		w.Done()
	}
}

// returns the sync.WaitGroup pointer for a given call chain node
// or nil if there is none.
func WaitGroup(chain Call) (wg *sync.WaitGroup) {
	var W Waiter
	var err error

	if W, err = chain.Waiter(); err != nil {
		return
	}
	wg, _ = W.(*sync.WaitGroup)
	return
}

func (cn *chainNode) RunFiltered(filter func(interface{}, []interface{}) bool,
	via func(interface{}, []interface{}),
	args ...interface{}) {
	gSync := &sync.WaitGroup{}
	defer gSync.Wait()
	var chainWait Waiter = NullWaiter

	for n := range cn.IterateAll() {
		wg := WaitGroup(n)
		for fn := range n.Iterate(gSync) {
			if !filter(fn, args) {
				gSync.Done()
				wg.Done()
				continue
			}
			go func(f interface{}, oWait Waiter, iWait *sync.WaitGroup, a []interface{}) {
				defer gSync.Done()
				if iWait != nil {
					defer iWait.Done()
				}
				oWait.Wait()
				via(f, a)
			}(fn, chainWait, wg, args)
		}
		if wg != nil {
			chainWait = wg
		}
	}
}

func (cn *chainNode) Run(via func(interface{}, []interface{}), args ...interface{}) {
	filt := func(interface{}, []interface{}) bool {
		return true
	}
	cn.RunFiltered(filt, via, args...)
}

func (cn *chainNode) Iterate(W ...*sync.WaitGroup) <-chan interface{} {
	C := make(chan interface{}, len(cn.funcs))
	if cn.wait != nil {
		W = append(W, cn.wait)
	}
	if len(W) > 0 {
		addAll(1, W...)
		defer doneAll(W...)
	}
	go func(funcs []interface{}, c chan<- interface{}, waits []*sync.WaitGroup) {
		defer close(c)
		var fn interface{}
		for _, fn = range funcs {
			if len(waits) > 0 {
				addAll(1, waits...)
			}
			select {
			case c <- fn:
			case <-time.After(time.Duration(10) * time.Second):
				if len(waits) > 0 {
					doneAll(waits...)
				}
				return
			}
		}
	}(cn.funcs, C, W)
	return C
}

// Iterate over the entire callchain list starting with
// antecdent nodes. See Iterate() for an example of usage.
func (root *chainNode) IterateAll() <-chan Call {
	C := make(chan Call, 0)
	go func(cn *chainNode, c chan<- Call) {
		defer close(c)
		for cnext := cn.getNext(); cn != nil; cn = cnext {
			select {
			case c <- cn:
			case <-time.After(time.Duration(10) * time.Second):
				return
			}
		}
	}(root.getFirst(), C)
	return C
}