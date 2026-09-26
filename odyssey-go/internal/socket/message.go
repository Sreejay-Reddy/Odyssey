package socket

import "encoding/json"

const (
	ProtocolVersion uint16 = 1
)

type ExecutionStatus uint8

const (
    StatusSuccess ExecutionStatus = 0
    StatusFailed  ExecutionStatus = 1
)

type Execution struct {
	Key      string
	TargetID uint32
	Input    []byte
}

type CommandMessage struct {
	Version    uint16
	BatchID    uint64
	Executions []Execution
}

type ResultExecution struct {
	Key      string
	TargetID uint32
	ExecutionResult json.RawMessage
	Status   ExecutionStatus
}

type Result struct {
	Version    uint16
	SDKID      [16]byte
	SessionID  [16]byte
	Executions []ResultExecution
}