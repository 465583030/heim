package proto_test

import (
	"testing"

	"euphoria.io/heim/backend"
	"euphoria.io/heim/backend/mock"
	"euphoria.io/heim/proto"
)

func TestIntegration(t *testing.T) {
	backend.IntegrationTest(
		t, func(*proto.Heim) (proto.Backend, error) { return &mock.TestBackend{}, nil })
}
