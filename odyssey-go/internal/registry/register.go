package registry

import (
	"errors"
	"reflect"
	"runtime"
)

type registered struct {
    target       string
    functionName string
    fn           any
    ttlMS        int
}

var registry = make(map[string]registered)

func Register(target string, fn any, ttlMS int) error {
    value := reflect.ValueOf(fn)

    if !value.IsValid() || value.Kind() != reflect.Func {
        return errors.New("fn must be a function")
    }

    if ttlMS <= 0 {
        return errors.New("ttlMS must be greater than zero")
    }

    if _, exists := registry[target]; exists {
        return errors.New("target already registered")
    }

    registry[target] = registered{
        target:       target,
        functionName: runtime.FuncForPC(value.Pointer()).Name(),
        fn:           fn,
        ttlMS:        ttlMS,
    }

    return nil
}

func Get(target string) (registered, bool) {
    value, exists := registry[target]
    return value, exists
}