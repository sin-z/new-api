package common

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

// FlexibleUnixTime 兼容上游时间字段的多种 JSON 形态。
// 支持 Unix 秒数、数字字符串和 RFC3339 字符串，统一保存为 Unix 秒。
type FlexibleUnixTime int64

func (t *FlexibleUnixTime) UnmarshalJSON(data []byte) error {
	raw := strings.TrimSpace(string(data))
	if raw == "" || raw == "null" {
		*t = 0
		return nil
	}
	if strings.HasPrefix(raw, `"`) {
		var value string
		if err := json.Unmarshal(data, &value); err != nil {
			return err
		}
		return t.setFromString(value)
	}
	seconds, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return err
	}
	*t = FlexibleUnixTime(seconds)
	return nil
}

func (t *FlexibleUnixTime) setFromString(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		*t = 0
		return nil
	}
	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil {
		*t = FlexibleUnixTime(seconds)
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return err
	}
	*t = FlexibleUnixTime(parsed.Unix())
	return nil
}

// Unix 返回标准 Unix 秒数。
func (t FlexibleUnixTime) Unix() int64 {
	return int64(t)
}
