package internal

import "encoding/json"

func Msg2Bytes(msg NetworkMessage) ([]byte, error) {
	s, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func Bytes2Msg[T NetworkMessage](data []byte) (*T, error) {
	var u T
	err := json.Unmarshal(data, &u)
	if err != nil {
		return nil, err
	}
	return &u, nil
}
