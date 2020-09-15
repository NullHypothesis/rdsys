package core

import (
	"time"
)

// Dummy implements a simple Resource.
type Dummy struct {
	ObjectId   Hashkey
	UniqueId   Hashkey
	ExpiryTime time.Duration
}

func NewDummy(oid Hashkey, uid Hashkey) *Dummy {
	return &Dummy{ObjectId: oid, UniqueId: uid}
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
	return d.ExpiryTime
}
func (d *Dummy) IsValid() bool {
	return true
}
