// Copyright (C) 2019-2025, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package customrawdb

import (
	"encoding/binary"

	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/core/types"
	"github.com/ava-labs/libevm/ethdb"
	"github.com/ava-labs/libevm/rlp"
)

var (
	// blobSidecarPrefix + blockHash -> RLP-encoded []BlobSidecarEntry
	blobSidecarPrefix = []byte("bs")
	// blobSidecarNumberPrefix + blockNumber (8 bytes big-endian) -> blockHash
	blobSidecarNumberPrefix = []byte("bn")
)

// BlobSidecarEntry stores a single blob sidecar along with its transaction metadata.
type BlobSidecarEntry struct {
	TxHash  common.Hash
	TxIndex uint64
	Sidecar *types.BlobTxSidecar
}

func blobSidecarKey(hash common.Hash) []byte {
	return append(blobSidecarPrefix, hash.Bytes()...)
}

func blobSidecarNumberKey(number uint64) []byte {
	key := make([]byte, len(blobSidecarNumberPrefix)+8)
	copy(key, blobSidecarNumberPrefix)
	binary.BigEndian.PutUint64(key[len(blobSidecarNumberPrefix):], number)
	return key
}

// WriteBlobSidecars stores blob sidecar entries for a block, indexed by both hash and number.
func WriteBlobSidecars(db ethdb.KeyValueWriter, hash common.Hash, number uint64, entries []BlobSidecarEntry) error {
	data, err := rlp.EncodeToBytes(entries)
	if err != nil {
		return err
	}
	if err := db.Put(blobSidecarKey(hash), data); err != nil {
		return err
	}
	return db.Put(blobSidecarNumberKey(number), hash.Bytes())
}

// ReadBlobSidecars retrieves blob sidecar entries for a block by its hash.
func ReadBlobSidecars(db ethdb.KeyValueReader, hash common.Hash) ([]BlobSidecarEntry, error) {
	data, err := db.Get(blobSidecarKey(hash))
	if err != nil {
		return nil, err
	}
	var entries []BlobSidecarEntry
	if err := rlp.DecodeBytes(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// ReadBlobSidecarsByNumber retrieves blob sidecar entries by block number.
func ReadBlobSidecarsByNumber(db ethdb.KeyValueStore, number uint64) ([]BlobSidecarEntry, error) {
	hashBytes, err := db.Get(blobSidecarNumberKey(number))
	if err != nil {
		return nil, err
	}
	if len(hashBytes) != common.HashLength {
		return nil, nil
	}
	return ReadBlobSidecars(db, common.BytesToHash(hashBytes))
}

// HasBlobSidecars checks whether blob sidecars exist for a given block hash.
func HasBlobSidecars(db ethdb.KeyValueReader, hash common.Hash) bool {
	has, _ := db.Has(blobSidecarKey(hash))
	return has
}

// DeleteBlobSidecars removes blob sidecar entries for a block by its hash and number.
func DeleteBlobSidecars(db ethdb.KeyValueWriter, hash common.Hash, number uint64) error {
	if err := db.Delete(blobSidecarKey(hash)); err != nil {
		return err
	}
	return db.Delete(blobSidecarNumberKey(number))
}

// IterateBlobSidecarNumbers calls fn for each stored block number in ascending order,
// starting from the lowest. It stops when fn returns false or there are no more entries.
// This is used by the pruner to find and delete old blob sidecars.
func IterateBlobSidecarNumbers(db ethdb.Iteratee, fn func(number uint64, hash common.Hash) bool) {
	it := db.NewIterator(blobSidecarNumberPrefix, nil)
	defer it.Release()

	for it.Next() {
		key := it.Key()
		if len(key) != len(blobSidecarNumberPrefix)+8 {
			continue
		}
		number := binary.BigEndian.Uint64(key[len(blobSidecarNumberPrefix):])
		value := it.Value()
		if len(value) != common.HashLength {
			continue
		}
		hash := common.BytesToHash(value)
		if !fn(number, hash) {
			break
		}
	}
}
