package execute

import (
	"encoding/json"
	"reflect"
	"fmt"
	"os"

)

func getOwnerID() string {
    hostname, _ := os.Hostname()

    return fmt.Sprintf("%s-%d", hostname, os.Getpid())
}

func decodeInput(input []byte, inputType reflect.Type) (reflect.Value, error) {
	inputValue := reflect.New(inputType)

	if err := json.Unmarshal(input, inputValue.Interface()); err != nil {
		return reflect.Value{}, err
	}

	return inputValue.Elem(), nil
}