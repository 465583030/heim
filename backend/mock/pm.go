package mock

import (
	"fmt"
	"sync"
	"time"

	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/security"
	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/scope"
)

type PMTracker struct {
	m   sync.Mutex
	b   *TestBackend
	pms map[snowflake.Snowflake]*PM
}

func (t *PMTracker) Initiate(ctx scope.Context, kms security.KMS, client *proto.Client, recipient proto.UserID) (snowflake.Snowflake, error) {
	pm, err := proto.InitiatePM(ctx, t.b, kms, client, recipient)
	if err != nil {
		return 0, err
	}

	pmKey, _, err := pm.Access(ctx, t.b, kms, client)
	if err != nil {
		return 0, err
	}

	t.m.Lock()
	defer t.m.Unlock()

	if t.pms == nil {
		t.pms = map[snowflake.Snowflake]*PM{}
	}
	t.pms[pm.ID] = &PM{
		RoomBase: RoomBase{
			name:    pm.ID.String(),
			version: t.b.version,
			log:     newMemLog(),
			messageKey: &roomMessageKey{
				id:        fmt.Sprintf("pm:%s", pm.ID),
				timestamp: time.Now(),
				key:       *pmKey,
			},
		},
		pm: pm,
	}
	return pm.ID, nil
}

func (t *PMTracker) Room(ctx scope.Context, kms security.KMS, pmID snowflake.Snowflake, client *proto.Client) (proto.Room, *security.ManagedKey, error) {
	pm, ok := t.pms[pmID]
	if !ok {
		return nil, nil, proto.ErrPMNotFound
	}

	pmKey, _, err := pm.pm.Access(ctx, t.b, kms, client)
	if err != nil {
		return nil, nil, err
	}

	return pm, pmKey, nil
}

type PM struct {
	RoomBase
	pm *proto.PM
}
