@0xd0fe7a49be37d4da;

using Go = import "/go.capnp";

$Go.package("protocol");
$Go.import("github.com/sreejay-reddy/odyssey/protocol/gen/go");


struct Execution {
	key         @0 :Text;
	targetID    @1 :UInt32;
	input       @2 :Data;
}