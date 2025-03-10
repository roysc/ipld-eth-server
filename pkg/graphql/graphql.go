// VulcanizeDB
// Copyright © 2020 Vulcanize

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.

// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

// Package graphql provides a GraphQL interface to Ethereum node data.
package graphql

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/cerc-io/ipld-eth-server/v4/pkg/eth"
	"github.com/cerc-io/ipld-eth-server/v4/pkg/shared"
)

var (
	errBlockInvariant = errors.New("block objects must be instantiated with at least one of num or hash")
)

// Account represents an Ethereum account at a particular block.
type Account struct {
	backend       *eth.Backend
	address       common.Address
	blockNrOrHash rpc.BlockNumberOrHash
}

// getState fetches the StateDB object for an account.
func (a *Account) getState(ctx context.Context) (*state.StateDB, error) {
	state, _, err := a.backend.StateAndHeaderByNumberOrHash(ctx, a.blockNrOrHash)
	return state, err
}

func (a *Account) Address(ctx context.Context) (common.Address, error) {
	return a.address, nil
}

func (a *Account) Balance(ctx context.Context) (hexutil.Big, error) {
	state, err := a.getState(ctx)
	if err != nil {
		return hexutil.Big{}, err
	}
	return hexutil.Big(*state.GetBalance(a.address)), nil
}

func (a *Account) TransactionCount(ctx context.Context) (hexutil.Uint64, error) {
	state, err := a.getState(ctx)
	if err != nil {
		return 0, err
	}
	return hexutil.Uint64(state.GetNonce(a.address)), nil
}

func (a *Account) Code(ctx context.Context) (hexutil.Bytes, error) {
	state, err := a.getState(ctx)
	if err != nil {
		return hexutil.Bytes{}, err
	}
	return hexutil.Bytes(state.GetCode(a.address)), nil
}

func (a *Account) Storage(ctx context.Context, args struct{ Slot common.Hash }) (common.Hash, error) {
	state, err := a.getState(ctx)
	if err != nil {
		return common.Hash{}, err
	}
	return state.GetState(a.address, args.Slot), nil
}

// Log represents an individual log message. All arguments are mandatory.
type Log struct {
	backend     *eth.Backend
	transaction *Transaction
	log         *types.Log
	cid         string
	receiptCID  string
	ipldBlock   []byte // log leaf node IPLD block data
	status      uint64
}

// Transaction returns transaction that generated this log entry.
func (l *Log) Transaction(_ context.Context) *Transaction {
	return l.transaction
}

// Account returns the contract account which generated this log.
func (l *Log) Account(_ context.Context, args BlockNumberArgs) *Account {
	return &Account{
		backend:       l.backend,
		address:       l.log.Address,
		blockNrOrHash: args.NumberOrLatest(),
	}
}

// Index returns the index of this log in the block
func (l *Log) Index(_ context.Context) int32 {
	return int32(l.log.Index)
}

// Topics returns the list of 0-4 indexed topics for the log.
func (l *Log) Topics(_ context.Context) []common.Hash {
	return l.log.Topics
}

// Data returns data of this log.
func (l *Log) Data(_ context.Context) hexutil.Bytes {
	return l.log.Data
}

// Cid returns cid of the leaf node of this log.
func (l *Log) Cid(_ context.Context) string {
	return l.cid
}

// IpldBlock returns IPLD block of the leaf node of this log.
func (l *Log) IpldBlock(_ context.Context) hexutil.Bytes {
	return l.ipldBlock
}

// Status returns the status of the receipt IPLD block this Log exists in.
func (l *Log) Status(_ context.Context) int32 {
	return int32(l.status)
}

// ReceiptCID returns the receipt CID of the receipt IPLD block this Log exists in.
func (l *Log) ReceiptCID(_ context.Context) string {
	return l.receiptCID
}

// Transaction represents an Ethereum transaction.
// backend and hash are mandatory; all others will be fetched when required.
type Transaction struct {
	backend *eth.Backend
	hash    common.Hash
	tx      *types.Transaction
	block   *Block
	index   uint64
}

// resolve returns the internal transaction object, fetching it if needed.
func (t *Transaction) resolve(ctx context.Context) (*types.Transaction, error) {
	if t.tx == nil {
		tx, blockHash, _, index := rawdb.ReadTransaction(t.backend.ChainDb(), t.hash)
		if tx != nil {
			t.tx = tx
			blockNrOrHash := rpc.BlockNumberOrHashWithHash(blockHash, false)
			t.block = &Block{
				backend:      t.backend,
				numberOrHash: &blockNrOrHash,
			}
			t.index = index
		}
	}
	return t.tx, nil
}

func (t *Transaction) Hash(ctx context.Context) common.Hash {
	return t.hash
}

func (t *Transaction) InputData(ctx context.Context) (hexutil.Bytes, error) {
	tx, err := t.resolve(ctx)
	if err != nil || tx == nil {
		return hexutil.Bytes{}, err
	}
	return hexutil.Bytes(tx.Data()), nil
}

func (t *Transaction) Gas(ctx context.Context) (hexutil.Uint64, error) {
	tx, err := t.resolve(ctx)
	if err != nil || tx == nil {
		return 0, err
	}
	return hexutil.Uint64(tx.Gas()), nil
}

