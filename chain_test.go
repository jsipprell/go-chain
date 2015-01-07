package chain_test

import (
	"fmt"
	_ "log"
	_ "testing"

	"github.com/jsipprell/go-chain"
)

var (
	testChain chain.Root
)

func initChain() {
	testChain = chain.New()
	pred, err := testChain.Register(func() {
		fmt.Println("startup 1")
	})
	if err != nil {
		panic(err.Error())
	}
	pred, err = pred.Before(func() {
		fmt.Println("before 1")
	})
	_, err = pred.Last(func() {
		fmt.Println("very last")
	})
	if err != nil {
		panic(err.Error())
	}
	pred, err = pred.Before(func() {
		fmt.Println("even more before 1")
	})
	if err != nil {
		panic(err.Error())
	}
	_, err = pred.After(func() {
		fmt.Println("after even more before 1")
	})
	if err != nil {
		panic(err.Error())
	}
	_, err = pred.Register(func() {
		fmt.Println("about the same time as even more before 1")
	})
	if err != nil {
		panic(err.Error())
	}
	_, err = pred.First(func() {
		fmt.Println("very first")
	})
	if err != nil {
		panic(err.Error())
	}
}

func ExampleChain() {
	initChain()

	r := func(i interface{}, args []interface{}) {
		_ = args
		i.(func())()
	}
	testChain.Run(r)
	// Output:
	// very first
	// even more before 1
	// about the same time as even more before 1
	// after even more before 1
	// before 1
	// startup 1
	// very last
}
