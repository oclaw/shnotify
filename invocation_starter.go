package main

import (
	"context"
	"github.com/google/uuid"
	"github.com/oclaw/shnotify/types"
	"os"
	"strings"
)

var uuidInvocationGen = types.InvocationGenFromStringer(uuid.NewUUID)

type invocationStarter struct {
	config          *ShellTrackerConfig
	clock           Clock
	storage         InvocationStorage
	invocationIDGen types.InvocationIDGen
}

func NewInvocationStarter(
	cfg *ShellTrackerConfig,
	clock Clock,
	invocationGen types.InvocationIDGen,
) (*invocationStarter, error) {
	storage, err := NewFsInvocationStorage(cfg.DirPath)
	if err != nil {
		return nil, err
	}

	return &invocationStarter{
		config:          cfg,
		storage:         storage,
		clock:           clock,
		invocationIDGen: invocationGen,
	}, nil
}

type preprocessedCommand struct {
	ShellLine string // cleaned up and safe to save on filesystem shell line
	Binary    string // extracted binary name (e.g. 'ping', 'traceroute', etc)
}

func (st *invocationStarter) preprocessCommand(line string) (preprocessedCommand, error) {
	return preprocessedCommand{
		ShellLine: strings.TrimSpace(line), // TODO add command cleanup
		Binary:    "",                      // TODO add extracting of the binary
	}, nil
}

func (st *invocationStarter) SaveInvocation(
	ctx context.Context,
	shellLine string,
	invocationID types.InvocationID,
) (types.InvocationID, error) {

	rec := types.ShellInvocationRecord{
		InvocationID: invocationID,
		ParentID:     os.Getppid(),
		Timestamp:    st.clock.NowUnix(),
	}

	if len(rec.InvocationID) == 0 {
		var err error
		rec.InvocationID, err = st.invocationIDGen()
		if err != nil {
			return "", err
		}
	}

	command, err := st.preprocessCommand(shellLine)
	if err != nil {
		return "", err
	}

	// TODO ban & allowlist lookups

	rec.ShellLine = command.ShellLine

	if err := st.storage.Store(ctx, &rec); err != nil {
		return "", err
	}

	return rec.InvocationID, nil
}
