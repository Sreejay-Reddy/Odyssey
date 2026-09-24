@0xf18dbdbd444ac10b;

using Go = import "/go.capnp";

$Go.package("protocol");
$Go.import("github.com/sreejay-reddy/odyssey/protocol/gen/go");

using Execution = import "execution.capnp";
using Registry = import "registry.capnp";


struct SubmitMessage {
    protocolVersion @0 :UInt16;
    batchID         @1 :UInt64;
    executions      @2 :List(Execution.Execution);
}

struct RegistryMessage {
	sdkID     @0    :Data;
	sessionID @1    :Data;
	targets   @2    :List(Registry.Target);
}

struct ResultMessage {
    protocolVersion @0 :UInt16;
    sdkID           @1 :Data;
    sessionID       @2 :Data;
    executions      @3 :List(Execution.ResultExecution);
}


