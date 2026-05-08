package main

// Package main provides an exception-style error handling system for the ya/ymake
// build system reimplementation. This replaces traditional Go error passing with
// a throw/catch pattern that allows cleaner, more linear code flow.
//
// The core design uses panic/recover internally: Throw() panics with an *Exception,
// and Try() catches those panics, converting them back to returnable *Exception values.
// This enables idiomatic "unwrap or die" patterns like:
//
//	data := Throw2(os.ReadFile(path))
//
// See STYLE.md for comprehensive usage guidelines and philosophical rationale.
import "fmt"

// Exception wraps an error for exception-based flow control. The error is evaluated
// lazily via the internal what() function, enabling deferred error construction.
// Type is primarily used with Throw(), Catch(), and Try() for panic/recover-based
// error handling that bypasses traditional Go error propagation.
type Exception struct {
	what func() error
}

func (e *Exception) Error() string {
	return e.what().Error()
}

func (e *Exception) Unwrap() error {
	return e.what()
}

func (e *Exception) throw() {
	panic(e)
}

// Catch conditionally invokes the callback function cb when the receiver is non-nil.
// This provides a fluent error handling pattern for optionally handling exceptions
// without breaking code flow. The callback receives the exception for inspection
// or deferred action. Does nothing if e == nil.
func (e *Exception) Catch(cb func(*Exception)) {
	if e != nil {
		cb(e)
	}
}

// AsError converts the Exception to the standard library error interface. Returns nil
// when the receiver is nil, otherwise returns the wrapped error. This enables
// compatibility with APIs expecting error types and facilitates integration with
// standard library error checking and error-handling utilities.
func (e *Exception) AsError() error {
	if e == nil {
		return nil
	}

	return e.what()
}

// New creates an Exception from the provided error. The error is wrapped and stored
// lazily, allowing deferred evaluation via the internal what() function. Returns
// *Exception for use with Throw(), Catch(), and Try(). Common entry point for
// converting traditional Go errors into the exception system.
func New(err error) *Exception {
	return &Exception{
		what: func() error {
			return err
		},
	}
}

// Fmt creates an Exception using fmt.Errorf-style formatting. Accepts a format string
// and variable arguments for formatted error messages. Internally constructs an
// error via fmt.Errorf and wraps it in an Exception. Returns *Exception for use
// with throw/catch patterns.
func Fmt(format string, args ...any) *Exception {
	return New(fmt.Errorf(format, args...))
}

// Throw is the core exception-raising primitive. Panics with an Exception wrapping
// the provided error if err is non-nil. Does nothing if err == nil. This is typically
// used at function boundaries where errors should propagate immediately rather than
// being passed up the call stack. Must be caught by Try() or will crash the program.
func Throw(err error) {
	if err != nil {
		New(err).throw()
	}
}

// Throw2 unwraps a two-value tuple (T, error) returning T on success or panicking
// on error. Generic type parameter T allows any return type. This enables concise
// "unwrap or die" syntax: data := Throw2(os.ReadFile(path)). Internally calls Throw
// on the error component. Panics if err != nil.
func Throw2[T any](val T, err error) T {
	Throw(err)

	return val
}

// Throw3 unwraps a three-value tuple (T1, T2, error) returning both values on success
// or panicking on error. Generic type parameters T1, T2 allow any return types.
// Extends Throw2 for functions returning two values plus an error. Internally
// calls Throw on the error component. Panics if err != nil.
func Throw3[T1, T2 any](v1 T1, v2 T2, err error) (T1, T2) {
	Throw(err)

	return v1, v2
}

// ThrowFmt combines formatted message creation with unconditional panic. Accepts
// format string and args for error formatting, constructs an Exception via Fmt,
// and immediately throws it. Used when errors are unrecoverable and the program
// should halt. Always panics regardless of condition.
func ThrowFmt(format string, args ...any) {
	Fmt(format, args...).throw()
}

// HTTPError is a typed exception payload that carries an HTTP status
// code alongside the message. Handlers use ThrowHTTP to raise 4xx
// client errors (invalid args, not-found, conflict); an outer Catch
// uses errors.As to distinguish them from unexpected panics (which
// default to 500).
type HTTPError struct {
	Status int
	Msg    string
}

func (e *HTTPError) Error() string {
	return e.Msg
}

// ThrowHTTP raises an HTTP-specific typed exception carrying a status code and message.
// Intended for HTTP handlers to signal client errors (4xx) while distinguishing them
// from unexpected panics (which default to 500). The status code and formatted message
// are wrapped in an HTTPError payload. Handlers should catch exceptions with Try()
// and use errors.As to distinguish HTTPError from other exceptions.
func ThrowHTTP(status int, format string, args ...any) {
	New(&HTTPError{Status: status, Msg: fmt.Sprintf(format, args...)}).throw()
}

// Try provides catch-all exception handling. Executes the provided callback cb and
// catches any *Exception panics raised within, converting them to returnable *Exception
// values. Non-Exception panics are re-raised unchanged. Returns nil on success, or the
// caught Exception on failure. Pattern: if exc := Try(func() { ... }); exc != nil { handle }.
func Try(cb func()) (err *Exception) {
	defer func() {
		if rec := recover(); rec != nil {
			if exc, ok := rec.(*Exception); ok {
				err = exc
			} else {
				panic(rec)
			}
		}
	}()

	cb()

	return nil
}
