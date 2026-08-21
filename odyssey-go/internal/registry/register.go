package registry

import (
	"errors"
	"reflect"
	"runtime"

	"github.com/sreejay-reddy/odyssey/odyssey-go/configutil"
)

type Registered struct {
    Target       string
    FunctionName string
    Fn           any
    TTLMS        int64
}
var registry = make(map[string]Registered)

func Register(cfg configutil.Config, target string, fn any, ttlMS int64) error {
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

    if _, exists := cfg.Registry["default"]; !exists {
        if _, exists := cfg.Registry[target]; !exists {
            return errors.New("target does not exist in odyssey.yaml and no default is defined")
        }
    }

    registry[target] = Registered{
        Target:       target,
        FunctionName: runtime.FuncForPC(value.Pointer()).Name(),
        Fn:           fn,
        TTLMS:        ttlMS,
    }

    return nil
}

func Get(target string) (Registered, bool) {
    value, exists := registry[target]
    return value, exists
}

func Reset() {
	registry = make(map[string]Registered)
}
