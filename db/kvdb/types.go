package kvdb

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrNotFound   = errors.New("key not found")
	ErrInvalidKey = errors.New("invalid key")
)

type InvalidKeyError struct {
	Key    string
	Reason string
}
type NotFoundError struct {
	Key string
}

func (e *InvalidKeyError) Error() string {
	return fmt.Sprintf("invalid key %s: %s", e.Key, e.Reason)
}

func (e *InvalidKeyError) Is(target error) bool {
	return target == ErrInvalidKey
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("key not found: %s", e.Key)
}

func (e *NotFoundError) Is(target error) bool {
	return target == ErrNotFound
}

type FileMetadata struct {
	LastIndexed time.Time `json:"last_indexed"`
}
