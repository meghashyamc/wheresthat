package kvdb

import "os"

var kvDBPath = os.Getenv("KVDB_PATH")

type DB interface {
	Set(key string, value string) error
	Get(key string) (string, error)
	Close() error
}
