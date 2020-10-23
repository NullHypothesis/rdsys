package salmon

import (
	"log"

	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/core"
)

// Proxy represents a circumvention proxy that's handed out to users.
type Proxy struct {
	core.Resource
	ReservedFor int
	Trust       Trust
}

// IsDepleted returns true if the proxy reached its capacity and can no longer
// accommodate new users.
func (p *Proxy) IsDepleted(assignments *ProxyAssignments) bool {
	return len(assignments.GetUsers(p)) >= MaxClients
}

// UpdateTrust promotes the proxy's trust level depending on its users.
func (p *Proxy) UpdateTrust(assignments *ProxyAssignments) {

	// Determine the minimum trust level of the proxy's users.
	newTrust := UntouchableTrustLevel
	users := assignments.GetUsers(p)
	for _, user := range users {
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
func (p *Proxy) SetBlocked(assignments *ProxyAssignments) {

	// numUsers := len(p.Users)
	numUsers := len(assignments.GetUsers(p))
	if numUsers == 0 {
		log.Printf("Warning: proxy marked as blocked but has no users.")
		return
	}

	users := assignments.GetUsers(p)
	for _, user := range users {
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
