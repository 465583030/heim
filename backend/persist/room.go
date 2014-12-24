package persist

import (
	"database/sql"
	"fmt"

	"heim/backend"

	"golang.org/x/net/context"
)

var notImpl = fmt.Errorf("not implemented")
var logger = backend.Logger

type Room struct {
	Name      string
	FoundedBy string `db:"founded_by"`
}

func (Room) AfterCreateTable(db *sql.DB) error {
	_, err := db.Exec("CREATE INDEX room_founded_by ON room(founded_by)")
	return err
}

func (r *Room) Bind(b *Backend) *RoomBinding { return &RoomBinding{b, r} }

type RoomBinding struct {
	*Backend
	*Room
}

func (rb *RoomBinding) Latest(ctx context.Context, n int, before backend.Snowflake) (
	[]backend.Message, error) {

	return rb.Backend.latest(ctx, rb.Room, n, before)
}

func (rb *RoomBinding) Join(ctx context.Context, session backend.Session) error {
	return rb.Backend.join(ctx, rb.Room, session)
}

func (rb *RoomBinding) Part(ctx context.Context, session backend.Session) error {
	return rb.Backend.part(ctx, rb.Room, session)
}

func (rb *RoomBinding) Send(ctx context.Context, session backend.Session, msg backend.Message) (
	backend.Message, error) {

	logger(ctx).Printf("Send\n")
	return rb.Backend.sendMessageToRoom(ctx, rb.Room, session, msg, session)
}

func (rb *RoomBinding) Listing(ctx context.Context) (backend.Listing, error) {
	return rb.Backend.listing(ctx, rb.Room)
}

func (rb *RoomBinding) RenameUser(ctx context.Context, session backend.Session, formerName string) (
	*backend.NickEvent, error) {

	event := &backend.NickEvent{
		ID:   session.Identity().ID(),
		From: formerName,
		To:   session.Identity().Name(),
	}
	return event, rb.Backend.broadcast(ctx, rb.Room, session, backend.NickEventType, event, session)
}
