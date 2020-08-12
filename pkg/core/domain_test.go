package core

import (
	"testing"
)

func TestBlockedIn(t *testing.T) {
	r := ResourceBase{}
	l := &Location{"DE", 1234}

	if r.IsBlockedIn(l) {
		t.Error("Falsely labeled resource as blocked.")
	}

	l = &Location{"AT", 1234}
	if r.IsBlockedIn(l) {
		t.Error("Falsely labeled resource as blocked.")
	}
}
