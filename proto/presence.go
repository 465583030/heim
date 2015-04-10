package proto

import (
	"time"

	"euphoria.io/heim/proto/snowflake"
)

type Presence struct {
	IdentityView
	LastInteracted time.Time           `json:"last_interacted"`
	MessageID      snowflake.Snowflake `json:"message_id"`
	Typing         bool                `json:"typing"`
}
