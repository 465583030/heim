package proto

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"golang.org/x/crypto/poly1305"

	"euphoria.io/heim/proto/security"
	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/scope"
)

// A Listing is a sortable list of Identitys present in a Room.
// TODO: these should be Sessions
type Listing []SessionView

func (l Listing) Len() int      { return len(l) }
func (l Listing) Swap(i, j int) { l[i], l[j] = l[j], l[i] }

func (l Listing) Less(i, j int) bool {
	if l[i].Name == l[j].Name {
		if l[i].ID == l[j].ID {
			return l[i].SessionID < l[j].SessionID
		}
		return l[i].ID < l[j].ID
	}
	return l[i].Name < l[j].Name
}

// A Room is a nexus of communication. Users connect to a Room via
// Session and interact.
type Room interface {
	Log

	// BanAgent bans an agent from the room. A zero value for until
	// indicates a permanent ban.
	BanAgent(ctc scope.Context, agentID string, until time.Time) error

	// UnbanAgent removes an agent ban from the room.
	UnbanAgent(ctc scope.Context, agentID string) error

	// BanIP bans an IP from the room. A zero value for until indicates
	// a permanent ban.
	BanIP(ctc scope.Context, ip string, until time.Time) error

	// UnbanIP removes an IP ban from the room.
	UnbanIP(ctc scope.Context, ip string) error

	// Join inserts a Session into the Room's global presence.
	Join(scope.Context, Session) error

	// Part removes a Session from the Room's global presence.
	Part(scope.Context, Session) error

	// IsValidParent checks whether the message with the given ID is able to be replied to.
	IsValidParent(id snowflake.Snowflake) (bool, error)

	// Send broadcasts a Message from a Session to the Room.
	Send(scope.Context, Session, Message) (Message, error)

	// Edit modifies or deletes a message.
	EditMessage(scope.Context, Session, EditMessageCommand) error

	// Listing returns the current global list of connected sessions to this
	// Room.
	Listing(scope.Context) (Listing, error)

	// RenameUser updates the nickname of a Session in this Room.
	RenameUser(ctx scope.Context, session Session, formerName string) (*NickEvent, error)

	// Version returns the version of the server hosting this Room.
	Version() string

	// GenerateMasterKey generates and stores a new key and nonce
	// for the room. This invalidates all grants made with the
	// previous key.
	GenerateMasterKey(ctx scope.Context, kms security.KMS) (RoomKey, error)

	// MasterKey returns the room's current key, or nil if the room is unlocked.
	MasterKey(ctx scope.Context) (RoomKey, error)

	// SaveCapability saves the given capability.
	SaveCapability(ctx scope.Context, capability security.Capability) error

	// GetCapability retrieves the capability under the given ID, or
	// returns nil if it doesn't exist.
	GetCapability(ctx scope.Context, id string) (security.Capability, error)

	// KeyPair returns the current encrypted ManagedKeyPair for the room.
	KeyPair() security.ManagedKeyPair

	// Unlock decrypts the room's ManagedKeyPair with the given key and returns it.
	Unlock(ownerKey *security.ManagedKey) (*security.ManagedKeyPair, error)
}

type RoomKey interface {
	// ID returns a unique identifier for the key.
	KeyID() string

	// Timestamp returns when the key was generated.
	Timestamp() time.Time

	// Nonce returns the current 128-bit nonce for the room.
	Nonce() []byte

	// ManagedKey returns the current encrypted ManagedKey for the room.
	ManagedKey() security.ManagedKey
}

func NewRoomSecurity(kms security.KMS, roomName string) (*RoomSecurity, error) {
	kType := security.AES128
	kpType := security.Curve25519

	// Use one KMS request to obtain all the randomness we need:
	//   - key-encrypting-key IV
	//   - private key for account grants
	randomData, err := kms.GenerateNonce(kType.BlockSize() + kpType.PrivateKeySize())
	if err != nil {
		return nil, fmt.Errorf("rng error: %s", err)
	}
	randomReader := bytes.NewReader(randomData)

	// Generate IV with random data.
	iv := make([]byte, kType.BlockSize())
	if _, err := io.ReadFull(randomReader, iv); err != nil {
		return nil, fmt.Errorf("rng error: %s", err)
	}

	// Generate private key using randomReader.
	keyPair, err := kpType.Generate(randomReader)
	if err != nil {
		return nil, fmt.Errorf("keypair generation error: %s", err)
	}

	// Generate key-encrypting-key. This will be returned encrypted, using the
	// name of the room as its context.
	encryptedKek, err := kms.GenerateEncryptedKey(kType, "room", roomName)
	if err != nil {
		return nil, fmt.Errorf("key generation error: %s", err)
	}

	// Decrypt key-encrypting-key so we can encrypt keypair.
	kek := encryptedKek.Clone()
	if err = kms.DecryptKey(&kek); err != nil {
		return nil, fmt.Errorf("key decryption error: %s", err)
	}

	// Encrypt private key.
	keyPair.IV = iv
	if err = keyPair.Encrypt(&kek); err != nil {
		return nil, fmt.Errorf("keypair encryption error: %s", err)
	}

	// Generate message authentication code, for verifying a given key-encryption-key.
	var (
		mac [16]byte
		key [32]byte
	)
	copy(key[:], kek.Plaintext)
	poly1305.Sum(&mac, iv, &key)

	sec := &RoomSecurity{
		MAC:              mac[:],
		KeyEncryptingKey: *encryptedKek,
		KeyPair:          *keyPair,
	}
	return sec, nil
}

type RoomSecurity struct {
	MAC              []byte
	KeyEncryptingKey security.ManagedKey
	KeyPair          security.ManagedKeyPair
}

func (sec *RoomSecurity) Unlock(ownerKey *security.ManagedKey) (*security.ManagedKeyPair, error) {
	if ownerKey.Encrypted() {
		return nil, security.ErrKeyMustBeDecrypted
	}

	var (
		mac [16]byte
		key [32]byte
	)
	copy(mac[:], sec.MAC)
	copy(key[:], ownerKey.Plaintext)
	if !poly1305.Verify(&mac, sec.KeyPair.IV, &key) {
		return nil, ErrAccessDenied
	}

	kek := sec.KeyEncryptingKey.Clone()
	if err := kek.Decrypt(ownerKey); err != nil {
		return nil, err
	}

	kp := sec.KeyPair.Clone()
	if err := kp.Decrypt(&kek); err != nil {
		return nil, err
	}

	return &kp, nil
}
