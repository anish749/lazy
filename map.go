package lazy

import "reflect"

// isNilError checks if an error value is nil, handling both interface nil
// and typed nil (e.g., (*MyError)(nil)) correctly.
func isNilError[E error](err E) bool {
	v := reflect.ValueOf(err)
	if !v.IsValid() {
		return true
	}
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func:
		return v.IsNil()
	}
	return false
}

// Map applies a transformation function to the lazy value and returns a new lazy with the result.
// The function transforms the value from type T to type U. The transformation is lazy -
// it only executes when Get() is called on the resulting Lazy.
//
// If the original lazy has an error, that error is propagated to the new lazy
// without calling the transformation function.
//
// Example:
//
//	strLazy := NewLazy(func() (string, error) { return "hello", nil })
//	intLazy := Map(strLazy, func(s string) int { return len(s) })
//	length, err := intLazy.Get() // returns 5, nil
func Map[T any, U any, E error](l *Lazy[T, E], fn func(T) U) *Lazy[U, E] {
	return NewLazy(func() (U, E) {
		value, err := l.Get()
		if !isNilError(err) {
			var zero U
			return zero, err
		}
		var zeroErr E
		return fn(value), zeroErr
	})
}

// FlatMap applies a function that returns a Lazy to the lazy value and flattens the result.
// This is useful when the transformation itself produces a lazy value, avoiding
// nested Lazy types (Lazy[Lazy[U, E], E]).
//
// This is the monadic bind operation for Lazy, enabling composition of lazy computations
// where each step depends on the result of the previous one.
//
// If the original lazy has an error, that error is propagated to the new lazy
// without calling the transformation function.
//
// Example:
//
//	userIdLazy := NewLazy(func() (int, error) { return 42, nil })
//	userLazy := FlatMap(userIdLazy, func(id int) *Lazy[User, error] {
//	    return NewLazy(func() (User, error) { return fetchUser(id) })
//	})
//	user, err := userLazy.Get()
func FlatMap[T any, U any, E error](l *Lazy[T, E], fn func(T) *Lazy[U, E]) *Lazy[U, E] {
	return NewLazy(func() (U, E) {
		value, err := l.Get()
		if !isNilError(err) {
			var zero U
			return zero, err
		}
		return fn(value).Get()
	})
}