func (t *Transaction) GasPrice(ctx context.Context) (hexutil.Big, error) {
	tx, err := t.resolve(ctx)
	if err != nil || tx == nil {
		return hexutil.Big{}, err
	}
	return hexutil.Big(*tx.GasPrice()), nil
}

func (t *Transaction) Value(ctx context.Context) (hexutil.Big, error) {
	tx, err := t.resolve(ctx)
	if err != nil || tx == nil {
		return hexutil.Big{}, err
	}
	return hexutil.Big(*tx.Value()), nil
}

func (t *Transaction) Nonce(ctx context.Context) (hexutil.Uint64, error) {
	tx, err := t.resolve(ctx)
	if err != nil || tx == nil {
		return 0, err
	}
	return hexutil.Uint64(tx.Nonce()), nil
}

func (t *Transaction) To(ctx context.Context, args BlockNumberArgs) (*Account, error) {
	tx, err := t.resolve(ctx)
	if err != nil || tx == nil {
		return nil, err
	}
	to := tx.To()
	if to == nil {
		return nil, nil
	}
	return &Account{
		backend:       t.backend,
		address:       *to,
		blockNrOrHash: args.NumberOrLatest(),
	}, nil
}

func (t *Transaction) From(ctx context.Context, args BlockNumberArgs) (*Account, error) {
	tx, err := t.resolve(ctx)
	if err != nil || tx == nil {
		return nil, err
	}
	var signer types.Signer = types.HomesteadSigner{}
	if tx.Protected() {
		signer = types.NewEIP155Signer(tx.ChainId())
	}
	from, _ := types.Sender(signer, tx)

	return &Account{
		backend:       t.backend,
		address:       from,
		blockNrOrHash: args.NumberOrLatest(),
	}, nil
}

func (t *Transaction) Block(ctx context.Context) (*Block, error) {
	if _, err := t.resolve(ctx); err != nil {
		return nil, err
	}
	return t.block, nil
}

func (t *Transaction) Index(ctx context.Context) (*int32, error) {
	if _, err := t.resolve(ctx); err != nil {
		return nil, err
	}
	if t.block == nil {
		return nil, nil
	}
	index := int32(t.index)
	return &index, nil
}

// getReceipt returns the receipt associated with this transaction, if any.
func (t *Transaction) getReceipt(ctx context.Context) (*types.Receipt, error) {
	if _, err := t.resolve(ctx); err != nil {
		return nil, err
	}
	if t.block == nil {
		return nil, nil
	}
	receipts, err := t.block.resolveReceipts(ctx)
	if err != nil {
		return nil, err
	}
	return receipts[t.index], nil
}

func (t *Transaction) Status(ctx context.Context) (*hexutil.Uint64, error) {
	receipt, err := t.getReceipt(ctx)
	if err != nil || receipt == nil {
		return nil, err
	}
	ret := hexutil.Uint64(receipt.Status)
	return &ret, nil
}

func (t *Transaction) GasUsed(ctx context.Context) (*hexutil.Uint64, error) {
	receipt, err := t.getReceipt(ctx)
	if err != nil || receipt == nil {
		return nil, err
	}
	ret := hexutil.Uint64(receipt.GasUsed)
	return &ret, nil
}

func (t *Transaction) CumulativeGasUsed(ctx context.Context) (*hexutil.Uint64, error) {
	receipt, err := t.getReceipt(ctx)
	if err != nil || receipt == nil {
		return nil, err
	}
	ret := hexutil.Uint64(receipt.CumulativeGasUsed)
	return &ret, nil
}

func (t *Transaction) CreatedContract(ctx context.Context, args BlockNumberArgs) (*Account, error) {
	receipt, err := t.getReceipt(ctx)
	if err != nil || receipt == nil || receipt.ContractAddress == (common.Address{}) {
		return nil, err
	}
	return &Account{
		backend:       t.backend,
		address:       receipt.ContractAddress,
		blockNrOrHash: args.NumberOrLatest(),
	}, nil
}

func (t *Transaction) Logs(ctx context.Context) (*[]*Log, error) {
	receipt, err := t.getReceipt(ctx)
	if err != nil || receipt == nil {
		return nil, err
	}
	ret := make([]*Log, 0, len(receipt.Logs))
	for _, log := range receipt.Logs {
		ret = append(ret, &Log{
			backend:     t.backend,
			transaction: t,
			log:         log,
		})
	}
	return &ret, nil
}

func (t *Transaction) R(ctx context.Context) (hexutil.Big, error) {
	tx, err := t.resolve(ctx)
	if err != nil || tx == nil {
		return hexutil.Big{}, err
	}
	_, r, _ := tx.RawSignatureValues()
	return hexutil.Big(*r), nil
}

func (t *Transaction) S(ctx context.Context) (hexutil.Big, error) {
	tx, err := t.resolve(ctx)
	if err != nil || tx == nil {
		return hexutil.Big{}, err
	}
	_, _, s := tx.RawSignatureValues()
	return hexutil.Big(*s), nil
}

