package repo

import (
	"errors"
	"fmt"

	ctrlpb "github.com/Phillezi/tunman/proto"
	"go.etcd.io/bbolt"
	berrors "go.etcd.io/bbolt/errors"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

var (
	ErrNotFound = errors.New("not found")
)

const (
	bucketFwds = "fwds"
)

type Repo struct {
	db *bbolt.DB
}

// OpenDB opens or creates the bbolt DB and initializes buckets.
func OpenDB(path string) (*Repo, error) {
	db, err := bbolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(bucketFwds)); err != nil {
			return fmt.Errorf("create bucket %s: %w", bucketFwds, err)
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, err
	}

	return &Repo{db: db}, nil
}

func (r *Repo) Close() error {
	defer zap.L().Info("db closed")
	return r.db.Close()
}

// SaveFwd stores or updates a fwd.
func (r *Repo) SaveFwd(f *ctrlpb.FwdState) error {
	return r.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketFwds))
		data, err := proto.Marshal(f)
		if err != nil {
			return err
		}
		return b.Put([]byte(f.Id), data)
	})
}

// LoadFwd loads a fwd by hash.
func (r *Repo) LoadFwd(id string) (*ctrlpb.FwdState, error) {
	var f ctrlpb.FwdState
	err := r.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketFwds))
		data := b.Get([]byte(id))
		if data == nil {
			return ErrNotFound
		}
		return proto.Unmarshal(data, &f)
	})
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// LoadAllFwds loads all fwds from the DB.
func (r *Repo) LoadAllFwds() ([]*ctrlpb.FwdState, error) {
	var fwds []*ctrlpb.FwdState
	err := r.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketFwds))
		return b.ForEach(func(_, v []byte) error {
			var f ctrlpb.FwdState
			if err := proto.Unmarshal(v, &f); err != nil {
				return err
			}
			fwds = append(fwds, &f)
			return nil
		})
	})
	return fwds, err
}

// DeleteFwd deletes a fwd by hash.
func (r *Repo) DeleteFwd(hash string) error {
	return r.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketFwds))
		return b.Delete([]byte(hash))
	})
}

func (r *Repo) DeleteFwds(hashes ...string) error {
	if len(hashes) == 0 {
		return nil // Nothing to do
	}
	return r.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketFwds))
		for _, hash := range hashes {
			if err := b.Delete([]byte(hash)); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *Repo) NukeFwds() error {
	return r.db.Update(func(tx *bbolt.Tx) error {
		if err := tx.DeleteBucket([]byte(bucketFwds)); err != nil && err != berrors.ErrBucketNotFound {
			return err
		}
		_, err := tx.CreateBucket([]byte(bucketFwds))
		return err
	})
}
