package core

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/oclaw/shnotify/common"
	"github.com/oclaw/shnotify/types"
)

type fsInvocationStorage struct {
	dirPath string
}

func NewFsInvocationStorage(dirPath string) (InvocationStorage, error) {
	storage := &fsInvocationStorage{
		dirPath: dirPath,
	}
	if err := os.Mkdir(storage.dirPath, os.ModePerm); common.IgnoreErr(err, os.ErrExist) != nil {
		return nil, err
	}

	return storage, nil
}

func (st *fsInvocationStorage) Store(ctx context.Context, rec *types.ShellInvocationRecord) error {
	if len(rec.InvocationID) == 0 {
		return fmt.Errorf("cannot store invocation without id")
	}

	marshaled, err := json.MarshalIndent(&rec, "", " ")
	if err != nil {
		return err
	}

	filename := fmt.Sprintf("%s.json", rec.InvocationID)
	file, err := os.OpenFile(path.Join(st.dirPath, filename), os.O_WRONLY|os.O_CREATE, os.ModePerm)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(marshaled)
	return err
}

func (st *fsInvocationStorage) Get(ctx context.Context, id types.InvocationID) (*types.ShellInvocationRecord, error) {
	if len(id) == 0 {
		return nil, fmt.Errorf("empty invocation id provided")
	}

	file, err := os.Open(path.Join(st.dirPath, fmt.Sprintf("%s.json", id)))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var rec types.ShellInvocationRecord
	if err := json.NewDecoder(file).Decode(&rec); err != nil {
		return nil, err
	}

	return &rec, nil
}

func (st *fsInvocationStorage) Erase(ctx context.Context, id types.InvocationID) error {
	err := os.Remove(fmt.Sprintf("%s.json", id))
	return common.IgnoreErr(err, os.ErrNotExist)
}
