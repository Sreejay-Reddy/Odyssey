package socket

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