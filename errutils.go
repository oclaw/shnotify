package main

import (
	"errors"
)

func Ignore(err error, toIgnore ...error) error {
	if err == nil {
		return nil
	}
	for _, ignoring := range toIgnore {
		if errors.Is(err, ignoring) {
			return nil
		}
	}
	return err
}
