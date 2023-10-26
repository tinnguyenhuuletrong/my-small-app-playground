package internal_test

import (
	"reflect"
	"testing"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/my-go-p2p/internal"
)

func Test_Encode_Decode_NetworkMessage(t *testing.T) {
	msg := internal.BuildUDPDiscoveryMessage(internal.UDPDiscoveryBody{
		Ipv4: "127.0.0.1:5000",
	})

	data, err := internal.Msg2Bytes(msg)
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}

	new_msg, err := internal.Bytes2Msg[internal.UDPDiscoveryMessage](data)

	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}

	if !reflect.DeepEqual(msg, new_msg) {
		t.Logf("Objst not equal")
		t.Fail()
		return
	}

}