func (t *Transaction) V(ctx context.Context) (hexutil.Big, error) {
	tx, err := t.resolve(ctx)
	if err != nil || tx == nil {
		return hexutil.Big{}, err
	}
	v, _, _ := tx.RawSignatureValues()
	return hexutil.Big(*v), nil
}

type BlockType int

// Block represents an Ethereum block.
// backend, and numberOrHash are mandatory. All other fields are lazily fetched
// when required.
type Block struct {
	backend      *eth.Backend
	numberOrHash *rpc.BlockNumberOrHash
	hash         common.Hash
	header       *types.Header
	block        *types.Block
	receipts     []*types.Receipt
}

// resolve returns the internal Block object representing this block, fetching
// it if necessary.
func (b *Block) resolve(ctx context.Context) (*types.Block, error) {
	if b.block != nil {
		return b.block, nil
	}
	if b.numberOrHash == nil {
		latest := rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)
		b.numberOrHash = &latest
	}
	var err error
	b.block, err = b.backend.BlockByNumberOrHash(ctx, *b.numberOrHash)
	if b.block != nil && b.header == nil {
		b.header = b.block.Header()
		if hash, ok := b.numberOrHash.Hash(); ok {
			b.hash = hash
		}
	}
	return b.block, err
}

// resolveHeader returns the internal Header object for this block, fetching it
// if necessary. Call this function instead of `resolve` unless you need the
// additional data (transactions and uncles).
func (b *Block) resolveHeader(ctx context.Context) (*types.Header, error) {
	if b.numberOrHash == nil && b.hash == (common.Hash{}) {
		return nil, errBlockInvariant
	}
	var err error
	if b.header == nil {
		if b.hash != (common.Hash{}) {
			b.header, err = b.backend.HeaderByHash(ctx, b.hash)
		} else {
			b.header, err = b.backend.HeaderByNumberOrHash(ctx, *b.numberOrHash)
		}
	}
	return b.header, err
}

// resolveReceipts returns the list of receipts for this block, fetching them
// if necessary.
func (b *Block) resolveReceipts(ctx context.Context) ([]*types.Receipt, error) {
	if b.receipts == nil {
		hash := b.hash
		if hash == (common.Hash{}) {
			header, err := b.resolveHeader(ctx)
			if err != nil {
				return nil, err
			}
			hash = header.Hash()
		}
		receipts, err := b.backend.GetReceipts(ctx, hash)
		if err != nil {
			return nil, err
		}
		b.receipts = []*types.Receipt(receipts)
	}
	return b.receipts, nil
}

func (b *Block) Number(ctx context.Context) (hexutil.Uint64, error) {
	header, err := b.resolveHeader(ctx)
	if err != nil {
		return 0, err
	}

	return hexutil.Uint64(header.Number.Uint64()), nil
}

func (b *Block) Hash(ctx context.Context) (common.Hash, error) {
	if b.hash == (common.Hash{}) {
		header, err := b.resolveHeader(ctx)
		if err != nil {
			return common.Hash{}, err
		}
		b.hash = header.Hash()
	}
	return b.hash, nil
}

func (b *Block) GasLimit(ctx context.Context) (hexutil.Uint64, error) {
	header, err := b.resolveHeader(ctx)
	if err != nil {
		return 0, err
	}
	return hexutil.Uint64(header.GasLimit), nil
}

func (b *Block) GasUsed(ctx context.Context) (hexutil.Uint64, error) {
	header, err := b.resolveHeader(ctx)
	if err != nil {
		return 0, err
	}
	return hexutil.Uint64(header.GasUsed), nil
}

func (b *Block) Parent(ctx context.Context) (*Block, error) {
	// If the block header hasn't been fetched, and we'll need it, fetch it.
	if b.numberOrHash == nil && b.header == nil {
		if _, err := b.resolveHeader(ctx); err != nil {
			return nil, err
		}
	}
	if b.header != nil && b.header.Number.Uint64() > 0 {
		num := rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(b.header.Number.Uint64() - 1))
		return &Block{
			backend:      b.backend,
			numberOrHash: &num,
			hash:         b.header.ParentHash,
		}, nil
	}
	return nil, nil
}

func (b *Block) Difficulty(ctx context.Context) (hexutil.Big, error) {
	header, err := b.resolveHeader(ctx)
	if err != nil {
		return hexutil.Big{}, err
	}
	return hexutil.Big(*header.Difficulty), nil
}

func (b *Block) Timestamp(ctx context.Context) (hexutil.Uint64, error) {
	header, err := b.resolveHeader(ctx)
	if err != nil {
		return 0, err
	}
	return hexutil.Uint64(header.Time), nil
}

func (b *Block) Nonce(ctx context.Context) (hexutil.Bytes, error) {
	header, err := b.resolveHeader(ctx)
	if err != nil {
		return hexutil.Bytes{}, err
	}
	return hexutil.Bytes(header.Nonce[:]), nil
}

func (b *Block) MixHash(ctx context.Context) (common.Hash, error) {
	header, err := b.resolveHeader(ctx)
	if err != nil {
		return common.Hash{}, err
	}
	return header.MixDigest, nil
}

func (b *Block) TransactionsRoot(ctx context.Context) (common.Hash, error) {
	header, err := b.resolveHeader(ctx)
	if err != nil {
		return common.Hash{}, err
	}
	return header.TxHash, nil
}

