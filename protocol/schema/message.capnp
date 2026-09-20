@0xf18dbdbd444ac10b;

using Go = import "/go.capnp";

$Go.package("protocol");
$Go.import("github.com/sreejay-reddy/odyssey/protocol/gen/go");

using Execution = import "execution.capnp";
using Registry = import "registry.capnp";


struct SubmitMessage {
    protocolVersion @0 :UInt16;
    messageID       @1 :UInt64;
    batchID         @2 :UInt64;
    executions      @3 :List(Execution.Execution);
}

struct RegistryMessage {
	sdkID     @0    :Data;
	sessionID @1    :Data;
	targets   @2    :List(Registry.Target);
}


