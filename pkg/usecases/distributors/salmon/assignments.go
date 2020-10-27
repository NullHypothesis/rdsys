package salmon

import (
	"log"
	"sync"

	"gitlab.torproject.org/tpo/anti-censorship/rdsys/internal"
	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/core"
)

// ProxyAssignments keeps track of what proxies are assigned to what users.
type ProxyAssignments struct {
	sync.Mutex
	UserToProxy map[*User]*internal.Set
	ProxyToUser map[*Proxy]*internal.Set
}

// NewProxyAssignments creates and returns a new ProxyAssignments struct.
func NewProxyAssignments() *ProxyAssignments {
	a := &ProxyAssignments{}
	a.UserToProxy = make(map[*User]*internal.Set)
	a.ProxyToUser = make(map[*Proxy]*internal.Set)
	return a
}

// GetUsers returns a slice of all users that were assigned the given proxy.
func (a *ProxyAssignments) GetUsers(p *Proxy) []*User {
	a.Lock()
	defer a.Unlock()

	users := []*User{}
	s, exists := a.ProxyToUser[p]
	if !exists {
		return users
	}
	for user, _ := range s.Set {
		users = append(users, user.(*User))
	}
	return users
}

// GetProxies returns a slice of all resources that were assigned to the given
// user.
func (a *ProxyAssignments) GetProxies(u *User) []core.Resource {
	a.Lock()
	defer a.Unlock()

	proxies := []core.Resource{}
	s, exists := a.UserToProxy[u]
	if !exists {
		return proxies
	}
	for proxy, _ := range s.Set {
		proxies = append(proxies, proxy.(*Proxy))
	}
	return proxies
}

// AddAssignment adds a bi-directional assignment from user to/from proxy.
func (a *ProxyAssignments) Add(u *User, p *Proxy) {
	a.Lock()
	defer a.Unlock()

	set, exists := a.UserToProxy[u]
	if !exists {
		set = internal.NewSet()
	}
	set.Add(p)
	a.UserToProxy[u] = set

	set, exists = a.ProxyToUser[p]
	if !exists {
		set = internal.NewSet()
	}
	set.Add(u)
	a.ProxyToUser[p] = set
}

// RemoveProxy removes a proxy from our assignments.
func (a *ProxyAssignments) RemoveProxy(p *Proxy) {
	users := a.GetUsers(p)
	a.Lock()
	defer a.Unlock()

	if _, exists := a.ProxyToUser[p]; !exists {
		return
	}

	delete(a.ProxyToUser, p)

	// Remove the proxy for all users that were ever assigned the proxy.
	log.Printf("removing proxy for %d users", len(users))
	for _, user := range users {
		s, exists := a.UserToProxy[user]
		if !exists {
			log.Printf("Bug: Inconsistent proxy mapping.")
			continue
		}
		s.Remove(p)
	}
}
