package salmon

import (
	"testing"
)

func TestUpdateProxyTrust(t *testing.T) {
	p := &Proxy{}
	u1 := &User{}
	u1.Trust = 1
	u2 := &User{}
	u2.Trust = 2
	a := NewProxyAssignments()
	a.Add(u1, p)
	a.Add(u2, p)

	// Proxy's trust level should be identical to minimum trust level of its
	// users.
	p.UpdateTrust(a)
	if p.Trust != 1 {
		t.Errorf("determined incorrect proxy trust level")
	}

	// When user gets promoted, the proxy's trust level should increase too.
	u1.Trust++
	p.UpdateTrust(a)
	if p.Trust != 2 {
		t.Errorf("determined incorrect proxy trust level")
	}

	u1.Trust++
	p.UpdateTrust(a)
	if p.Trust != 2 {
		t.Errorf("determined incorrect proxy trust level")
	}
}
