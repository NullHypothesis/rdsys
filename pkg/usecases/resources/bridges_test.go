package resources

import (
	"net"
	"testing"
)

func TestHashFingerprint(t *testing.T) {
	orig := "FDCF0A662099B0EAFE97F9B4467A9149898805AE"
	expected := "9CB4AE64AFF3B9E6BB4F9DD4A5EE3B834A65EA0E"

	received, err := HashFingerprint(orig)
	if err != nil {
		t.Fatal(err)
	}

	if received != expected {
		t.Errorf("expected %s but got %s", expected, received)
	}

	_, err = HashFingerprint("foobar")
	if err == nil {
		t.Fatal("accepted invalid fingerprint")
	}
}

func TestPrintTorAddr(t *testing.T) {
	a := &IPAddr{}
	a.IP = net.ParseIP("1.2.3.4")
	if PrintTorAddr(a) != "1.2.3.4" {
		t.Errorf("failed to print IPv4 address correctly")
	}
	a.IP = net.ParseIP("2a0c:4d80:42:702::1")
	if PrintTorAddr(a) != "[2a0c:4d80:42:702::1]" {
		t.Errorf("failed to print IPv666666ess correctly")
	}
}
