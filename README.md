richerrors
==========

Provides golang errors with stacktraces, and a few other features.
            
The package is largely based on http://github.com/go-errors/errors, with additional
inspiration from https://github.com/go-errgo/errgo and https://github.com/amattn/deeperror.

Installation
------------

    go get github.com/ansel1/richerrors
    
Features
--------

* New errors have a stacktrace captured where they are created
* Wrap existing errors with a stacktrace (captured where they are wrapped)

    ```go
    err := lib.Read()
    return richerrors.Wrap(err)  // no-op if err is already a RichError
    ```
        
* Allow golang idiom of comparing an err value to an exported value, using `Is()`

    ```go
    var ParseError = richerrors.New("Parse error")
    // ...
    return richerrors.Extend(ParseError) // captures a stacktrace here
    // ...
    richerrors.Is(err, ParseError)  // instead of err == ParseError
    ```
        
* Put a new message on an error, while still using `Is()` to compare to the original error

    ```go
    return richerrors.Extend(ParseError).WithMessage("Bad input")
    ```
        
* Use `Extend()` and `Is()` for hierarchies of errors

    ```go
    var ParseError = richerrors.New("Parse error")
    var InvalidCharSet = richerrors.Extend(ParseError).WithMessage("Invalid char set")
    var InvalidSyntax = richerrors.Extend(ParseError).WithMessage("Invalid syntax")
    
    func Parse(s string) error {
        return richerrors.Extend(InvalidSyntax).WithMessagef("Invalid char set: %s", "UTF-8")
    }
    
    func Check() {
        err := Parse("fields")
        richerrors.Is(err, ParseError) // yup
        richerrors.Is(err, InvalidCharSet) // yup
        richerrors.Is(err, InvalidSyntax) // nope
    }
    ```
        
* RichErrors have an HTTP status code, and there's a function to get the code for any error

    ```go
    richerrors.HTTPCode(errors.New("regular error")) // 500
    richerrors.HTTPCode(richerrors.New("rich error").WithHTTPCode(404)) // 404
    ```
        
* Functions for printing error details
 
    ```go
    err := richerrors.New("boom")
    m := richerrors.Stacktrace(err) // just the stacktrace
    m = richerrors.Details(err) // error message and stacktrace
    ```
    
Basic Usage
-----------

The package contains functions for creating richerrors, and turning `error` instances into
rich errors.  There are also utility functions which take `error` instances and do stuff
with them.  All functions work with regular `error` instances, but will return more information
if the instance implements RichError.

The utility functions look for interfaces, not concrete types, so you can use them with anything
that implements them: `Wrapper`, `Stacker`, and `HTTPCoder`.  Errors created with this package 
implement them all.

You can create new error types which pick up these implementations by embedding an instance of 
RichError in your error type.

Example:

```go
package main

import (
    "github.com/ansel1/richerrors"
    "errors"
)

var InvalidInputs = errors.New("Input is invalid")

func main() {
    // create a new error, with a stacktrace attached
    err := richerrors.New("bad stuff happened")
    
    // create a new error with format string, like fmt.Errorf
    err = richerrors.Errorf("bad input: %v", os.Args)
    
    // extend an error, capturing a fresh stacktrace from this callsite
    err = richerrors.Extend(InvalidInputs)
    
    // turn any error into a RichError.  The stacktrace will be captured here if the
    // error did not already have a stacktrace attached.  If it did, this call is a no-op
    err = richerrors.Wrap(err, 0)

    // override the original error's message
    err.WithMessagef("Input is invalid: %v", os.Args)
    
    // Use Is to compare errors against values, which is a common golang idiom
    richerrors.Is(err, InvalidInputs) // will be true
    
    // associated an http code
    err.WithHTTPCode(400)
    
    // Wrap converts any error to a RichError.  If the error was not
    // already a RichError, it captures a stacktrace
    perr := parser.Parse("blah")
    err = Wrap(perr, 0)
    
    // Get the original error back
    richerrors.Unwrap(err) == perr  // will be true
    
    // Print the error to a string, with the stacktrace, if it has one
    s := richerrors.Details(err)
    
    // Just print the stacktrace (empty string if err is not a RichError)
    s := richerrors.Stacktrace(err)

    // Get the location of the error (the first line in the stacktrace)
    // File will be "unknown" if the err is not a RichError
    file, line := richerrors.Location(err)
    
    // Get an HTTP status code for an error.  Defaults to 500.
    code := richerrors.HTTPCode(err)
    
}
```
    
See inline docs for more details.

License
-------

This package is licensed under the MIT license, see LICENSE.MIT for details.