func (b *Block) StateRoot(ctx context.Context) (common.Hash, error) {
	header, err := b.resolveHeader(ctx)
	if err != nil {
		return common.Hash{}, err
	}
	return header.Root, nil
}

func (b *Block) ReceiptsRoot(ctx context.Context) (common.Hash, error) {
	header, err := b.resolveHeader(ctx)
	if err != nil {
		return common.Hash{}, err
	}
	return header.ReceiptHash, nil
}

func (b *Block) OmmerHash(ctx context.Context) (common.Hash, error) {
	header, err := b.resolveHeader(ctx)
	if err != nil {
		return common.Hash{}, err
	}
	return header.UncleHash, nil
}

func (b *Block) OmmerCount(ctx context.Context) (*int32, error) {
	block, err := b.resolve(ctx)
	if err != nil || block == nil {
		return nil, err
	}
	count := int32(len(block.Uncles()))
	return &count, err
}

func (b *Block) Ommers(ctx context.Context) (*[]*Block, error) {
	block, err := b.resolve(ctx)
	if err != nil || block == nil {
		return nil, err
	}
	ret := make([]*Block, 0, len(block.Uncles()))
	for _, uncle := range block.Uncles() {
		blockNumberOrHash := rpc.BlockNumberOrHashWithHash(uncle.Hash(), false)
		ret = append(ret, &Block{
			backend:      b.backend,
			numberOrHash: &blockNumberOrHash,
			header:       uncle,
		})
	}
	return &ret, nil
}

func (b *Block) ExtraData(ctx context.Context) (hexutil.Bytes, error) {
	header, err := b.resolveHeader(ctx)
	if err != nil {
		return hexutil.Bytes{}, err
	}
	return hexutil.Bytes(header.Extra), nil
}

func (b *Block) LogsBloom(ctx context.Context) (hexutil.Bytes, error) {
	header, err := b.resolveHeader(ctx)
	if err != nil {
		return hexutil.Bytes{}, err
	}
	return hexutil.Bytes(header.Bloom.Bytes()), nil
}

func (b *Block) TotalDifficulty(ctx context.Context) (hexutil.Big, error) {
	h := b.hash
	if h == (common.Hash{}) {
		header, err := b.resolveHeader(ctx)
		if err != nil {
			return hexutil.Big{}, err
		}
		h = header.Hash()
	}
	td, err := b.backend.GetTd(h)
	if err != nil {
		return hexutil.Big{}, err
	}
	return hexutil.Big(*td), nil
}

// BlockNumberArgs encapsulates arguments to accessors that specify a block number.
type BlockNumberArgs struct {
	// TODO: Ideally we could use input unions to allow the query to specify the
	// block parameter by hash, block number, or tag but input unions aren't part of the
	// standard GraphQL schema SDL yet, see: https://github.com/graphql/graphql-spec/issues/488
	Block *hexutil.Uint64
}

// NumberOr returns the provided block number argument, or the "current" block number or hash if none
// was provided.
func (a BlockNumberArgs) NumberOr(current rpc.BlockNumberOrHash) rpc.BlockNumberOrHash {
	if a.Block != nil {
		blockNr := rpc.BlockNumber(*a.Block)
		return rpc.BlockNumberOrHashWithNumber(blockNr)
	}
	return current
}

// NumberOrLatest returns the provided block number argument, or the "latest" block number if none
// was provided.
func (a BlockNumberArgs) NumberOrLatest() rpc.BlockNumberOrHash {
	return a.NumberOr(rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber))
}

func (b *Block) Miner(ctx context.Context, args BlockNumberArgs) (*Account, error) {
	header, err := b.resolveHeader(ctx)
	if err != nil {
		return nil, err
	}
	return &Account{
		backend:       b.backend,
		address:       header.Coinbase,
		blockNrOrHash: args.NumberOrLatest(),
	}, nil
}

func (b *Block) TransactionCount(ctx context.Context) (*int32, error) {
	block, err := b.resolve(ctx)
	if err != nil || block == nil {
		return nil, err
	}
	count := int32(len(block.Transactions()))
	return &count, err
}

func (b *Block) Transactions(ctx context.Context) (*[]*Transaction, error) {
	block, err := b.resolve(ctx)
	if err != nil || block == nil {
		return nil, err
	}
	ret := make([]*Transaction, 0, len(block.Transactions()))
	for i, tx := range block.Transactions() {
		ret = append(ret, &Transaction{
			backend: b.backend,
			hash:    tx.Hash(),
			tx:      tx,
			block:   b,
			index:   uint64(i),
		})
	}
	return &ret, nil
}

func (b *Block) TransactionAt(ctx context.Context, args struct{ Index int32 }) (*Transaction, error) {
	block, err := b.resolve(ctx)
	if err != nil || block == nil {
		return nil, err
	}
	txs := block.Transactions()
	if args.Index < 0 || int(args.Index) >= len(txs) {
		return nil, nil
	}
	tx := txs[args.Index]
	return &Transaction{
		backend: b.backend,
		hash:    tx.Hash(),
		tx:      tx,
		block:   b,
		index:   uint64(args.Index),
	}, nil
}

