package kvdb

type DB interface {
	Set(key string, value string) error
	Get(key string) (string, error)
}
