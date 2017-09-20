package planb

import (
	"io"
)

// KVStore is an abstraction of an underlying
// key/value store implementation.
type KVStore interface {
	// Begin starts a batch operations. The writable
	// flag indicates if the operation is potentially
	// mutating (true) or read-only (false).
	//
	// Please note that a KVBatch operation is not
	// necessarily transactional. The level of atomicity and
	// isolation is subject to the underlying store
	// implementation.
	Begin(mutating bool) (KVBatch, error)

	// Restore restores the store from a data stream.
	Restore(r io.Reader) error

	// Snapshot writes a snapshot to the provided writer.
	Snapshot(w io.Writer) error

	// Close closes the store.
	Close() error
}

// KVBatch allows to perform action on the store.
type KVBatch interface {
	// Rollback rolls back the batch.
	Rollback() error

	// Commit executes the batch. Please
	Commit() error

	// Get retrieves a value for a key.
	Get(key []byte) ([]byte, error)

	// Put stores a value at a key.
	Put(key, val []byte) error

	// Delete deletes a key.
	Delete(key []byte) error
}