func (b *Block) OmmerAt(ctx context.Context, args struct{ Index int32 }) (*Block, error) {
	block, err := b.resolve(ctx)
	if err != nil || block == nil {
		return nil, err
	}
	uncles := block.Uncles()
	if args.Index < 0 || int(args.Index) >= len(uncles) {
		return nil, nil
	}
	uncle := uncles[args.Index]
	blockNumberOrHash := rpc.BlockNumberOrHashWithHash(uncle.Hash(), false)
	return &Block{
		backend:      b.backend,
		numberOrHash: &blockNumberOrHash,
		header:       uncle,
	}, nil
}

// BlockFilterCriteria encapsulates criteria passed to a `logs` accessor inside
// a block.
type BlockFilterCriteria struct {
	Addresses *[]common.Address // restricts matches to events created by specific contracts

	// The Topic list restricts matches to particular event topics. Each event has a list
	// of topics. Topics matches a prefix of that list. An empty element slice matches any
	// topic. Non-empty elements represent an alternative that matches any of the
	// contained topics.
	//
	// Examples:
	// {} or nil          matches any topic list
	// {{A}}              matches topic A in first position
	// {{}, {B}}          matches any topic in first position, B in second position
	// {{A}, {B}}         matches topic A in first position, B in second position
	// {{A, B}}, {C, D}}  matches topic (A OR B) in first position, (C OR D) in second position
	Topics *[][]common.Hash
}

// runFilter accepts a filter and executes it, returning all its results as
// `Log` objects.
func runFilter(ctx context.Context, be *eth.Backend, filter *filters.Filter) ([]*Log, error) {
	logs, err := filter.Logs(ctx)
	if err != nil || logs == nil {
		return nil, err
	}
	ret := make([]*Log, 0, len(logs))
	for _, log := range logs {
		ret = append(ret, &Log{
			backend:     be,
			transaction: &Transaction{backend: be, hash: log.TxHash},
			log:         log,
		})
	}
	return ret, nil
}

func (b *Block) Logs(ctx context.Context, args struct{ Filter BlockFilterCriteria }) ([]*Log, error) {
	var addresses []common.Address
	if args.Filter.Addresses != nil {
		addresses = *args.Filter.Addresses
	}
	var topics [][]common.Hash
	if args.Filter.Topics != nil {
		topics = *args.Filter.Topics
	}
	hash := b.hash
	if hash == (common.Hash{}) {
		header, err := b.resolveHeader(ctx)
		if err != nil {
			return nil, err
		}
		hash = header.Hash()
	}
	// Construct the range filter
	filterSys := filters.NewFilterSystem(b.backend, filters.Config{})
	filter := filterSys.NewBlockFilter(hash, addresses, topics)
	// Run the filter and return all the logs
	return runFilter(ctx, b.backend, filter)
}

func (b *Block) Account(ctx context.Context, args struct {
	Address common.Address
}) (*Account, error) {
	if b.numberOrHash == nil {
		_, err := b.resolveHeader(ctx)
		if err != nil {
			return nil, err
		}
	}
	return &Account{
		backend:       b.backend,
		address:       args.Address,
		blockNrOrHash: *b.numberOrHash,
	}, nil
}

// CallData encapsulates arguments to `call` or `estimateGas`.
// All arguments are optional.
type CallData struct {
	From     *common.Address // The Ethereum address the call is from.
	To       *common.Address // The Ethereum address the call is to.
	Gas      *hexutil.Uint64 // The amount of gas provided for the call.
	GasPrice *hexutil.Big    // The price of each unit of gas, in wei.
	Value    *hexutil.Big    // The value sent along with the call.
	Data     *hexutil.Bytes  // Any data sent with the call.
}

// CallResult encapsulates the result of an invocation of the `call` accessor.
type CallResult struct {
	data    hexutil.Bytes  // The return data from the call
	gasUsed hexutil.Uint64 // The amount of gas used
	status  hexutil.Uint64 // The return status of the call - 0 for failure or 1 for success.
}

func (c *CallResult) Data() hexutil.Bytes {
	return c.data
}

func (c *CallResult) GasUsed() hexutil.Uint64 {
	return c.gasUsed
}

func (c *CallResult) Status() hexutil.Uint64 {
	return c.status
}

func (b *Block) Call(ctx context.Context, args struct {
	Data eth.CallArgs
}) (*CallResult, error) {
	if b.numberOrHash == nil {
		_, err := b.resolve(ctx)
		if err != nil {
			return nil, err
		}
	}
	result, err := eth.DoCall(ctx, b.backend, args.Data, *b.numberOrHash, nil, 5*time.Second, b.backend.RPCGasCap())
	if err != nil {
		return nil, err
	}
	status := hexutil.Uint64(1)
	if result.Failed() {
		status = 0
	}

	return &CallResult{
		data:    result.ReturnData,
		gasUsed: hexutil.Uint64(result.UsedGas),
		status:  status,
	}, nil
}

// Resolver is the top-level object in the GraphQL hierarchy.
type Resolver struct {
	backend *eth.Backend
}

