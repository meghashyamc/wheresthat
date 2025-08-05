package index

type MetadataStore interface {
	Set(bucket string, key string, value string) error
	Get(bucket, key string) (string, error)
	Delete(bucket, key string) error
	GetAllKeys(bucket string) ([]string, error)
	Close() error
}
