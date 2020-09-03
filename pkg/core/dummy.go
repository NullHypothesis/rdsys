package core

import (
	"time"
)

// Dummy implements a simple Resource.
type Dummy struct {
	ObjectId Hashkey
	UniqueId Hashkey
}

func (d *Dummy) Oid() Hashkey {
	return d.ObjectId
}
func (d *Dummy) Uid() Hashkey {
	return d.UniqueId
}
func (d *Dummy) String() string {
	return "dummy"
}
func (d *Dummy) Name() string {
	return d.String()
}
func (d *Dummy) IsDepleted() bool {
	return false
}
func (d *Dummy) IsPublic() bool {
	return false
}
func (d *Dummy) GetState() int {
	return 1
}
func (d *Dummy) SetState(state int) {
}
func (d *Dummy) Expiry() time.Duration {
	return time.Duration(0)
}
