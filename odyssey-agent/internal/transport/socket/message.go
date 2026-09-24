package socket

import (
	"encoding/json"
)


const (
	ProtocolVersion uint16 = 1
)

type MessageType uint8
type ExecutionStatus uint8

const (
	MessageSubmit MessageType = 1
	MessageAck    MessageType = 2
	MessageResult MessageType = 3
)

const (
    StatusSuccess ExecutionStatus = 0
    StatusFailed  ExecutionStatus = 1
)

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
	BatchID    uint64
	Executions []ResultExecution
}

type Ack struct {
	Version   uint8
	BatchID   uint64
	SessionID [16]byte
	SDKID     [16]byte
}

type RegistryMessage struct {
	SDKID     [16]byte
	SessionID [16]byte
	Targets   []Target
}

type Target struct {
	TargetID uint32
	Name     string
}
