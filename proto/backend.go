package proto

import (
	"time"

	"euphoria.io/heim/cluster"
	"euphoria.io/heim/proto/jobs"
	"euphoria.io/heim/proto/security"
	"euphoria.io/scope"
)

// A Backend provides Rooms and an implementation version.
type Backend interface {
	AccountManager() AccountManager
	AgentTracker() AgentTracker
	EmailTracker() EmailTracker
	Jobs() jobs.JobService
	PMTracker() PMTracker

	// BanIP globally bans an IP. A zero value for until indicates a
	// permanent ban.
	BanIP(ctx scope.Context, ip string, until time.Time) error

	// UnbanIP removes a global ban.
	UnbanIP(ctx scope.Context, ip string) error

	Close()

	// Create creates a new room.
	CreateRoom(
		ctx scope.Context, kms security.KMS, private bool, name string, managers ...Account) (ManagedRoom, error)

	// Gets an existing Room by name.
	GetRoom(ctx scope.Context, name string) (ManagedRoom, error)

	// Peers returns a snapshot of known peers in this backend's cluster.
	Peers() []cluster.PeerDesc

	// Version returns the implementation version string.
	Version() string

	// NotifyUser broadcasts a packet to all sessions associated with the given userID
	NotifyUser(ctx scope.Context, userID UserID, packetType PacketType, payload interface{}, excluding ...Session) error
}

type BackendFactory func(*Heim) (Backend, error)
