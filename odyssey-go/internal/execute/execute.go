package execute

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"

	"github.com/jackc/pgx/v5"
	"github.com/sreejay-reddy/odyssey/odyssey-go/internal/registry"
)

func Execute(ctx context.Context, conn *pgx.Conn, key string, target string) (any, bool, error) {
	registered, exists := registry.Get(target)

	if !exists {
		return nil, false, errors.New("target is not registered")
	}

	e := execution{
		key: key,
		target: target,
		ttlMS: registered.TTLMS,
		conn: conn,
	}

	acquired, err := e.acquire(ctx)
	if err != nil {
		return nil, false, err
	}

	if !acquired{
		if e.metadata.status == "completed"{
			found, err := e.fetchResponse(ctx)
			if err != nil {
				return nil, false, err
			}

			if found {
				return e.metadata.response, false, nil
			}
		}

		return nil, false, nil
	}

	success, err := e.startExecution(ctx)
	if err != nil {
		return nil, false, err
	}

	if !success {
		return  nil, false, nil
	}

	found, err :=  e.fetchInput(ctx)
	if err != nil {
		return nil, false, err
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

	if found {
		inputJSON := e.input

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

		response := results[0].Interface()
		errValue := results[1]

		if !errValue.IsNil() {
			functionErr := errValue.Interface().(error)

			if _, abandonErr := e.abandon(ctx); abandonErr != nil {
				return nil, false, abandonErr
			}

			return nil, false, functionErr
		}

		e.metadata.response = response
	}

	if !found {
		if fnType.NumIn() != 1 {
        	return nil, false, errors.New(
            	"no input exists but registered function requires input",
        	)
    	}

		results := fnValue.Call([]reflect.Value{
			reflect.ValueOf(ctx),
		})

		response := results[0].Interface()
		errValue := results[1]

		if !errValue.IsNil() {
			functionErr := errValue.Interface().(error)

			if _, abandonErr := e.abandon(ctx); abandonErr != nil {
				return nil, false, abandonErr
			}

			return nil, false, functionErr
		}

		e.metadata.response = response
	}

	complete, err := e.complete(ctx)

	if err != nil {
		return nil, false, err
	}

	if !complete {
		return nil, false, errors.New(
			"Execution completed but could not be canonically finalized.",
		)
	}

	return e.metadata.response, true, nil
}
