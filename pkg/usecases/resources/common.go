package resources

import (
	"encoding/json"

	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/core"
)

const (
	Crc64Polynomial = 0x42F0E1EBA9EA3693

	ResourceTypeVanilla      = "vanilla"
	ResourceTypeObfs4        = "obfs4"
	ResourceTypeScrambleSuit = "scramblesuit"
)

var ResourceMap = map[string]func() interface{}{
	ResourceTypeObfs4:        func() interface{} { return NewTransport() },
	ResourceTypeScrambleSuit: func() interface{} { return NewTransport() },
	ResourceTypeVanilla:      func() interface{} { return NewBridge() },
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
