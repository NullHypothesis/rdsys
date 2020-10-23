package salmon

import (
	"fmt"
	"math/rand"
	"net"
	"strings"
	"testing"
	"time"

	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/core"
	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/usecases/resources"
)

func TestTokenCache(t *testing.T) {
	salmon := NewSalmonDistributor()
	u := &User{SecretId: "foo"}
	salmon.Users[u.SecretId] = u

	// Banned users are not allowed to invite.
	u.Banned = true
	_, err := salmon.CreateInvite(u.SecretId)
	if err == nil {
		t.Errorf("banned users are not allowed to invite")
	}
	u.Banned = false

	// New users are not allowed to invite.
	_, err = salmon.CreateInvite(u.SecretId)
	if err == nil {
		t.Errorf("user should not yet be allowed to issue invites")
	}

	u.Trust = MaxTrustLevel
	token, err := salmon.CreateInvite(u.SecretId)
	if err != nil {
		t.Errorf("failed to create invite token: %s", err)
	}
	if token == "" {
		t.Errorf("got empty invite token")
	}

	// We should now have a new entry in our token cache.
	if len(salmon.TokenCache) != 1 {
		t.Errorf("new token was not cached")
	}

	// We should now be able to successfully redeem our token.
	_, err = salmon.RedeemInvite(token)
	if err != nil {
		t.Errorf("failed to redeem invite: %s", err)
	}

	// Our token cache should be empty again.
	if len(salmon.TokenCache) != 0 {
		t.Errorf("token was not deleted upon successful redemption")
	}

	// It must not be possible to redeem a token twice.
	_, err = salmon.RedeemInvite(token)
	if err == nil {
		t.Errorf("must not be possible to redeem a token twice")
	}

	// It also must not be possible to redeem a token that doesn't exist in the
	// cache.
	_, err = salmon.RedeemInvite("ThisTokenDoesNotExist")
	if err == nil {
		t.Errorf("must not be possible to redeem token that's not cached")
	}

	// Create another invite, which we'll let expire.
	token, err = salmon.CreateInvite(u.SecretId)
	if err != nil {
		t.Errorf("failed to create invite token: %s", err)
	}
	metaInfo, _ := salmon.TokenCache[token]
	now := time.Now().UTC()
	metaInfo.IssueTime = now.Add(-InvitationTokenExpiry - time.Minute)

	// An expired token must not be redeemable.
	_, err = salmon.RedeemInvite(token)
	if err == nil {
		t.Errorf("expired token must not be redeemable")
	}
}

func TestPruneTokenCache(t *testing.T) {
	salmon := NewSalmonDistributor()
	expiredTime := time.Now().UTC().Add(-InvitationTokenExpiry - time.Minute)
	salmon.TokenCache["DummyToken"] = &TokenMetaInfo{"foo", expiredTime}
	if len(salmon.TokenCache) != 1 {
		t.Errorf("failed to add expired token to cache")
	}

	salmon.pruneTokenCache()
	if len(salmon.TokenCache) != 0 {
		t.Errorf("failed to prune token cache")
	}
}

func TestProcessDiff(t *testing.T) {
	salmon := NewSalmonDistributor()
	a := NewProxyAssignments()
	salmon.Assignments = a

	// Create a user, a proxy, and assign the proxy to the user.
	u, _ := salmon.AddUser(0, nil)
	p := &Proxy{Resource: resources.NewTransport()}
	a.Add(u, p)

	queue := core.ResourceQueue{p}
	salmon.AssignedProxies[resources.ResourceTypeObfs4] = queue

	diff := core.NewResourceDiff()
	// Create a new copy of the proxy and mark it as blocked.
	pNew := &Proxy{Resource: resources.NewTransport()}
	pNew.SetBlockedIn(core.LocationSet{"no": true})
	diff.Changed = core.ResourceMap{resources.ResourceTypeObfs4: core.ResourceQueue{pNew}}
	salmon.processDiff(diff)

	// User should now have a blocking event.
	if len(u.InnocencePs) == 0 {
		t.Errorf("expected 1 blocking event but got %d", len(u.InnocencePs))
	}
	// User should now also be banned.
	if !u.Banned {
		t.Errorf("failed to ban user")
	}
}

// genResourceMap generates a resource map consisting of the given number of
// obfs4 bridges.
func genResourceMap(num int) core.ResourceMap {

	rm := make(core.ResourceMap)
	q := core.ResourceQueue{}

	for i := 0; i < num; i++ {
		r := resources.NewTransport()
		r.RType = resources.ResourceTypeObfs4

		var octets []string
		for octet := 0; octet < 4; octet++ {
			octets = append(octets, fmt.Sprintf("%d", rand.Intn(256)))
		}
		addrStr := strings.Join(octets, ".")
		ip := net.ParseIP(addrStr)
		ipaddr := net.IPAddr{IP: ip}
		r.Address = resources.IPAddr{IPAddr: ipaddr}
		r.Port = uint16(rand.Intn(65536))
		r.Parameters["iat-mode"] = "0"
		// No need to have "real-looking" certificates here.
		r.Parameters["cert"] = "foo"
		q.Enqueue(&Proxy{Resource: r})
	}
	rm[resources.ResourceTypeObfs4] = q

	return rm
}

func TestUserFlow(t *testing.T) {

	salmon := NewSalmonDistributor()
	salmon.cfg.Distributors.Salmon.Resources = []string{resources.ResourceTypeObfs4}
	salmon.UnassignedProxies = genResourceMap(100)

	admin, err := salmon.AddUser(UntouchableTrustLevel, nil)
	if err != nil {
		t.Fatalf("Failed to create new admin user: %s", err)
	}

	token, err := salmon.CreateInvite(admin.SecretId)
	if err != nil {
		t.Fatalf("Failed to create Salmon invite: %s", err)
	}
	userId, err := salmon.RedeemInvite(token)
	if err != nil {
		t.Fatalf("Failed to redeem Salmon invite: %s", err)
	}

	proxies, err := salmon.GetProxies(userId, "obfs4")
	if err != nil {
		t.Fatalf("Failed to get proxies: %s", err)
	}

	if len(proxies) == 0 {
		t.Fatalf("Got no proxies.")
	}
}
