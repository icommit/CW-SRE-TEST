// Tests in this package are expected to both pass and fail.
package core

import (
	"log"
	"testing"
)

func TestHttpState(t *testing.T) {
	C, err := ReadConf("../app.yaml")
	if err != nil {
		log.Fatal(err)
	}
	host := C.Handlers.TcpUrl
	port := C.Handlers.Port
	token := C.Handlers.Token
	timeout := C.Handlers.Timeout
	msg := C.Handlers.Msg
	i, _, _ := TcpState(host, port, token, msg, timeout)
	if !i {
		t.Errorf(
			"unexpected status: got (%v) want (%v)",
			false,
			true,
		)
	}
}

func TestTcpState(t *testing.T) {
	C, err := ReadConf("../app.yaml")
	if err != nil {
		log.Fatal(err)
	}
	token := C.Handlers.Token
	timeout := C.Handlers.Timeout
	msg := C.Handlers.Msg

	i, _, _ := HttpState(C.Handlers.HttpUrl, token, msg, timeout)
	if !i {
		t.Errorf(
			"unexpected status: got (%v) want (%v)",
			false,
			true,
		)
	}
}
