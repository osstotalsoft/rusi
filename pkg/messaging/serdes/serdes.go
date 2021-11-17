package serdes

import (
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"rusi/pkg/messaging"
)

func Marshal(data interface{}) ([]byte, error) {
	return jsoniter.Marshal(data)
}

func Unmarshal(data []byte, v interface{}) error {
	return jsoniter.Unmarshal(data, v)
}

func MarshalMessageEnvelope(data *messaging.MessageEnvelope) ([]byte, error) {
	return Marshal(data)
}

func UnmarshalMessageEnvelope(data []byte) (messaging.MessageEnvelope, error) {
	env := internalMessageEnvelope{}
	env.MessageEnvelope.Headers = map[string]string{}

	if err := Unmarshal(data, &env); err != nil {
		return env.MessageEnvelope, err
	}

	for key, val := range env.Headers {
		if val != nil {
			env.MessageEnvelope.Headers[key] = fmt.Sprintf("%v", val)
		} else {
			env.MessageEnvelope.Headers[key] = ""
		}
	}
	return env.MessageEnvelope, nil
}

type internalMessageEnvelope struct {
	messaging.MessageEnvelope
	Headers map[string]interface{} `json:"headers"`
}
