package kvdb

type DB interface {
	Set(key string, value string) error
	Get(key string) (string, error)
	Delete(key string) error
	GetAllKeys() ([]string, error)
	Close() error
}
