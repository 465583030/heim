package psql

import (
	"database/sql"
	"fmt"

	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/security"
	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/scope"
)

type PM struct {
	ID                    string
	Initiator             string
	Receiver              string
	ReceiverMAC           []byte `db:"receiver_mac"`
	IV                    []byte
	EncryptedSystemKey    []byte `db:"encrypted_system_key"`
	EncryptedInitiatorKey []byte `db:"encrypted_initiator_key"`
	EncryptedReceiverKey  []byte `db:"encrypted_receiver_key"`
}

func (pm *PM) ToBackend() *proto.PM {
	bpm := &proto.PM{
		Receiver:    proto.UserID(pm.Receiver),
		ReceiverMAC: pm.ReceiverMAC,
		IV:          pm.IV,
		EncryptedSystemKey: &security.ManagedKey{
			KeyType:      proto.RoomMessageKeyType,
			Ciphertext:   pm.EncryptedSystemKey,
			ContextKey:   "pm",
			ContextValue: pm.ID,
		},
		EncryptedInitiatorKey: &security.ManagedKey{
			KeyType:    proto.RoomMessageKeyType,
			IV:         pm.IV,
			Ciphertext: pm.EncryptedInitiatorKey,
		},
	}
	if len(pm.EncryptedReceiverKey) > 0 {
		bpm.EncryptedReceiverKey = &security.ManagedKey{
			KeyType:    proto.RoomMessageKeyType,
			IV:         pm.IV,
			Ciphertext: pm.EncryptedReceiverKey,
		}
	}
	// ignore id parsing errors
	_ = bpm.ID.FromString(pm.ID)
	_ = bpm.Initiator.FromString(pm.Initiator)
	return bpm
}

type PMRoomBinding struct {
	RoomBinding
	pm *proto.PM
}

func (pmrb *PMRoomBinding) MessageKeyID(ctx scope.Context) (string, bool, error) {
	return fmt.Sprintf("pm:%s", pmrb.pm.ID), true, nil
}

type PMTracker struct {
	*Backend
}

func (t *PMTracker) Initiate(ctx scope.Context, kms security.KMS, client *proto.Client, recipient proto.UserID) (snowflake.Snowflake, error) {
	pm, err := proto.InitiatePM(ctx, t.Backend, kms, client, recipient)
	if err != nil {
		return 0, err
	}
	row := &PM{
		ID:                    pm.ID.String(),
		Initiator:             pm.Initiator.String(),
		Receiver:              string(pm.Receiver),
		ReceiverMAC:           pm.ReceiverMAC,
		IV:                    pm.IV,
		EncryptedSystemKey:    pm.EncryptedSystemKey.Ciphertext,
		EncryptedInitiatorKey: pm.EncryptedInitiatorKey.Ciphertext,
	}
	if pm.EncryptedReceiverKey != nil {
		row.EncryptedReceiverKey = pm.EncryptedReceiverKey.Ciphertext
	}
	if err := t.Backend.Insert(row); err != nil {
		return 0, err
	}
	return pm.ID, nil
}

func (t *PMTracker) Room(ctx scope.Context, kms security.KMS, pmID snowflake.Snowflake, client *proto.Client) (proto.Room, *security.ManagedKey, error) {
	row, err := t.Backend.Get(PM{}, pmID.String())
	if row == nil || err != nil {
		if row == nil || err == sql.ErrNoRows {
			return nil, nil, proto.ErrPMNotFound
		}
	}

	pm := row.(*PM).ToBackend()
	pmKey, modified, err := pm.Access(ctx, t.Backend, kms, client)
	if err != nil {
		return nil, nil, err
	}

	if modified {
		_, err := t.Backend.DbMap.Exec(
			"UPDATE pm SET receiver = $2, encrypted_receiver_key = $3 WHERE id = $1",
			pm.ID.String(), string(pm.Receiver), pm.EncryptedReceiverKey.Ciphertext)
		if err != nil {
			return nil, nil, err
		}
	}

	room := &PMRoomBinding{
		RoomBinding: RoomBinding{
			RoomName: fmt.Sprintf("pm:%s", pm.ID),
			Backend:  t.Backend,
		},
		pm: pm,
	}

	return room, pmKey, nil
}