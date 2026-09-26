package workers

import (
	"encoding/json"

	"github.com/sreejay-reddy/odyssey/odyssey-go/internal/socket"
)

type Response struct {
	Response json.RawMessage
	err 	 error
	Key      string
	TargetID uint32
	Status   socket.ExecutionStatus
}