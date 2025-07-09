package searchdb

type DB interface {
	BuildIndex(documents []Document) error
	Search(queryString string, limit int, offset int) (*Response, error)
	Close() error
}
