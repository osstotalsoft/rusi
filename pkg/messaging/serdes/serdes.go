package serdes

import jsoniter "github.com/json-iterator/go"

func Marshal(data interface{}) ([]byte, error) {
	return jsoniter.Marshal(data)
}

func MarshalToString(data interface{}) (string, error) {
	return jsoniter.MarshalToString(data)
}

func Unmarshal(data []byte, v interface{}) error {
	return jsoniter.Unmarshal(data, v)
}

func UnmarshalFromString(str string, v interface{}) error {
	return jsoniter.UnmarshalFromString(str, v)
}
