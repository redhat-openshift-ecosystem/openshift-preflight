# Errors

* Errors should have a custom variable if they are being used to control execution. Otherwise, create the error at the time of use. 

```
var ErrMyAwesomeError = errors.New("awesome error")
err := someFailingFunction()
if errors.Is(err, ErrMyAwesomeError) {
  // React to the error
}
```

If creating an error this way, it is recommended to add it to an `errors.go` file in the package that is throwing the error.

* Taken from [Working with Errors in Go 1.13](https://go.dev/blog/go1.13-errors):
> Wrap an error to expose it to callers. Do not wrap an error when doing so would expose implementation details.

When one does wrap an error, it should be wrapped as such:
```
fmt.Errorf("my additional context: %w", err)
```

# Logging

Logging of errors should be limited to only the upper-most layers. Otherwise, wrapped errors could be logged multiple times, causing clutter and limiting the usefulness of the logs.

If an error is logged, and then returned, one should either wrap the current error
with more context, and return that entire error, or log the wrapped error and then
return only the wrapper, without the current error.
