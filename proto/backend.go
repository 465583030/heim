package proto

import (
	"time"

	"euphoria.io/heim/backend/cluster"
	"euphoria.io/heim/proto/security"
	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/scope"
)

// A Backend provides Rooms and an implementation version.
type Backend interface {
	AgentTracker() AgentTracker

	// BanIP globally bans an IP. A zero value for until indicates a
	// permanent ban.
	BanIP(ctx scope.Context, ip string, until time.Time) error

	// UnbanIP removes a global ban.
	UnbanIP(ctx scope.Context, ip string) error

	Close()

	// Create creates a new room.
	CreateRoom(
		ctx scope.Context, kms security.KMS, private bool, name string, managers ...Account) (Room, error)

	// Gets an existing Room by name.
	GetRoom(ctx scope.Context, name string) (Room, error)

	// Peers returns a snapshot of known peers in this backend's cluster.
	Peers() []cluster.PeerDesc

	// Version returns the implementation version string.
	Version() string

	// GetAccount returns the account with the given ID.
	GetAccount(ctx scope.Context, id snowflake.Snowflake) (Account, error)

	// RegisterAccount creates and returns a new, unverified account, along with
	// its (unencrypted) client key.
	RegisterAccount(
		ctx scope.Context, kms security.KMS, namespace, id, password string,
		agentID string, agentKey *security.ManagedKey) (
		Account, *security.ManagedKey, error)

	// ResolveAccount returns any account registered under the given account identity.
	ResolveAccount(ctx scope.Context, namespace, id string) (Account, error)

	// SetStaff marks the given account as staff or not staff.
	SetStaff(ctx scope.Context, accountID snowflake.Snowflake, isStaff bool) error
}
