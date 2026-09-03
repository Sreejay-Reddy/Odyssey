package storage

import(
	"encoding/json"
)

type Execution struct {
	Key       string
	Target    string
	WorkerID string
	Status    string
	Attempts  int
	Input     json.RawMessage
	ExecutionResult json.RawMessage
}