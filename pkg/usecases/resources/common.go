package resources

import (
	"encoding/json"

	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/core"
)

const (
	Crc64Polynomial = 0x42F0E1EBA9EA3693

	ResourceTypeVanilla      = "vanilla"
	ResourceTypeObfs2        = "obfs2"
	ResourceTypeObfs3        = "obfs3"
	ResourceTypeObfs4        = "obfs4"
	ResourceTypeScrambleSuit = "scramblesuit"
	ResourceTypeMeek         = "meek"
	ResourceTypeSnowflake    = "snowflake"
	ResourceTypeWebSocket    = "websocket"
	ResourceTypeFTE          = "fte"
	ResourceTypeHTTPT        = "httpt"
)

var ResourceMap = map[string]func() interface{}{
	ResourceTypeVanilla:      func() interface{} { return NewBridge() },
	ResourceTypeObfs2:        func() interface{} { return NewTransport() },
	ResourceTypeObfs3:        func() interface{} { return NewTransport() },
	ResourceTypeObfs4:        func() interface{} { return NewTransport() },
	ResourceTypeScrambleSuit: func() interface{} { return NewTransport() },
	ResourceTypeMeek:         func() interface{} { return NewTransport() },
	ResourceTypeSnowflake:    func() interface{} { return NewTransport() },
	ResourceTypeWebSocket:    func() interface{} { return NewTransport() },
	ResourceTypeFTE:          func() interface{} { return NewTransport() },
	ResourceTypeHTTPT:        func() interface{} { return NewTransport() },
}

type TmpHashringDiff struct {
	New     map[string][]json.RawMessage
	Changed map[string][]json.RawMessage
	Gone    map[string][]json.RawMessage
}

// UnmarshalTmpHashringDiff unmarshals the raw JSON messages in the given
// temporary hashring into the respective data structures.
func UnmarshalTmpHashringDiff(tmp *TmpHashringDiff) (*core.HashringDiff, error) {

	ret := core.NewHashringDiff()

	process := func(data map[string][]json.RawMessage) error {
		for k, vs := range data {
			for _, v := range vs {
				rStruct := ResourceMap[k]()
				if err := json.Unmarshal(v, rStruct); err != nil {
					return err
				}
				ret.New[k] = append(ret.New[k], rStruct.(core.Resource))
			}
		}
		return nil
	}

	if err := process(tmp.New); err != nil {
		return nil, err
	}
	if err := process(tmp.Changed); err != nil {
		return nil, err
	}
	if err := process(tmp.Gone); err != nil {
		return nil, err
	}

	return ret, nil
}
