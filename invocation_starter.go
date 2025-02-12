package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"

	"github.com/google/uuid"
	"github.com/oclaw/shnotify/types"
)

type InvocationIDGen func() (string, error)

func InvocationGenFromStringer[T fmt.Stringer](gen func() (T, error)) InvocationIDGen {
	return func() (string, error) {
		val, err := gen()
		if err != nil {
			return "", err
		}
		return val.String(), nil
	}
}

var uuidInvocationGen = InvocationGenFromStringer(uuid.NewUUID)


type invocationStarter struct {
	config          *ShellTrackerConfig
	clock           Clock
	storage         InvocationStorage
	invocationIDGen InvocationIDGen
}

func NewInvocationStarter(
	cfg *ShellTrackerConfig,
	clock Clock,
	invocationGen InvocationIDGen,
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
		ShellLine: line, // TODO add command cleanup
		Binary:    "",   // TODO add extracting of the binary
	}, nil
}

func (st *invocationStarter) getExtInvocationID(rec *types.ShellInvocationRecord) (string, error) {
	if len(rec.ShellLine) == 0 {
		return "", fmt.Errorf("Empty shell line input for invocation '%s'", rec.InvocationID)
	}
	hash := sha256.New()
	hash.Write([]byte(rec.InvocationID))
	hash.Write([]byte(strconv.Itoa(rec.ParentID)))
	hash.Write([]byte(rec.ShellLine))
	ret := hash.Sum(nil)
	return hex.EncodeToString(ret), nil
}

func (st *invocationStarter) SaveInvocation(ctx context.Context, shellLine, invocationID string) (string, error) {
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

	rec.ExternalInvocationID, err = st.getExtInvocationID(&rec)
	if err != nil {
		return "", err
	}

	if err := st.storage.Store(ctx, &rec); err != nil {
		return "", err
	}

	return rec.ExternalInvocationID, nil
}
