// Package concrunner is a
package concrunner

import (
	"fmt"
	"reflect"
	"sync"
)

type NoResultFunc func()
type ErrorOnlyFunc func() error
type ResultFunc func() (interface{}, error)

// Contains a list of errors in the order in which the functions were scheduled
// This list also includes nil errors
type MultiError []error

type multiResult []interface{}

// Creates a new concrunner with max number of goroutines running at a given time
func New(max int) *concRunner {
	c := new(concRunner)
	c.sem = make(chan bool, max)
	c.me = []error{}
	return c
}

// Run these functions without combining any errors/results
// This uses the default value of a max of 10 goroutines running concurrently
func Run(fns ...NoResultFunc) {
	RunAndCombine(noResultToResult(fns)...)
}

// Run the functions and return a combined error.
// The error is nil only if all functions return a nil error
// Otherwise it returns a MultiError
// This uses the default value of a max of 10 goroutines running concurrently
func RunAndError(fns ...ErrorOnlyFunc) error {
	_, err := RunAndCombine(errorOnlyToResult(fns)...)
	return err
}

// Runs the functions and returns a list of results and a combined error.
// The error is nil only if all functions return a nil error
// Otherwise it returns a MultiError
// This uses the default value of a max of 10 goroutines running concurrently
func RunAndCombine(fns ...ResultFunc) ([]interface{}, error) {
	return runMax(5, fns...)
}


func runMax(max int, fns ...ResultFunc) ([]interface{}, error) {
	if len(fns) == 0 {
		return nil, nil
	}

	if len(fns) == 1 {
		res, err := fns[0]()
		if err != nil {
			return []interface{}{res}, MultiError{err}
		} else {
			return []interface{}{res}, nil
		}
	}

	pr := New(max)
	pr.RunAndCombine(fns...)
	return pr.Wait()
}

type concRunner struct {
	me      MultiError
	sem     chan bool
	mut     sync.Mutex
	done    bool
	results multiResult
}

var osErrorType = reflect.TypeOf((*error)(nil)).Elem()

// Run a function with the given args.
// This is useful when calling this from within a for loop when you want to use the looping variable
func (p *concRunner) RunWithArgs(fn interface{}, args ...interface{}) {
	v := reflect.ValueOf(fn)
	if v.Kind() != reflect.Func {
		panic("Fn needs to be a function")
	}
	argVals := make([]reflect.Value, len(args))
	for idx, arg := range args {
		argVals[idx] = reflect.ValueOf(arg)
	}
	// TODO: add combining of results here also
	p.RunAndCombine(func() (interface{}, error) {
		res := v.Call(argVals)
		if res == nil || len(res) == 0 {
			return nil, nil
		}
		errRetVal := res[len(res)-1]
		if errRetVal.Interface() == nil {
			return nil, nil
		}
		if errRetVal.Type().Implements(osErrorType) {
			return nil, errRetVal.Interface().(error)
		}
		return nil, nil
	})
}

// Run the given functions
func (p *concRunner) Run(fns ...NoResultFunc) {
	p.RunAndCombine(noResultToResult(fns)...)
}

// Run and combine the errors of the functions
func (p *concRunner) RunAndError(fns ...ErrorOnlyFunc) {
	p.RunAndCombine(errorOnlyToResult(fns)...)
}

// Run and combine the errors and the results of the function
func (p *concRunner) RunAndCombine(fns ...ResultFunc) {
	if len(fns) == 0 {
		return
	}
	errs := make([]error, len(fns))
	results := make([]interface{}, len(fns))
	p.mut.Lock()
	if p.done {
		p.mut.Unlock()
		panic("This runner has already been used and can't be reused")
	}
	low := len(p.me)
	p.me = append(p.me, errs...)
	p.results = append(p.results, results...)
	high := len(p.me)
	p.mut.Unlock()
	for i := low; i < high; i++ {
		p.sem <- true
		go func(idx int) {
			p.results[idx], p.me[idx] = fns[idx-low]()
			<-p.sem
		}(i)
	}
	return
}

// Waits for all functions to finish running and returns a list of results and a MultiError
// This error is nil only when all the errors are nil
func (p *concRunner) Wait() (multiResult, error) {
	p.mut.Lock()
	if p.done {
		p.mut.Unlock()
		return p.results, p.me.ReturnError()
	}
	p.done = true
	p.mut.Unlock()
	// Waiting for all requests to finish
	for i := 0; i < cap(p.sem); i++ {
		p.sem <- true
	}
	return p.results, p.me.ReturnError()
}

func (me MultiError) Error() string {
	numErrs := 0
	var err error
	for _, e := range me {
		if e != nil {
			err = e
			numErrs++
		}
	}
	if numErrs == 0 {
		return "No errors"
	} else if numErrs == 1 {
		return err.Error()
	} else {
		return fmt.Sprintf("%s and %d other errors", err.Error(), numErrs-1)
	}
}

// Returns true if the multierror has any error
func (me MultiError) HasError() bool {
	if len(me) == 0 {
		return false
	}
	for _, e := range me {
		if e != nil {
			return true
		}
	}
	return false
}

func (me MultiError) ReturnError() error {
	if me.HasError() {
		return me
	}
	return nil
}

func (mr multiResult) HasResults() bool {
	for _, r := range mr {
		if r != nil {
			return true
		}
	}
	return false
}

func noResultToResult(fns []NoResultFunc) []ResultFunc {
	resultFuncs := make([]ResultFunc, len(fns))
	for idx := range fns {
		newFun := fns[idx]
		resultFuncs[idx] = func() (interface{}, error) {
			newFun()
			return nil, nil
		}
	}
	return resultFuncs
}

func errorOnlyToResult(fns []ErrorOnlyFunc) []ResultFunc {
	resultFuncs := make([]ResultFunc, len(fns))
	for idx := range fns {
		newFun := fns[idx]
		resultFuncs[idx] = func() (interface{}, error) {
			return nil, newFun()
		}
	}
	return resultFuncs
}
