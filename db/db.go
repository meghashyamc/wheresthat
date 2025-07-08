package db

type DB interface {
	BuildIndex(documents []Document) error
	Search(queryString string, limit int, offset int) (*SearchResponse, error)
}
