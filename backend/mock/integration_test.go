package mock

import (
	"testing"

	"euphoria.io/heim/backend"
	"euphoria.io/heim/proto"

	//. "github.com/smartystreets/goconvey/convey"
)

func TestTestBackend(t *testing.T) {
	backend.IntegrationTest(t, func() proto.Backend { return &TestBackend{} })
}
