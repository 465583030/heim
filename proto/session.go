package proto

import (
	"golang.org/x/net/context"
)

// A Session is a connection between a client and a Room.
type Session interface {
	// ID returns the globally unique identifier for the Session.
	ID() string

	// ServerID returns the globally unique identifier of the server hosting
	// the Session.
	ServerID() string

	// Identity returns the Identity associated with the Session.
	Identity() Identity

	// SetName sets the acting nickname for the Session.
	SetName(name string)

	// Send sends a packet to the Session's client.
	Send(context.Context, PacketType, interface{}) error

	// Close terminates the Session and disconnects the client.
	Close()
}
