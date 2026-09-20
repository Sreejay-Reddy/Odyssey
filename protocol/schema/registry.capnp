@0xb1b306d693bbbef5;

using Go = import "/go.capnp";

$Go.package("protocol");
$Go.import("github.com/sreejay-reddy/odyssey/protocol/gen/go");


struct Target {
	targetID @0 :UInt32;
	name     @1 :Text;
    functionName @2 :Text;
    ttlMS    @3 :UInt32;
}