func (r *Resolver) Block(ctx context.Context, args struct {
	Number *hexutil.Uint64
	Hash   *common.Hash
}) (*Block, error) {
	var block *Block
	if args.Number != nil {
		number := rpc.BlockNumber(uint64(*args.Number))
		numberOrHash := rpc.BlockNumberOrHashWithNumber(number)
		block = &Block{
			backend:      r.backend,
			numberOrHash: &numberOrHash,
		}
	} else if args.Hash != nil {
		numberOrHash := rpc.BlockNumberOrHashWithHash(*args.Hash, false)
		block = &Block{
			backend:      r.backend,
			numberOrHash: &numberOrHash,
		}
	} else {
		numberOrHash := rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)
		block = &Block{
			backend:      r.backend,
			numberOrHash: &numberOrHash,
		}
	}
	// Resolve the header, return nil if it doesn't exist.
	// Note we don't resolve block directly here since it will require an
	// additional network request for light client.
	h, err := block.resolveHeader(ctx)
	if err != nil {
		return nil, err
	} else if h == nil {
		return nil, nil
	}
	return block, nil
}

func (r *Resolver) Blocks(ctx context.Context, args struct {
	From hexutil.Uint64
	To   *hexutil.Uint64
}) ([]*Block, error) {
	from := rpc.BlockNumber(args.From)

	var to rpc.BlockNumber
	if args.To != nil {
		to = rpc.BlockNumber(*args.To)
	} else {
		block, err := r.backend.CurrentBlock()
		if err != nil {
			return []*Block{}, nil
		}
		to = rpc.BlockNumber(block.Number().Int64())
	}
	if to < from {
		return []*Block{}, nil
	}
	ret := make([]*Block, 0, to-from+1)
	for i := from; i <= to; i++ {
		numberOrHash := rpc.BlockNumberOrHashWithNumber(i)
		ret = append(ret, &Block{
			backend:      r.backend,
			numberOrHash: &numberOrHash,
		})
	}
	return ret, nil
}

func (r *Resolver) Transaction(ctx context.Context, args struct{ Hash common.Hash }) (*Transaction, error) {
	tx := &Transaction{
		backend: r.backend,
		hash:    args.Hash,
	}
	// Resolve the transaction; if it doesn't exist, return nil.
	t, err := tx.resolve(ctx)
	if err != nil {
		return nil, err
	} else if t == nil {
		return nil, nil
	}
	return tx, nil
}

// FilterCriteria encapsulates the arguments to `logs` on the root resolver object.
type FilterCriteria struct {
	FromBlock *hexutil.Uint64   // beginning of the queried range, nil means genesis block
	ToBlock   *hexutil.Uint64   // end of the range, nil means latest block
	Addresses *[]common.Address // restricts matches to events created by specific contracts

	// The Topic list restricts matches to particular event topics. Each event has a list
	// of topics. Topics matches a prefix of that list. An empty element slice matches any
	// topic. Non-empty elements represent an alternative that matches any of the
	// contained topics.
	//
	// Examples:
	// {} or nil          matches any topic list
	// {{A}}              matches topic A in first position
	// {{}, {B}}          matches any topic in first position, B in second position
	// {{A}, {B}}         matches topic A in first position, B in second position
	// {{A, B}}, {C, D}}  matches topic (A OR B) in first position, (C OR D) in second position
	Topics *[][]common.Hash
}

func (r *Resolver) Logs(ctx context.Context, args struct{ Filter FilterCriteria }) ([]*Log, error) {
	// Convert the RPC block numbers into internal representations
	begin := rpc.LatestBlockNumber.Int64()
	if args.Filter.FromBlock != nil {
		begin = int64(*args.Filter.FromBlock)
	}
	end := rpc.LatestBlockNumber.Int64()
	if args.Filter.ToBlock != nil {
		end = int64(*args.Filter.ToBlock)
	}
	var addresses []common.Address
	if args.Filter.Addresses != nil {
		addresses = *args.Filter.Addresses
	}
	var topics [][]common.Hash
	if args.Filter.Topics != nil {
		topics = *args.Filter.Topics
	}
	// Construct the range filter
	filterSys := filters.NewFilterSystem(r.backend, filters.Config{})
	filter := filterSys.NewRangeFilter(begin, end, addresses, topics)
	return runFilter(ctx, r.backend, filter)
}

// StorageResult represents a storage slot value. All arguments are mandatory.
type StorageResult struct {
	value     []byte
	cid       string
	ipldBlock []byte
}

func (s *StorageResult) Value(ctx context.Context) common.Hash {
	return common.BytesToHash(s.value)
}

func (s *StorageResult) Cid(ctx context.Context) string {
	return s.cid
}

func (s *StorageResult) IpldBlock(ctx context.Context) hexutil.Bytes {
	return hexutil.Bytes(s.ipldBlock)
}

