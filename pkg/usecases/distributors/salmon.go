// This file implements the core logic of the Salmon proxy distribution system.
// The theory behind Salmon is presented in the following PETS'16 paper:
// https://censorbib.nymity.ch/#Douglas2016a
// Note that this file does *not* implement any user-facing code.
// TODO: We may want to move this code to its separate package.
package distributors

import (
	"errors"
	"log"
	"math"
	"sync"
	"time"

	"gitlab.torproject.org/tpo/anti-censorship/rdsys/internal"
	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/core"
	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/delivery"
	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/delivery/mechanisms"
	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/usecases/resources"
)

const (
	SalmonDistName = "salmon"
	// The Salmon paper calls this threshold "T".  Simulation results suggest T
	// = 1/3: <https://censorbib.nymity.ch/pdf/Douglas2016a.pdf#page=7>
	MaxSuspicion = 0.333
	// MaxTrustLevel represents the maximum trust level that a user can get
	// promoted to.  The paper refers to the maximum trust level as "L" and
	// argues that six is a good compromise:
	// <https://censorbib.nymity.ch/pdf/Douglas2016a.pdf#page=4>
	MaxTrustLevel = Trust(6)
	// A user can get UntouchableTrustLevel by being invited directly by us.
	UntouchableTrustLevel = Trust(MaxTrustLevel + 1)
	MaxClients            = 10
	SalmonTickerInterval  = time.Hour * 24
	// Number of bytes.
	InvitationTokenLength = 20
	UserSecretIdLength    = 20
	InvitationTokenExpiry = time.Hour * 24 * 7
	NumProxiesPerUser     = 3 // TODO: This should be configurable.
	TokenCacheFile        = "token-cache.bin"
	UsersFile             = "users.bin"
)

// SalmonDistributor contains all the context that the distributor needs to
// run.
type SalmonDistributor struct {
	ipc      delivery.Mechanism
	cfg      *internal.Config
	wg       sync.WaitGroup
	shutdown chan bool

	TokenCache        map[string]*TokenMetaInfo
	TokenCacheMutex   sync.Mutex
	Users             map[string]*User
	AssignedProxies   core.ResourceMap
	UnassignedProxies core.ResourceMap
}

// Trust represents the level of trust we have for a user or proxy.
type Trust int

// TokenMetaInfo represents meta information that's associated with an
// invitation token.  In particular, we keep track of when an invitation token
// was issued and who issued the token.
type TokenMetaInfo struct {
	SecretInviterId string
	IssueTime       time.Time
}

// User represents a user account.
type User struct {
	SecretId string
	Banned   bool
	// The probability of the client *not* being an agent is the product of the
	// probabilities of innocence of each proxy blocking event that the client
	// was involved in.  The complement of this probability is the client's
	// suspicion.  We permanently ban client whose suspicion meets or exceeds
	// our suspicion threshold.
	InnocencePs []float64
	Trust       Trust
	InvitedBy   *User `json:"-"` // We have to omit this field to prevent cycles.
	Invited     []*User
	Proxies     []core.Resource
	// The last time the user got promoted to a higher trust level.
	LastPromoted time.Time
}

// Proxy represents a circumvention proxy that's handed out to users.
type Proxy struct {
	core.Resource
	ReservedFor int
	Users       []*User
	Trust       Trust
}

// NewSalmonDistributor allocates and returns a new distributor object.
func NewSalmonDistributor() *SalmonDistributor {
	salmon := &SalmonDistributor{}
	salmon.TokenCache = make(map[string]*TokenMetaInfo)
	salmon.Users = make(map[string]*User)
	salmon.AssignedProxies = make(core.ResourceMap)
	salmon.UnassignedProxies = make(core.ResourceMap)
	return salmon
}

func (s *SalmonDistributor) NewUser(trust Trust, inviterId string) (*User, error) {
	secretId, err := internal.GetRandBase32(UserSecretIdLength)
	if err != nil {
		return nil, err
	}

	u := &User{}
	inviter, exists := s.Users[inviterId]
	if !exists {
		log.Println("Creating new server admin account.")
		inviter = nil
	} else {
		inviter.Invited = append(inviter.Invited, u)
	}

	u.InvitedBy = inviter
	u.Trust = trust
	u.LastPromoted = time.Now().UTC()
	u.SecretId = secretId

	s.Users[secretId] = u
	log.Printf("Created new user with secret ID %q.", secretId)

	return u, nil
}

