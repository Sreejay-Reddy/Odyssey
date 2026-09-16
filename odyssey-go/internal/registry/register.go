package registry

import (
	"errors"
	"reflect"
	"runtime"
    "context"
    "sync/atomic"

	"github.com/sreejay-reddy/odyssey/odyssey-go/configutil"
)

type Registry struct {
	byID   map[uint32]*Registered
	byName map[string]*Registered
}

type Registered struct {
    Target       string
    TargetID     uint32
    FunctionName string
    Fn           any
    TTLMS        int64
    InputType    reflect.Type
}

var counter atomic.Uint32
var contextType = reflect.TypeOf((*context.Context)(nil)).Elem()

func New() *Registry {
	return &Registry{
		byID:   make(map[uint32]*Registered),
		byName: make(map[string]*Registered),
	}
}

func (r *Registry) Register(cfg configutil.Config, target string, fn any, ttlMS int64) error {
    value := reflect.ValueOf(fn)
    t := reflect.TypeOf(fn)

    if t.Kind() != reflect.Func {
        return errors.New("fn must be a function")
    }

    if t.NumIn() != 2 {
        return errors.New("function must accept context.Context and an input struct")
    }

    	if t.In(0) != contextType {
		return errors.New(
			"first argument must be context.Context",
		)
	}


    inputType := t.In(1)


    if ttlMS <= 0 {
        return errors.New("ttlMS must be greater than zero")
    }

	if inputType.Kind() != reflect.Struct {
		return errors.New("inputType must be a struct")
	}

    if _, exists := r.byName[target]; exists {
        return errors.New("target already registered")
    }

    if _, exists := cfg.Registry["default"]; !exists {
        if _, exists := cfg.Registry[target]; !exists {
            return errors.New("target does not exist in odyssey.yaml and no default is defined")
        }
    }

    id := counter.Add(1)

    registered := Registered{
        Target:       target,
        TargetID:     id,     
        FunctionName: runtime.FuncForPC(value.Pointer()).Name(),
        Fn:           fn,
        TTLMS:        ttlMS,
        InputType:    inputType,
    }

    r.byID[registered.TargetID] = &registered
	r.byName[registered.Target] = &registered

    return nil
}

func (r *Registry) GetByName(target string) (*Registered, bool) {
    value, exists := r.byName[target]
    return value, exists
}

func (r *Registry) GetByID(targetID uint32) (*Registered, bool) {
    value, exists := r.byID[targetID]
    return value, exists
}