func (r *Resolver) GetStorageAt(ctx context.Context, args struct {
	BlockHash common.Hash
	Contract  common.Address
	Slot      common.Hash
}) (*StorageResult, error) {
	cid, ipldBlock, rlpValue, err := r.backend.IPLDRetriever.RetrieveStorageAtByAddressAndStorageSlotAndBlockHash(args.Contract, args.Slot, args.BlockHash)

	if err != nil {
		if err == sql.ErrNoRows {
			ret := StorageResult{value: []byte{}, cid: "", ipldBlock: []byte{}}

			return &ret, nil
		}

		return nil, err
	}

	if bytes.Equal(rlpValue, eth.EmptyNodeValue) {
		return &StorageResult{value: eth.EmptyNodeValue, cid: cid, ipldBlock: ipldBlock}, nil
	}

	var value interface{}
	err = rlp.DecodeBytes(rlpValue, &value)
	if err != nil {
		return nil, err
	}

	ret := StorageResult{value: value.([]byte), cid: cid, ipldBlock: ipldBlock}
	return &ret, nil
}

func (r *Resolver) GetLogs(ctx context.Context, args struct {
	BlockHash   common.Hash
	BlockNumber *BigInt
	Addresses   *[]common.Address
}) (*[]*Log, error) {
	var filter eth.ReceiptFilter

	if args.Addresses != nil {
		filter.LogAddresses = make([]string, len(*args.Addresses))
		for i, address := range *args.Addresses {
			filter.LogAddresses[i] = address.String()
		}
	}

	// Begin tx
	tx, err := r.backend.DB.Beginx()
	if err != nil {
		return nil, err
	}

	filteredLogs, err := r.backend.Retriever.RetrieveFilteredGQLLogs(tx, filter, &args.BlockHash, args.BlockNumber.ToInt())
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	rctLog := decomposeGQLLogs(filteredLogs)
	if err != nil {
		return nil, err
	}

	ret := make([]*Log, 0, 10)
	for _, l := range rctLog {
		ret = append(ret, &Log{
			backend:    r.backend,
			log:        l.Log,
			cid:        l.CID,
			receiptCID: l.RctCID,
			ipldBlock:  l.LogLeafData,
			transaction: &Transaction{
				hash: l.Log.TxHash,
			},
			status: l.RctStatus,
		})
	}

	return &ret, nil
}

type logsCID struct {
	Log         *types.Log
	CID         string
	RctCID      string
	LogLeafData []byte
	RctStatus   uint64
}

// decomposeGQLLogs return logs for graphql.
func decomposeGQLLogs(logCIDs []eth.LogResult) []logsCID {
	logs := make([]logsCID, len(logCIDs))
	for i, l := range logCIDs {
		topics := make([]common.Hash, 0)
		if l.Topic0 != "" {
			topics = append(topics, common.HexToHash(l.Topic0))
		}
		if l.Topic1 != "" {
			topics = append(topics, common.HexToHash(l.Topic1))
		}
		if l.Topic2 != "" {
			topics = append(topics, common.HexToHash(l.Topic2))
		}
		if l.Topic3 != "" {
			topics = append(topics, common.HexToHash(l.Topic3))
		}

		logs[i] = logsCID{
			Log: &types.Log{
				Address: common.HexToAddress(l.Address),
				Topics:  topics,
				Data:    l.Data,
				Index:   uint(l.Index),
				TxHash:  common.HexToHash(l.TxHash),
			},
			CID:         l.LeafCID,
			RctCID:      l.RctCID,
			LogLeafData: l.LogLeafData,
			RctStatus:   l.RctStatus,
		}
	}

	return logs
}

type EthTransactionCID struct {
	cid       string
	txHash    string
	index     int32
	src       string
	dst       string
	ipfsBlock IPFSBlock
}

func (t EthTransactionCID) Cid(ctx context.Context) string {
	return t.cid
}

func (t EthTransactionCID) TxHash(ctx context.Context) string {
	return t.txHash
}

func (t EthTransactionCID) Index(ctx context.Context) int32 {
	return t.index
}

func (t EthTransactionCID) Src(ctx context.Context) string {
	return t.src
}

func (t EthTransactionCID) Dst(ctx context.Context) string {
	return t.dst
}

func (t EthTransactionCID) BlockByMhKey(ctx context.Context) IPFSBlock {
	return t.ipfsBlock
}

type EthTransactionCIDsConnection struct {
	nodes []*EthTransactionCID
}

func (transactionCIDResult EthTransactionCIDsConnection) Nodes(ctx context.Context) []*EthTransactionCID {
	return transactionCIDResult.nodes
}

type IPFSBlock struct {
	key  string
	data string
}

func (b IPFSBlock) Key(ctx context.Context) string {
	return b.key
}

func (b IPFSBlock) Data(ctx context.Context) string {
	return b.data
}

type EthHeaderCID struct {
	cid          string
	blockNumber  BigInt
	blockHash    string
	parentHash   string
	timestamp    BigInt
	stateRoot    string
	td           BigInt
	txRoot       string
	receiptRoot  string
	uncleRoot    string
	bloom        string
	transactions []*EthTransactionCID
	ipfsBlock    IPFSBlock
}

func (h EthHeaderCID) Cid(ctx context.Context) string {
	return h.cid
}

func (h EthHeaderCID) BlockNumber(ctx context.Context) BigInt {
	return h.blockNumber
}

func (h EthHeaderCID) BlockHash(ctx context.Context) string {
	return h.blockHash
}

func (h EthHeaderCID) ParentHash(ctx context.Context) string {
	return h.parentHash
}

func (h EthHeaderCID) Timestamp(ctx context.Context) BigInt {
	return h.timestamp
}

