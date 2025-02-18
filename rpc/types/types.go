package types

import (
	"fmt"
	"github.com/oclaw/shnotify/types"
)

type (
	SaveInvocationRequest = types.InvocationRequest

	SaveInvocationResponse struct {
		InvocationID types.InvocationID `json:"invocation_id"`
	}

	NotifyRequest struct {
		InvocationID types.InvocationID `json:"invocation_id"`
	}

	NotifyResponse struct {
	}

	ErrResponse struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}

	Response[Payload any] struct {
		Data  Payload     `json:"data,omitempty"`
		Error ErrResponse `json:"error,omitempty"`
	}
)

func (rsp *Response[Payload]) Unwrap() (*Payload, error) {
	var emptyErr ErrResponse
	if rsp.Error != emptyErr {
		return nil, fmt.Errorf("rpc error: code=%d message='%s'", rsp.Error.Code, rsp.Error.Message)
	}
	return &rsp.Data, nil
}
