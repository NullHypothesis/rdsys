package salmon

import (
	"testing"
	"time"
)

func TestUpdateUserTrust(t *testing.T) {
	u := &User{}
	u.Trust = -2

	u.LastPromoted = time.Now().UTC()
	u.UpdateTrust()
	if u.Trust != -2 {
		t.Errorf("incorrect user trust level")
	}

	// Ten seconds before midnight means no promotion.
	u.LastPromoted = time.Now().UTC().Add(-time.Hour*24*2 + time.Second*10)
	u.UpdateTrust()
	if u.Trust != -2 {
		t.Errorf("incorrect user trust level: %d", u.Trust)
	}

	// After 2^abs(-2 + 1) days, the user should be promoted to trust level -1.
	u.LastPromoted = time.Now().UTC().Add(-time.Hour*24*2 - time.Second*10)
	u.UpdateTrust()
	if u.Trust != -1 {
		t.Errorf("incorrect user trust level")
	}

	// After 2^abs(-1 + 1) days, the user should be promoted to trust level 0.
	u.LastPromoted = time.Now().UTC().Add(-time.Hour*24 - time.Second*10)
	u.UpdateTrust()
	if u.Trust != 0 {
		t.Errorf("incorrect user trust level")
	}

	// After 2^abs(0 + 1) days, the user should be promoted to trust level 1.
	u.LastPromoted = time.Now().UTC().Add(-time.Hour*24*2 - time.Second*10)
	u.UpdateTrust()
	if u.Trust != 1 {
		t.Errorf("incorrect user trust level")
	}

	// Ten seconds before midnight means no promotion.
	u.LastPromoted = time.Now().UTC().Add(-time.Hour*24*4 + time.Second*10)
	u.UpdateTrust()
	if u.Trust != 1 {
		t.Errorf("incorrect user trust level")
	}

	// After 2^abs(1 + 1) days, the user should be promoted to trust level 2.
	u.LastPromoted = time.Now().UTC().Add(-time.Hour*24*4 - time.Second*10)
	u.UpdateTrust()
	if u.Trust != 2 {
		t.Errorf("incorrect user trust level")
	}
}
