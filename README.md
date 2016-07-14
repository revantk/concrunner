# concrunner

Package concrunner is a package for running a max number of functions concurrently. It runs those functions keeping goroutines limited to the max number and then waits for them to be done. It also combines the results and errors of those functions.

You can use the package level methods to run a bunch of functions concurrently or create a new concrunner and enqueue functions to it. (Eg. in a for loop)


## Usage
```
err := concrunner.RunAndError(func() error { 
                                //... Do something 
                                return nil
                                }, func() error {
                                // .. Do something else
                                return nil
                                })
          
cr := concrunner.New(10) // Run a max of 10 functions concurrently
cr.Run(func() {
    //Do something
    })
cr.RunAndError(func() error {
    err := doSomethingElse()
    return err
  })
cr.RunAndCombine(func() (interface{}, error) {
  // Do something 
  return result, err
})

for st := range structs {
  cr.RunWithArgs(func(val StructType) error {
    // Do stuff with val
    return err //Return optional error
  }, st)
}

results, err := cr.Wait()
// The resulting error will be nill if none of the functions returned an error 
// and of type MultiError if any of them did. 
// Results is just a list of results from each function
```
