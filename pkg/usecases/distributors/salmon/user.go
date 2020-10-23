package salmon

import (
	"math"
	"time"
)

const (
	// MaxTrustLevel represents the maximum trust level that a user can get
	// promoted to.  The paper refers to the maximum trust level as "L" and
	// argues that six is a good compromise:
	// <https://censorbib.nymity.ch/pdf/Douglas2016a.pdf#page=4>
	MaxTrustLevel = Trust(6)
	// A user can get UntouchableTrustLevel by being invited directly by us.
	UntouchableTrustLevel = Trust(MaxTrustLevel + 1)
)

// User represents a Salmon user account.
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
	// The last time the user got promoted to a higher trust level.
	LastPromoted time.Time
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
