package execute

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"

	"github.com/sreejay-reddy/odyssey/odyssey-go/internal/registry"
)

type Execution struct {
	registry *registry.Registry
}

func (e *Execution) Execute(ctx context.Context, key string, target string, input json.RawMessage) (any, bool, error) {
	registered, exists := e.registry.GetByName(target)

	if !exists {
		return nil, false, errors.New("target is not registered")
	}

	if len(input) == 0 {
		input = json.RawMessage(`{}`)
	}

	fnValue := reflect.ValueOf(registered.Fn)

	var response any

	inputJSON := input
	inputValue := reflect.New(registered.InputType)

	err := json.Unmarshal(inputJSON, inputValue.Interface())
	if err != nil {
		return nil, false, err
	}

	inputStruct := inputValue.Elem()

	results := fnValue.Call([]reflect.Value{
		reflect.ValueOf(ctx),
		inputStruct,
	})

	response = results[0].Interface()
	errValue := results[1]

	if !errValue.IsNil() {
		functionErr := errValue.Interface().(error)

		return nil, false, functionErr
	}
	

	return response, true, nil
}
