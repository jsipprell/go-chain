package chain_test

import (
	"fmt"
	_ "log"
	"reflect"
	"testing"

	"github.com/jsipprell/go-chain"
)

type Printing interface {
	Println(...interface{})
}
type PrintingFunc func(...interface{})
type PrintFunc func(Printing)
type TestFunc func(*testing.T)
type TestVariadicFunc func(*testing.T, ...string)

type TestWrapper struct {
	Score int
	fp    reflect.Value
}

var (
	testChain chain.Root
)

func (pf PrintingFunc) Println(v ...interface{}) {
	pf(v...)
}

func (tw *TestWrapper) Call(in []reflect.Value) (out []reflect.Value) {
	in = append(in, reflect.Value{})
	copy(in[1:], in[0:])
	in[0] = reflect.ValueOf(tw.Score)
	tw.fp.Call(in)
	return
}

func initChain() {
	testChain = chain.NewTyped(PrintFunc(nil))
	pred, err := testChain.Register(func(p Printing) {
		p.Println("startup 1")
	})
	if err != nil {
		panic(err.Error())
	}
	pred, err = pred.Before(func(p Printing) {
		p.Println("before 1")
	})
	_, err = pred.Last(func(p Printing) {
		p.Println("very last")
	})
	if err != nil {
		panic(err.Error())
	}
	pred, err = pred.Before(func(p Printing) {
		p.Println("even more before 1")
	})
	if err != nil {
		panic(err.Error())
	}
	_, err = pred.After(func(p Printing) {
		p.Println("after even more before 1")
	})
	if err != nil {
		panic(err.Error())
	}
	_, err = pred.Register(func(p Printing) {
		p.Println("about the same time as even more before 1")
	})
	if err != nil {
		panic(err.Error())
	}
	_, err = pred.First(func(p Printing) {
		p.Println("very first")
	})
	if err != nil {
		panic(err.Error())
	}
}

func ExampleChain() {
	initChain()

	pf := PrintingFunc(func(v ...interface{}) {
		fmt.Println(v...)
	})
	testChain.Run(pf)
	// Output:
	// very first
	// even more before 1
	// about the same time as even more before 1
	// after even more before 1
	// before 1
	// startup 1
	// very last
}

func TestChainLen(t *testing.T) {
	initChain()

	if l := testChain.Len(); l != 7 {
		t.Fatalf("incorrect chain length, should be 7 instead of %d", l)
	}
	n := testChain.Middle()
	n.Before(func(p Printing) {
		p.Println("this is near the middle")
	})
	testChain.Run(PrintingFunc(func(v ...interface{}) {
		t.Log(v...)
	}))
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

func TestFilter1(t *testing.T) {
	validation := &chain.ValidationFilter{
		V: chain.ValidationFunc(func(i ...interface{}) (ok bool, err error) {
			ok = (len(i) == 2)
			if ok {
				_, ok = i[0].(int)
			}
			return
		}),
		F: chain.FilterFunc(func(i ...interface{}) (out interface{}, err error) {
			if len(i) == 2 {
				T := reflect.TypeOf(i[1])
				_ = T
				out = &TestWrapper{
					Score: i[0].(int),
					fp:    reflect.ValueOf(i[1]),
				}
				return
			}
			panic("failure")
		}),
	}

	filter := func(i interface{}, args []interface{}) bool {
		o := i.(*TestWrapper)
		t.Logf("TestWrapper %d", o.Score)
		return true
	}
	c := chain.NewValidating(validation)
	_, err := c.Register(1, func(x int) {
		t.Logf("#%d", x)
		if x != 1 {
			t.Fatal("incorrect argument")
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("start")
	c.RunFiltered(filter)
	t.Log("done")
}
