package core

import (
	"fmt"
	"time"
)

// Dummy implements a simple Resource, which we use in unit tests.
type Dummy struct {
	ObjectId   Hashkey
	UniqueId   Hashkey
	ExpiryTime time.Duration
	test       *ResourceTest
}

func NewDummy(oid Hashkey, uid Hashkey) *Dummy {
	return &Dummy{
		ObjectId:   oid,
		UniqueId:   uid,
		test:       &ResourceTest{State: StateFunctional},
		ExpiryTime: time.Hour}
}
func (d *Dummy) Oid() Hashkey {
	return d.ObjectId
}
func (d *Dummy) Uid() Hashkey {
	return d.UniqueId
}
func (d *Dummy) String() string {
	return fmt.Sprintf("dummy-%d-%d", d.UniqueId, d.ObjectId)
}
func (d *Dummy) Type() string {
	return "dummy"
}
func (d *Dummy) SetType(rType string) {
}
func (d *Dummy) IsPublic() bool {
	return false
}
func (d *Dummy) Test() *ResourceTest {
	return d.test
}
func (d *Dummy) SetTest(t *ResourceTest) {
	d.test = t
}
func (d *Dummy) Expiry() time.Duration {
	return d.ExpiryTime
}
func (d *Dummy) IsValid() bool {
	return true
}
func (d *Dummy) BlockedIn() LocationSet {
	return make(LocationSet)
}
func (d *Dummy) SetBlockedIn(LocationSet) {
}
