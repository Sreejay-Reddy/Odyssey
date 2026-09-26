package execute

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"

	"github.com/sreejay-reddy/odyssey/odyssey-go/internal/registry"
)

type Execution struct {
	Registry *registry.Registry
}

func (e *Execution) Execute(ctx context.Context, key string, targetID uint32, input json.RawMessage) (json.RawMessage, error) {
	select {
    	case <-ctx.Done():
        	return nil, ctx.Err()
    	default:
    }
	registered, exists := e.Registry.GetByID(targetID)

	if !exists {
		return nil, errors.New("target is not registered")
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
		return nil, err
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

		return nil, functionErr
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}
	

	return responseJSON, nil
}
