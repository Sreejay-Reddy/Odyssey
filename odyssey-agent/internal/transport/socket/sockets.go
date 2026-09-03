package socket

import(
	"net"
)

const(
	SubmitPath = "odyssey-submit.sock"
	AckPath = "odyssey-ack.sock"
	ResultPath = "odyssey-result.sock"
)


func createSocket(path string) (net.Conn, error){
	conn, err := net.Dial("unix", path)
	if err != nil{
		return nil, err
	}
	return conn, nil
}

func CreateSubmitSocket() (net.Conn, error) {
	return createSocket(SubmitPath)
}

func CreateAckSocket() (net.Conn, error) {
	return createSocket(AckPath)
}

func CreateResultSocket() (net.Conn, error) {
	return createSocket(ResultPath)
}