// convertToProxies converts the Resource elements in the given ResourceDiff to
// Proxy elements, which extend Resources.
func convertToProxies(diff *core.ResourceDiff) {

	convert := func(m core.ResourceMap) {
		for _, rQueue := range m {
			for i, r := range rQueue {
				rQueue[i] = &Proxy{Resource: r}
			}
		}
	}
	convert(diff.New)
	convert(diff.Changed)
	convert(diff.Gone)
}

// processDiff takes as input a resource diff and feeds it into Salmon's
// existing set of resources.
// * New proxies are added to UnassignedProxies.
// *
func (s *SalmonDistributor) processDiff(diff *core.ResourceDiff) {

	convertToProxies(diff)
	for rType, rQueue := range diff.Changed {
		for _, r1 := range rQueue {
			// Is the given resource blocked in a new place?
			q, exists := s.AssignedProxies[rType]
			if !exists {
				continue
			}
			r2, err := q.Search(r1.Uid())
			if err == nil {
				if r1.BlockedIn().HasLocationsNotIn(r2.BlockedIn()) {
					r2.(*Proxy).SetBlocked()
				}
			}
		}
	}

	s.UnassignedProxies.ApplyDiff(diff)
	// New proxies only belong in UnassignedProxies.
	diff.New = nil
	s.AssignedProxies.ApplyDiff(diff)
	log.Printf("Unassigned proxies: %s; assigned proxies: %s",
		s.UnassignedProxies, s.AssignedProxies)

	// Potential problems:
	// 0. How should we handle new proxies that are blocked already?
	// 1. Gone proxies: Users are assigned to proxies that no longer exist.
	//    Maybe add a destructor to proxies?
}

// Init initialises the given Salmon distributor.
func (s *SalmonDistributor) Init(cfg *internal.Config) {
	log.Printf("Initialising %s distributor.", SalmonDistName)

	s.NewUser(UntouchableTrustLevel, "")
	s.cfg = cfg
	s.shutdown = make(chan bool)

	log.Printf("Initialising resource stream.")
	s.ipc = mechanisms.NewHttpsIpc("http://" + cfg.Backend.ApiAddress + cfg.Backend.ResourceStreamEndpoint)
	rStream := make(chan *core.ResourceDiff)
	req := core.ResourceRequest{
		RequestOrigin: SalmonDistName,
		ResourceTypes: s.cfg.Distributors.Salmon.Resources,
		BearerToken:   s.cfg.Backend.ApiTokens[SalmonDistName],
		Receiver:      rStream,
	}
	s.ipc.StartStream(&req)

	s.wg.Add(1)
	go s.Housekeeping(rStream)

	s.TokenCacheMutex.Lock()
	defer s.TokenCacheMutex.Unlock()
	err := internal.Deserialise(cfg.Distributors.Salmon.WorkingDirectory+TokenCacheFile, &s.TokenCache)
	if err != nil {
		log.Printf("Warning: Failed to deserialise token cache: %s", err)
	}
}

// Shutdown shuts down the given Salmon distributor.
func (s *SalmonDistributor) Shutdown() {

	// Write our token cache to disk so it can persist across restarts.
	s.TokenCacheMutex.Lock()
	defer s.TokenCacheMutex.Unlock()
	err := internal.Serialise(s.cfg.Distributors.Salmon.WorkingDirectory+TokenCacheFile, s.TokenCache)
	if err != nil {
		log.Printf("Warning: Failed to serialise token cache: %s", err)
	}

	// Signal to housekeeping that it's time to stop.
	close(s.shutdown)
	s.wg.Wait()
}

// IsDepleted returns true if the proxy reached its capacity and can no longer
// accommodate new users.
func (p *Proxy) IsDepleted() bool {
	return len(p.Users) >= MaxClients
}

// Don't call this function directly.  Call findProxies instead.
func findAssignedProxies(inviter *User) []core.Resource {

	var proxies []core.Resource

	// Do the given user's proxies have any free slots?
	for _, proxy := range inviter.Proxies {
		if proxy.IsDepleted() {
			continue
		}
		proxies = append(proxies, proxy)
		if len(proxies) >= NumProxiesPerUser {
			return proxies
		}
	}

	// If we don't have enough proxies yet, we are going to recursively
	// traverse invitation tree to find already-assigned, non-depleted proxies.
	for _, invitee := range inviter.Invited {
		ps := findAssignedProxies(invitee)
		proxies = append(proxies, ps...)
		if len(proxies) >= NumProxiesPerUser {
			return proxies[:NumProxiesPerUser]
		}
	}

	return proxies
}

