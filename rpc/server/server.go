package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/oclaw/shnotify/config"
	"github.com/oclaw/shnotify/core"
	rpctypes "github.com/oclaw/shnotify/rpc/types"
)

type Server struct {
	impl   core.InvocationTracker
	config *config.ShellTrackerConfig
}

func NewServer(
	config *config.ShellTrackerConfig,
	impl core.InvocationTracker,
) (*Server, error) {

	srv := &Server{
		impl:   impl,
		config: config,
	}

	return srv, nil
}

func (s *Server) Serve(ctx context.Context) error {
	if _, err := os.Stat(s.config.RPCSocketName); err == nil {
		os.Remove(s.config.RPCSocketName)
	}

	var listenCfg net.ListenConfig
	listener, err := listenCfg.Listen(ctx, "unix", s.config.RPCSocketName)
	if err != nil {
		return err
	}
	defer listener.Close()

	// TODO cleanup all this copypaste
	http.HandleFunc("/save-invocation",
		func(rw http.ResponseWriter, r *http.Request) {
			var req rpctypes.SaveInvocationRequest
			defer r.Body.Close()
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				rw.WriteHeader(http.StatusBadRequest)
				return
			}
			id, err := s.impl.SaveInvocation(r.Context(), &req)
			if err != nil {
				if err := writeErr(rw, err); err != nil {
					rw.WriteHeader(http.StatusInternalServerError)
				}
				return
			}
			if err := writeOK(rw, rpctypes.SaveInvocationResponse{
				InvocationID: id,
			}); err != nil {
				rw.WriteHeader(http.StatusInternalServerError)
			}
		},
	)

	http.HandleFunc("/notify",
		func(rw http.ResponseWriter, r *http.Request) {
			var req rpctypes.NotifyRequest
			defer r.Body.Close()
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				rw.WriteHeader(http.StatusBadRequest)
				return
			}
			err := s.impl.Notify(r.Context(), req.InvocationID)
			if err != nil {
				if err := writeErr(rw, err); err != nil {
					rw.WriteHeader(http.StatusInternalServerError)
				}
				return
			}
			if err := writeOK(rw, &rpctypes.NotifyResponse{}); err != nil {
				rw.WriteHeader(http.StatusInternalServerError)
			}
		},
	)

	done := make(chan error)
	go func() {
		err = http.Serve(listener, nil)
		done <- err
	}()

	select {
	case err, ok := <-done:
		if ok && err != nil {
			fmt.Printf("server finalized with error %v", err)
			return err
		}
	case <-ctx.Done():
		return ctx.Err()
	}

	panic("unreachable")
}

func writeOK[Response any](rw http.ResponseWriter, appRes Response) error {
	var rpcResponse rpctypes.Response[Response]
	rpcResponse.Data = appRes
	rw.WriteHeader(http.StatusOK)
	return json.NewEncoder(rw).Encode(&rpcResponse)
}

func writeErr(rw http.ResponseWriter, err error) error {
	var rpcResponse rpctypes.Response[any]
	rpcResponse.Error = rpctypes.ErrResponse{
		Code:    1, // TODO
		Message: err.Error(),
	}
	rw.WriteHeader(http.StatusOK)
	return json.NewEncoder(rw).Encode(&rpcResponse)
}
