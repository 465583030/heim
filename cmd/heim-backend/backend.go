package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"heim/backend"
	"heim/backend/console"
	"heim/backend/psql"
	"heim/proto"
	"heim/proto/security"
)

var (
	addr    = flag.String("http", ":8080", "")
	id      = flag.String("id", "singleton", "")
	psqlDSN = flag.String("psql", "psql", "")
	static  = flag.String("static", "", "")

	ctrlAddr     = flag.String("control", ":2222", "")
	ctrlHostKey  = flag.String("control-hostkey", "", "")
	ctrlAuthKeys = flag.String("control-authkeys", "", "")
)

var version string

func main() {
	flag.Parse()

	b, err := psql.NewBackend(*psqlDSN, version)
	if err != nil {
		fmt.Printf("error: %s\n", err)
		os.Exit(1)
	}

	server := backend.NewServer(b, *id, *static)
	kms := security.LocalKMS()
	// TODO: get key from somewhere
	kms.SetMasterKey(make([]byte, security.AES256.KeySize()))

	if err := controller(b, kms); err != nil {
		fmt.Printf("error: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("serving on %s\n", *addr)
	http.ListenAndServe(*addr, newVersioningHandler(server))
}

func controller(b proto.Backend, kms security.KMS) error {
	if *ctrlAddr != "" {
		ctrl, err := console.NewController(*ctrlAddr, b, kms)
		if err != nil {
			return err
		}

		if *ctrlHostKey != "" {
			if err := ctrl.AddHostKey(*ctrlHostKey); err != nil {
				return err
			}
		}

		if *ctrlAuthKeys != "" {
			if err := ctrl.AddAuthorizedKeys(*ctrlAuthKeys); err != nil {
				return err
			}
		}

		go ctrl.Serve()
	}
	return nil
}

type versioningHandler struct {
	version string
	handler http.Handler
}

func newVersioningHandler(handler http.Handler) http.Handler {
	return &versioningHandler{
		version: version,
		handler: handler,
	}
}

func (vh *versioningHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if vh.version != "" {
		w.Header().Set("X-Heim-Version", vh.version)
	}
	vh.handler.ServeHTTP(w, r)
}
