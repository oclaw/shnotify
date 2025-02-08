package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
)

type InvocationStorage interface {
	Store(ctx context.Context, rec *ShellInvocationRecord) error
	Get(ctx context.Context, id string) (*ShellInvocationRecord, error)
	Erase(ctx context.Context, id string) error
}

type fsInvocationStorage struct {
	dirPath string
}

func (st *shellTracker) Start(ctx context.Context) error {
	return nil
}

func NewFsInvocationStorage(dirPath string) (InvocationStorage, error) {
	storage := &fsInvocationStorage{
		dirPath: dirPath,
	}
	if err := os.Mkdir(storage.dirPath, os.ModePerm); Ignore(err, os.ErrExist) != nil {
		return nil, err
	}

	return storage, nil
}

func (st *fsInvocationStorage) Store(ctx context.Context, rec *ShellInvocationRecord) error {
	if len(rec.ExternalInvocationID) == 0 {
		return fmt.Errorf("cannot store invocation '%s' without external id", rec.InvocationID)
	}

	marshaled, err := json.MarshalIndent(&rec, "", " ")
	if err != nil {
		return err
	}

	return os.WriteFile(
		path.Join(st.dirPath, fmt.Sprintf("%s.json", rec.ExternalInvocationID)),
		marshaled,
		os.ModePerm,
	)
}

func (st *fsInvocationStorage) Get(ctx context.Context, extId string) (*ShellInvocationRecord, error) {
	if len(extId) == 0 {
		return nil, fmt.Errorf("empty ext invocation id provided")
	}

	file, err := os.Open(path.Join(st.dirPath, fmt.Sprintf("%s.json", extId)))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var rec ShellInvocationRecord
	if err := json.NewDecoder(file).Decode(&rec); err != nil {
		return nil, err
	}

	return &rec, nil
}

func (st *fsInvocationStorage) Erase(ctx context.Context, id string) error {
	err := os.Remove(fmt.Sprintf("%s.json", id))
	return Ignore(err, os.ErrNotExist)
}
