package kvdb

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/meghashyamc/wheresthat/config"
	"github.com/meghashyamc/wheresthat/logger"
	bolt "go.etcd.io/bbolt"
)

type BoltDB struct {
	store  *bolt.DB
	logger logger.Logger
}

const (
	boltDefaultBucket = "default"
	lastIndexTimeKey  = "__last_index_time__"
)

func New(logger logger.Logger, cfg *config.Config) (*BoltDB, error) {
	kvDBPath := filepath.Join(cfg.GetStoragePath(), cfg.GetKVDBPath())
	if err := os.MkdirAll(filepath.Dir(kvDBPath), 0755); err != nil {
		logger.Error("failed to create key-value database directory", "err", err.Error(), "path", kvDBPath)
		return nil, fmt.Errorf("failed to create key-value database directory: %w", err)
	}

	store, err := bolt.Open(kvDBPath, 0600, &bolt.Options{
		Timeout: 1 * time.Second,
	})
	if err != nil {
		logger.Error("failed to open database", "err", err.Error(), "path", kvDBPath)
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	boltDB := &BoltDB{
		store:  store,
		logger: logger,
	}

	if err := boltDB.initBucket(); err != nil {
		store.Close()
		return nil, fmt.Errorf("failed to initialize bucket: %w", err)
	}

	return boltDB, nil
}

func (b *BoltDB) initBucket() error {
	return b.store.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte([]byte(boltDefaultBucket)))
		if err != nil {
			b.logger.Error("failed to create bucket", "err", err.Error())
			return fmt.Errorf("failed to create bucket: %w", err)
		}
		return nil
	})
}

func (b *BoltDB) Set(key string, value string) error {
	if err := b.validateKey(key); err != nil {
		return err
	}

	return b.store.Update(func(tx *bolt.Tx) error {
		bucket, err := b.getBucket(tx)
		if err != nil {
			return err
		}

		if err := bucket.Put([]byte(key), []byte(value)); err != nil {
			b.logger.Error("failed to set key", "key", key, "err", err.Error())
			return fmt.Errorf("failed to set key %s: %w", key, err)
		}

		return nil
	})
}

func (b *BoltDB) Get(key string) (string, error) {
	if err := b.validateKey(key); err != nil {
		return "", err
	}

	var value []byte
	err := b.store.View(func(tx *bolt.Tx) error {
		bucket, err := b.getBucket(tx)
		if err != nil {
			return err
		}

		v := bucket.Get([]byte(key))
		if v == nil {
			return &NotFoundError{Key: key}
		}

		value = make([]byte, len(v))
		copy(value, v)
		return nil
	})

	if err != nil {
		if notFoundErr, ok := err.(*NotFoundError); ok {
			b.logger.Warn("key not found", "key", key)
			return "", notFoundErr
		}
		return "", err
	}

	return string(value), nil
}

func (b *BoltDB) Close() error {
	if b.store != nil {
		return b.store.Close()
	}
	return nil
}

func (b *BoltDB) Delete(key string) error {
	if err := b.validateKey(key); err != nil {
		return err
	}

	return b.store.Update(func(tx *bolt.Tx) error {
		bucket, err := b.getBucket(tx)
		if err != nil {
			return err
		}

		if err := bucket.Delete([]byte(key)); err != nil {
			b.logger.Error("failed to delete key", "key", key, "err", err.Error())
			return fmt.Errorf("failed to delete key %s: %w", key, err)
		}

		return nil
	})
}

func (b *BoltDB) GetAllKeys() ([]string, error) {
	var keys []string

	err := b.store.View(func(tx *bolt.Tx) error {
		bucket, err := b.getBucket(tx)
		if err != nil {
			return err
		}

		return bucket.ForEach(func(k, v []byte) error {
			keys = append(keys, string(k))
			return nil
		})
	})

	if err != nil {
		b.logger.Error("failed to get all keys", "err", err.Error())
		return nil, fmt.Errorf("failed to get all keys: %w", err)
	}

	return keys, nil
}

func (b *BoltDB) validateKey(key string) error {
	if key == "" {
		b.logger.Error("key cannot be empty", "key", key)
		return &InvalidKeyError{
			Key:    key,
			Reason: "key cannot be empty",
		}
	}
	return nil
}

func (b *BoltDB) getBucket(tx *bolt.Tx) (*bolt.Bucket, error) {
	bucket := tx.Bucket([]byte(boltDefaultBucket))
	if bucket == nil {
		b.logger.Error("bucket not found", "bucket", boltDefaultBucket)
		return nil, fmt.Errorf("bucket not found")
	}
	return bucket, nil
}
