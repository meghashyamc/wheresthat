package searchdb

type DB interface {
	BuildIndex(documents []Document) error
	DeleteDocuments(documentIDs []string) error
	Search(queryString string, limit int, offset int) (*Response, error)
	GetDocCount() (uint64, error)
	Close() error
}
