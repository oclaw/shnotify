package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/oclaw/shnotify/types"
	rpctypes "github.com/oclaw/shnotify/rpc/types"
)

type Client struct {
	http *http.Client
}

func NewClient(path string) (*Client, error) {
	httpClient := &http.Client{
		Transport: &http.Transport{
			// TODO adapt for any kind of sockets
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, "unix", path)
			},
		},
	}
	return &Client {
		http: httpClient,
	}, nil
}

func (cl *Client) SaveInvocation(ctx context.Context, req *types.InvocationRequest) (types.InvocationID, error) {
	res, err := callHTTP[rpctypes.SaveInvocationRequest, rpctypes.SaveInvocationResponse](
		ctx,
		cl,
		(*rpctypes.SaveInvocationRequest)(req),
		requestContext{
			method: http.MethodPost,
			path: "save-invocation",
		},
	)
	if err != nil {
		return "", err
	}
	return res.InvocationID, nil
}

func (cl *Client) Notify(ctx context.Context, id types.InvocationID) error {
    _, err := callHTTP[rpctypes.NotifyRequest, any](
		ctx,
		cl,
		&rpctypes.NotifyRequest{
			InvocationID: id,
		},
		requestContext{
			method: http.MethodPost,
			path: "notify",
		},
	)
	if err != nil {
		return err
	}
	return nil
}
type requestContext struct {
	method string
	path string
}

func callHTTP[Req, Res any](ctx context.Context, cl *Client, req *Req, reqCtx requestContext) (*Res, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	var remoteURL url.URL
	remoteURL.Host = "localhost"
	remoteURL.Path = reqCtx.path
	remoteURL.Scheme = "http"

	httpReq, err := http.NewRequest(reqCtx.method, remoteURL.String(), bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	httpRes, err := cl.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = httpRes.Body.Close()
	}()

	if httpRes.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Unexpected RPC response: %v", httpRes.Status)
	}

	var rpcResponse rpctypes.Response[Res]
	if err := json.NewDecoder(httpRes.Body).Decode(&rpcResponse); err != nil {
		return nil, err
	}

	return rpcResponse.Unwrap()
}

type InvocationTracker interface {
	SaveInvocation(ctx context.Context, req *types.InvocationRequest) (types.InvocationID, error)
	Notify(ctx context.Context, id types.InvocationID) error
}
