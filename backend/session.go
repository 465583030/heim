package backend

import (
	"fmt"

	"github.com/gorilla/websocket"
	"golang.org/x/net/context"
)

type Session interface {
	Identity() Identity
	Send(context.Context, Message) error
}

type memSession struct {
	ctx      context.Context
	conn     *websocket.Conn
	identity *memIdentity
	room     Room

	incoming chan *Command
	outgoing chan *Command
}

func newMemSession(ctx context.Context, conn *websocket.Conn, room Room) *memSession {
	id := conn.RemoteAddr().String()
	loggingCtx := LoggingContext(ctx, fmt.Sprintf("[%s] ", id))

	session := &memSession{
		ctx:      loggingCtx,
		conn:     conn,
		identity: newMemIdentity(id),
		room:     room,

		incoming: make(chan *Command),
		outgoing: make(chan *Command, 100),
	}
	return session
}

func (s *memSession) Identity() Identity { return s.identity }

func (s *memSession) Send(ctx context.Context, msg Message) error {
	encoded, err := msg.Encode()
	if err != nil {
		return err
	}

	cmd := &Command{
		Type: SendType,
		Data: encoded,
	}

	logger := Logger(s.ctx)

	go func() {
		logger.Printf("pushing message: %#v", msg)
		s.outgoing <- cmd
	}()

	return nil
}

func (s *memSession) serve() {
	go s.readMessages()

	logger := Logger(s.ctx)

	for {
		select {
		case cmd := <-s.incoming:
			logger.Printf("received command: id=%s, type=%s", cmd.ID, cmd.Type)

			reply, err := s.handleCommand(cmd)
			if err != nil {
				logger.Printf("error: handleCommand: %s", err)
				reply = err
			}

			logger.Printf("response: id=%s, type=%s, %#v", cmd.ID, cmd.Type, reply)

			resp, err := Response(cmd.ID, cmd.Type, reply)
			if err != nil {
				logger.Printf("error: Response: %s", err)
				return
			}

			data, err := resp.Encode()
			if err != nil {
				logger.Printf("error: Response encode: %s", err)
				return
			}

			if err := s.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				logger.Printf("error: write message: %s", err)
				return
			}
		case cmd := <-s.outgoing:
			data, err := cmd.Encode()
			if err != nil {
				logger.Printf("error: push message encode: %s", err)
				return
			}

			if err := s.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				logger.Printf("error: write message: %s", err)
				return
			}
		}
	}
}

func (s *memSession) readMessages() {
	logger := Logger(s.ctx)

	// TODO: termination condition?
	for {
		_, data, err := s.conn.ReadMessage()
		if err != nil {
			logger.Printf("error: read message: %s", err)
			return
		}

		// TODO: check messageType

		cmd, err := ParseRequest(data)
		if err != nil {
			logger.Printf("error: ParseRequest: %s", err)
			return
		}

		s.incoming <- cmd
	}
}

func (s *memSession) handleCommand(cmd *Command) (interface{}, error) {
	payload, err := cmd.Payload()
	if err != nil {
		return nil, err
	}

	switch msg := payload.(type) {
	case *SendCommand:
		return s.room.Send(s.ctx, s, Message{Content: msg.Content})
	case *LogCommand:
		return s.room.Latest(s.ctx, msg.N)
	case *NickCommand:
		s.identity.name = msg.Name
		return msg, nil
	case *WhoCommand:
		return s.room.Listing(s.ctx)
	default:
		return nil, fmt.Errorf("command type %T not implemented", payload)
	}
}
