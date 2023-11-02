package internal

import (
	"encoding/json"
)

const DELIM_BYTE = 0

func Msg2Bytes(msg NetworkMessage) ([]byte, error) {
	s, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	pack := append(s, DELIM_BYTE)
	return pack, nil
}

func Bytes2Msg[T NetworkMessage](data []byte) (*T, error) {
	var u T
	unpack := data[0 : len(data)-1]
	err := json.Unmarshal(unpack, &u)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func Bytes2GenericMsg(data []byte) (map[string]any, error) {
	var u map[string]any
	unpack := data[0 : len(data)-1]
	err := json.Unmarshal(unpack, &u)
	if err != nil {
		return nil, err
	}
	return u, nil
}