func (s *SalmonDistributor) findProxies(invitee *User, rType string) []core.Resource {

	if invitee == nil {
		return nil
	}

	var proxies []core.Resource
	// People who registered and admin friends don't have an inviter.
	if invitee.InvitedBy != nil {
		proxies := findAssignedProxies(invitee.InvitedBy)
		if len(proxies) == NumProxiesPerUser {
			log.Printf("Returning %d proxies to user.", len(proxies))
			return proxies
		}
	}

	// Take some of our unassigned proxies and allocate them for the given user
	// graph, T(u).
	numRemaining := NumProxiesPerUser - len(proxies)
	if len(s.UnassignedProxies) < numRemaining {
		numRemaining = len(s.UnassignedProxies)
	}
	newProxies := s.UnassignedProxies[rType][:numRemaining]
	s.UnassignedProxies[rType] = s.UnassignedProxies[rType][numRemaining:]
	log.Printf("Not enough assigned proxies; allocated %d unassigned proxies, %d remaining",
		len(newProxies), len(s.UnassignedProxies))

	for _, p := range newProxies {
		s.AssignedProxies[rType] = append(s.AssignedProxies[rType], p)
		invitee.Proxies = append(invitee.Proxies, p)
		proxies = append(proxies, p)
	}

	return proxies
}

// GetProxies attempts to return proxies for the given user.
func (s *SalmonDistributor) GetProxies(secretId string, rType string) ([]core.Resource, error) {

	user, exists := s.Users[secretId]
	if !exists {
		return nil, errors.New("user ID does not exists")
	}

	if _, exists := resources.ResourceMap[rType]; !exists {
		return nil, errors.New("requested resource type does not exist")
	}

	// Is Salmon handing out the resources that is requested?
	isSupported := false
	for _, supportedType := range s.cfg.Distributors.Salmon.Resources {
		if rType == supportedType {
			isSupported = true
		}
	}
	if !isSupported {
		return nil, errors.New("requested resource type not supported")
	}

	if user.Banned {
		return nil, errors.New("user is blocked and therefore unable to get proxies")
	}

	// Does the user already have assigned proxies?
	if len(user.Proxies) > 0 {
		return user.Proxies, nil
	}

	return s.findProxies(user, rType), nil
}

// Housekeeping keeps track of periodic tasks.
func (s *SalmonDistributor) Housekeeping(rStream chan *core.ResourceDiff) {

	defer s.wg.Done()
	defer close(rStream)
	defer s.ipc.StopStream()
	ticker := time.NewTicker(SalmonTickerInterval)
	defer ticker.Stop()

	for {
		select {
		case diff := <-rStream:
			s.processDiff(diff)
		case <-s.shutdown:
			log.Printf("Shutting down housekeeping.")
			return
		case <-ticker.C:
			// Iterate over all users and proxies and update their trust levels if
			// necessary.
			log.Printf("Updating trust levels of %d users.", len(s.Users))
			for _, user := range s.Users {
				user.UpdateTrust()
			}
			log.Printf("Updating trust levels of %d proxies.", len(s.AssignedProxies))
			for _, proxies := range s.AssignedProxies {
				for _, proxy := range proxies {
					proxy.(*Proxy).UpdateTrust()
				}
			}
			log.Printf("Pruning token cache.")
			s.pruneTokenCache()
		}
	}
}

// UpdateTrust promotes the user's trust level if the time has come.
func (u *User) UpdateTrust() {

	// Users can not be promoted beyond MaxTrustLevel.
	if u.Trust >= MaxTrustLevel {
		return
	}

	// A promotion from level n to n+1 takes 2^{n+1} days.
	daysPassed := int64(time.Now().UTC().Sub(u.LastPromoted).Hours() / 24)
	daysRequired := int64(math.Exp2(math.Abs(float64(u.Trust + 1))))
	if daysPassed >= daysRequired {
		u.Trust++
	}
}

