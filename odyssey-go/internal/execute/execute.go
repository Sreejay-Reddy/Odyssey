package execute

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"

	"github.com/jackc/pgx/v5"
	"github.com/sreejay-reddy/odyssey/odyssey-go/internal/registry"
)

func Execute(ctx context.Context, conn *pgx.Conn, registry *registry.Registry, key string, target string, input json.RawMessage) (any, bool, error) {
	registered, exists := registry.GetByName(target)

	if !exists {
		return nil, false, errors.New("target is not registered")
	}

	found := true
	if len(input) == 0 {
		found = false
	}

	fnValue := reflect.ValueOf(registered.Fn)
	fnType := fnValue.Type()

	if fnType.NumOut() != 2 {
		return nil, false, errors.New(
				"registered function must return (response, error)",
		)
	}

	errorType := reflect.TypeOf((*error)(nil)).Elem()

	if !fnType.Out(1).Implements(errorType) {
		return nil, false, errors.New(
			"registered function must return (response, error)",
		)
	}

	responseType := fnType.Out(0)

	if responseType.Kind() != reflect.Struct {
		return nil, false, errors.New(
			"registered function response must be a struct",
		)
	}

	var response any
	if found {
		inputJSON := input

		if fnType.NumIn() != 2 {
			return nil, false, errors.New(
				"input exists but registered function does not accept input",
			)
		}

		inputType := fnType.In(1)

		if inputType.Kind() != reflect.Struct {
			return nil, false, errors.New(
				"registered function input must be a struct",
			)
		}

		inputValue := reflect.New(inputType)

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
	}

	if fnType.NumIn() != 1 {
        return nil, false, errors.New(
            "no input exists but registered function requires input",
        )
    }

	results := fnValue.Call([]reflect.Value{
		reflect.ValueOf(ctx),
	})

	response = results[0].Interface()
	errValue := results[1]

	if !errValue.IsNil() {
		functionErr := errValue.Interface().(error)

		return nil, false, functionErr
	}

	return response, true, nil
}
