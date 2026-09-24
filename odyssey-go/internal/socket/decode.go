package socket

import (
	"github.com/sreejay-reddy/odyssey/protocol/gen/go"
	"capnproto.org/go/capnp/v3"
)

func DecodeCommand(msg *capnp.Message) (CommandMessage, error) {
	root, err := protocol.ReadRootSubmitMessage(msg)
	if err != nil {
		return CommandMessage{}, err
	}

	version := root.ProtocolVersion()
	batchid := root.BatchID()

	command := CommandMessage{
		Version: version,
		BatchID: batchid,
	}

	executions, err := root.Executions()
	if err != nil {
		return CommandMessage{}, err
	}

	executables := make([]Execution, 0, executions.Len())

	for i := range executions.Len(){
		execution := executions.At(i)

		key, err := execution.Key()
		if err != nil {
			return CommandMessage{}, err
		}

		targetID := execution.TargetID()
		input, err := execution.Input()
		if err != nil {
			return CommandMessage{}, err
		}

		executable := Execution{
			Key: key,
			TargetID: targetID,
			Input: input,
		}

		executables = append(executables, executable)
	}

	command.Executions = executables

	return command, nil
}