// UpdateTrust promotes the proxy's trust level depending on its users.
func (p *Proxy) UpdateTrust() {

	// Determine the minimum trust level of the proxy's users.
	newTrust := UntouchableTrustLevel
	for _, user := range p.Users {
		if user.Trust < newTrust {
			newTrust = user.Trust
		}
	}

	if newTrust > p.Trust {
		log.Printf("Increasing proxy's trust level from %d to %d.", p.Trust, newTrust)
		p.Trust = newTrust
	}

	// A proxy's trust level should be monotonically increasing because its
	// users would only lose a trust level if the proxy was blocked.
	if newTrust < p.Trust {
		// TODO: How do we handle server blocking?
		log.Printf("Bug: proxy was assigned to user with too low a trust level")
	}
}

// SetBlocked marks the given proxy as blocked and adjusts the innocence scores
// of (and potentially blocks) all assigned users.  This function doesn't care
// *where* a proxy is blocked.
func (p *Proxy) SetBlocked() {

	numUsers := len(p.Users)
	if numUsers == 0 {
		log.Printf("Warning: proxy marked as blocked but has no users.")
		return
	}

	for _, user := range p.Users {
		// Add blocking event and determine user's innocence score.
		user.InnocencePs = append(user.InnocencePs, (float64(numUsers)-1.0)/float64(numUsers))
		score := 1.0
		for _, p := range user.InnocencePs {
			score *= p
		}

		// A user's suspicion is the complement of her innocence.
		if 1-score >= MaxSuspicion {
			log.Printf("Banning user %q with suspicion %.2f", user.SecretId, 1-score)
			user.Banned = true
		}
	}
}

// pruneTokenCache removes expired tokens from our token cache.
func (s *SalmonDistributor) pruneTokenCache() {

	s.TokenCacheMutex.Lock()
	defer s.TokenCacheMutex.Unlock()

	prevLen := len(s.TokenCache)
	for token, metaInfo := range s.TokenCache {
		if time.Since(metaInfo.IssueTime) > InvitationTokenExpiry {
			// Time to delete the token.
			log.Printf("Deleting expired token %q issued by user %q.", token, metaInfo.SecretInviterId)
			delete(s.TokenCache, token)
		}
	}
	log.Printf("Pruned token cache from %d to %d entries.", prevLen, len(s.TokenCache))
}

// CreateInvite returns an invitation token if the given user is allowed to
// issue invites, and an error otherwise.
func (s *SalmonDistributor) CreateInvite(secretId string) (string, error) {

	u, exists := s.Users[secretId]
	if !exists {
		return "", errors.New("user ID does not exists")
	}

	if u.Banned {
		return "", errors.New("user is blocked and therefore unable to issue invites")
	}

	if u.Trust < MaxTrustLevel {
		return "", errors.New("user's trust level not high enough to issue invites")
	}

	s.TokenCacheMutex.Lock()
	defer s.TokenCacheMutex.Unlock()

	var token string
	var err error
	for {
		token, err = internal.GetRandBase32(InvitationTokenLength)
		if err != nil {
			return "", err
		}

		if _, exists := s.TokenCache[token]; !exists {
			break
		} else {
			// In the highly unlikely case of a token collision, we simply try
			// again.
			log.Printf("Newly created token already exists.  Trying again.")
		}
	}
	log.Printf("User %q issued new invite token %q.", u.SecretId, token)

	// Add token to our token cache, where it remains until it's redeemed or
	// until it expires.
	s.TokenCache[token] = &TokenMetaInfo{secretId, time.Now().UTC()}

	return token, nil
}

// RedeemInvite redeems the given token.  If redemption was successful, the
// function returns the new user's secret ID; otherwise an error.
func (s *SalmonDistributor) RedeemInvite(token string) (string, error) {

	s.TokenCacheMutex.Lock()
	defer s.TokenCacheMutex.Unlock()

	metaInfo, exists := s.TokenCache[token]
	if !exists {
		return "", errors.New("invite token does not exist")
	}
	// Remove token from our token cache.
	delete(s.TokenCache, token)

	// Is our token still valid?
	if time.Since(metaInfo.IssueTime) > InvitationTokenExpiry {
		return "", errors.New("invite token already expired")
	}

	inviter, exists := s.Users[metaInfo.SecretInviterId]
	if !exists {
		log.Printf("Bug: could not find valid user for invite token.")
		return "", errors.New("invite token came from non-existing user (this is a bug)")
	}

	u, err := s.NewUser(inviter.Trust-1, inviter.SecretId)
	if err != nil {
		return "", err
	}

	return u.SecretId, nil
}

// Register lets a user sign up for Salmon.
func (s *SalmonDistributor) Register() (string, error) {

	return "", errors.New("registration not yet implemented")
}
