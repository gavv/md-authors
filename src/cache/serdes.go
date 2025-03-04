package cache

import (
	"encoding/base64"
	"encoding/json"
)

func Serialize(data any) string {
	if data == nil {
		return ""
	}
	b, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(b)
}

func Deserialize(str string, data any) {
	if str == "" {
		return
	}
	b, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return
	}
	json.Unmarshal(b, data)
}
