package chain_test

import (
	"fmt"
	_ "log"
	"testing"

	"github.com/jsipprell/go-chain"
)

type TestFunc func(*testing.T)
type TestVariadicFunc func(*testing.T, ...string)

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

	testChain.Run()
	// Output:
	// very first
	// even more before 1
	// about the same time as even more before 1
	// after even more before 1
	// before 1
	// startup 1
	// very last
}

func TestTypedChain(t *testing.T) {
	c := chain.NewTyped(TestFunc(nil))
	_, err := c.Register(func(x *testing.T) { x.Log("success") })
	if err != nil {
		t.Fatal(err)
	}
	t.Log("start")
	c.Run(t)
	t.Log("done")
}

func TestVariadicTypedChain(t *testing.T) {
	c := chain.NewTyped(TestVariadicFunc(nil))
	pred, err := c.Register(func(x *testing.T, vals ...string) {
		for i, l := range vals {
			x.Logf("%d: %v", i+1, l)
		}
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Log("start")
	c.Run(t, "a", "b", "c", "d")
	if _, err := pred.After(func(x *testing.T) { x.Log("should never see this") }); err == nil {
		t.Fatal("expected exception didn't occur")
	} else {
		_, err = pred.After(func(x *testing.T, vals ...string) {
			x.Logf("%v", vals)
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	c.Run(t, "e", "f", "g", "h")
	t.Log("done")
}
