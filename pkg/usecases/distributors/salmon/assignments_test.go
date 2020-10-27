package salmon

import (
	"testing"
)

func TestGetUsersAndProxies(t *testing.T) {
	a := NewProxyAssignments()
	u1, _ := NewUser()
	u2, _ := NewUser()
	u3, _ := NewUser()
	p1 := &Proxy{}
	p2 := &Proxy{}

	a.Add(u1, p1)
	a.Add(u2, p1)

	users := a.GetUsers(p1)
	if len(users) != 2 {
		t.Fatalf("expected 2 but got %d users", len(users))
	}

	users = a.GetUsers(p2)
	if len(users) != 0 {
		t.Fatalf("expected 0 but got %d users", len(users))
	}

	proxies := a.GetProxies(u1)
	if len(proxies) != 1 {
		t.Fatalf("expected 1 but got %d proxies", len(proxies))
	}

	proxies = a.GetProxies(u3)
	if len(proxies) != 0 {
		t.Fatalf("expected 0 but got %d proxies", len(proxies))
	}
}

func TestRemoveProxies(t *testing.T) {
	a := NewProxyAssignments()
	u1, _ := NewUser()
	u2, _ := NewUser()
	p1 := &Proxy{}
	p2 := &Proxy{}

	a.Add(u1, p1)
	a.Add(u1, p2)
	a.Add(u2, p1)

	if len(a.GetProxies(u1)) != 2 {
		t.Fatalf("expected 2 but got %d proxies", len(a.GetProxies(u1)))
	}
	if len(a.GetProxies(u2)) != 1 {
		t.Fatalf("expected 1 but got %d proxies", len(a.GetProxies(u1)))
	}

	a.RemoveProxy(p1)

	if len(a.ProxyToUser) != 1 {
		t.Fatalf("expected 1 but got %d proxies", len(a.ProxyToUser))
	}
	if len(a.GetUsers(p1)) != 0 {
		t.Fatalf("expected 0 but got %d users", len(a.GetUsers(p1)))
	}
	if len(a.GetProxies(u1)) != 1 {
		t.Fatalf("expected 1 but got %d proxies", len(a.GetProxies(u1)))
	}
	if len(a.GetProxies(u2)) != 0 {
		t.Fatalf("expected 0 but got %d proxies", len(a.GetProxies(u1)))
	}
}