func (h EthHeaderCID) StateRoot(ctx context.Context) string {
	return h.stateRoot
}

func (h EthHeaderCID) Td(ctx context.Context) BigInt {
	return h.td
}

func (h EthHeaderCID) TxRoot(ctx context.Context) string {
	return h.txRoot
}

func (h EthHeaderCID) ReceiptRoot(ctx context.Context) string {
	return h.receiptRoot
}

func (h EthHeaderCID) UncleRoot(ctx context.Context) string {
	return h.uncleRoot
}

func (h EthHeaderCID) Bloom(ctx context.Context) string {
	return h.bloom
}

func (h EthHeaderCID) EthTransactionCidsByHeaderId(ctx context.Context) EthTransactionCIDsConnection {
	return EthTransactionCIDsConnection{nodes: h.transactions}
}

func (h EthHeaderCID) BlockByMhKey(ctx context.Context) IPFSBlock {
	return h.ipfsBlock
}

type EthHeaderCIDsConnection struct {
	nodes []*EthHeaderCID
}

func (headerCIDResult EthHeaderCIDsConnection) Nodes(ctx context.Context) []*EthHeaderCID {
	return headerCIDResult.nodes
}

type EthHeaderCIDCondition struct {
	BlockNumber *BigInt
	BlockHash   *string
}

func (r *Resolver) AllEthHeaderCids(ctx context.Context, args struct {
	Condition *EthHeaderCIDCondition
}) (*EthHeaderCIDsConnection, error) {
	var headerCIDs []eth.HeaderCIDRecord
	var err error
	if args.Condition.BlockHash != nil {
		headerCID, err := r.backend.Retriever.RetrieveHeaderAndTxCIDsByBlockHash(common.HexToHash(*args.Condition.BlockHash), args.Condition.BlockNumber.ToInt())
		if err != nil {
			if !strings.Contains(err.Error(), "not found") {
				return nil, err
			}
		} else {
			headerCIDs = append(headerCIDs, headerCID)
		}
	} else if args.Condition.BlockNumber != nil {
		headerCIDs, err = r.backend.Retriever.RetrieveHeaderAndTxCIDsByBlockNumber(args.Condition.BlockNumber.ToInt().Int64())
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("provide block number or block hash")
	}

	// Begin tx
	tx, err := r.backend.DB.Beginx()
	if err != nil {
		return nil, err
	}
	defer func() {
		if p := recover(); p != nil {
			shared.Rollback(tx)
			panic(p)
		} else if err != nil {
			shared.Rollback(tx)
		} else {
			err = tx.Commit()
		}
	}()

	var resultNodes []*EthHeaderCID
	for _, headerCID := range headerCIDs {
		var blockNumber BigInt
		blockNumber.UnmarshalText([]byte(headerCID.BlockNumber))

		var timestamp BigInt
		timestamp.SetUint64(headerCID.Timestamp)

		var td BigInt
		td.UnmarshalText([]byte(headerCID.TotalDifficulty))

		ethHeaderCIDNode := EthHeaderCID{
			cid:         headerCID.CID,
			blockNumber: blockNumber,
			blockHash:   headerCID.BlockHash,
			parentHash:  headerCID.ParentHash,
			timestamp:   timestamp,
			stateRoot:   headerCID.StateRoot,
			td:          td,
			txRoot:      headerCID.TxRoot,
			receiptRoot: headerCID.RctRoot,
			uncleRoot:   headerCID.UncleRoot,
			bloom:       Bytes(headerCID.Bloom).String(),
			ipfsBlock: IPFSBlock{
				key:  headerCID.IPLD.Key,
				data: Bytes(headerCID.IPLD.Data).String(),
			},
		}

		for _, txCID := range headerCID.TransactionCIDs {
			ethHeaderCIDNode.transactions = append(ethHeaderCIDNode.transactions, &EthTransactionCID{
				cid:    txCID.CID,
				txHash: txCID.TxHash,
				index:  int32(txCID.Index),
				src:    txCID.Src,
				dst:    txCID.Dst,
			})
		}

		resultNodes = append(resultNodes, &ethHeaderCIDNode)
	}

	return &EthHeaderCIDsConnection{
		nodes: resultNodes,
	}, nil
}

func (r *Resolver) EthTransactionCidByTxHash(ctx context.Context, args struct {
	TxHash      string
	BlockNumber *BigInt
}) (*EthTransactionCID, error) {
	// Need not check args.BlockNumber for nil as .ToInt() uses a pointer receiver and returns nil if BlockNumber is nil
	// https://stackoverflow.com/questions/42238624/calling-a-method-on-a-nil-struct-pointer-doesnt-panic-why-not
	txCID, err := r.backend.Retriever.RetrieveTxCIDByHash(args.TxHash, args.BlockNumber.ToInt())

	if err != nil {
		return nil, err
	}

	return &EthTransactionCID{
		cid:    txCID.CID,
		txHash: txCID.TxHash,
		index:  int32(txCID.Index),
		src:    txCID.Src,
		dst:    txCID.Dst,
		ipfsBlock: IPFSBlock{
			key:  txCID.IPLD.Key,
			data: Bytes(txCID.IPLD.Data).String(),
		},
	}, nil
}
