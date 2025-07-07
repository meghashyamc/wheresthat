package db

type DB interface {
	BuildIndex(documents any, path string) error
}
