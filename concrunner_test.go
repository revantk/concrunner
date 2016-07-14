package concrunner

import (
	"errors"
	"runtime"
	"testing"
)

func TestParallelRunner(t *testing.T) {
	t.Logf("%d goroutines running", runtime.NumGoroutine())
	pr := New(20)
	n := 10000
	resChan := make(chan int)
	argCountChan := make(chan int)
	count, argCount := 0, 0
	go func() {
		for _ = range resChan {
			count += 1
		}
	}()

	go func() {
		for c := range argCountChan {
			argCount += c
		}
	}()

	for i := 0; i < n; i++ {
		pr.Run(func() {
			resChan <- 1
		})
		pr.RunWithArgs(func(i int) error {
			argCountChan <- i
			return errors.New("Error doing stuff")
		}, i)

		temp := i
		pr.RunAndCombine(func() (interface{}, error) {
			return 1, nil
		}, func() (interface{}, error) {
			return temp, nil
		})
	}

	t.Logf("%d goroutines running", runtime.NumGoroutine())
	results, err := pr.Wait()
	t.Logf("%d goroutines running", runtime.NumGoroutine())

	if err == nil {
		t.Errorf("Functions didn't return an error. this should return that")
	}

	close(resChan)
	close(argCountChan)
	if n != count {
		t.Errorf("All functions did not run, n = %d, cound = %d", n, count)
	}

	e := (n * (n - 1) / 2)
	if e != argCount {
		t.Errorf("Issue with run with args, expected %d, got %d", e, argCount)
	}

	if !results.HasResults() {
		t.Errorf("Functions should have returned results, but they haven't")
	}
	sum := 0
	for _, res := range results {
		if ires, ok := res.(int); ok {
			sum = sum + ires
		}
	}
	if sum != e+n {
		t.Errorf("The result aggregation is flawed")
	}
	t.Logf("%d goroutines running", runtime.NumGoroutine())
}

//TODO: add a benchmark