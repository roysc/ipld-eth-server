// VulcanizeDB
// Copyright © 2019 Vulcanize

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

package eth

import (
	"fmt"
	"math/big"
	"strconv"

	"github.com/cerc-io/ipld-eth-server/v4/pkg/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/statediff/indexer/models"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/cerc-io/ipld-eth-server/v4/pkg/shared"
)

// Retriever interface for substituting mocks in tests
type Retriever interface {
	RetrieveFirstBlockNumber() (int64, error)
	RetrieveLastBlockNumber() (int64, error)
	Retrieve(filter SubscriptionSettings, blockNumber int64) ([]CIDWrapper, bool, error)
}

// CIDRetriever satisfies the CIDRetriever interface for ethereum
type CIDRetriever struct {
	db     *sqlx.DB
	gormDB *gorm.DB
}

type IPLDModelRecord struct {
	models.IPLDModel
}

// TableName overrides the table name used by IPLD
func (IPLDModelRecord) TableName() string {
	return "public.blocks"
}

type HeaderCIDRecord struct {
	CID             string `gorm:"column:cid"`
	BlockHash       string `gorm:"primaryKey"`
	BlockNumber     string `gorm:"primaryKey"`
	ParentHash      string
	Timestamp       uint64
	StateRoot       string
	TotalDifficulty string `gorm:"column:td"`
	TxRoot          string
	RctRoot         string `gorm:"column:receipt_root"`
	UncleRoot       string
	Bloom           []byte
	MhKey           string

	// gorm doesn't check if foreign key exists in database.
	// It is required to eager load relations using preload.
	TransactionCIDs []TransactionCIDRecord `gorm:"foreignKey:HeaderID,BlockNumber;references:BlockHash,BlockNumber"`
	IPLD            IPLDModelRecord        `gorm:"foreignKey:MhKey,BlockNumber;references:Key,BlockNumber"`
}

// TableName overrides the table name used by HeaderCIDRecord
func (HeaderCIDRecord) TableName() string {
	return "eth.header_cids"
}

type TransactionCIDRecord struct {
	CID         string `gorm:"column:cid"`
	TxHash      string `gorm:"primaryKey"`
	BlockNumber string `gorm:"primaryKey"`
	HeaderID    string `gorm:"column:header_id"`
	Index       int64
	Src         string
	Dst         string
	MhKey       string
	IPLD        IPLDModelRecord `gorm:"foreignKey:MhKey,BlockNumber;references:Key,BlockNumber"`
}

// TableName overrides the table name used by TransactionCIDRecord
func (TransactionCIDRecord) TableName() string {
	return "eth.transaction_cids"
}

// NewCIDRetriever returns a pointer to a new CIDRetriever which supports the CIDRetriever interface
func NewCIDRetriever(db *sqlx.DB) *CIDRetriever {
	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})

	if err != nil {
		log.Error(err)
		return nil
	}

	return &CIDRetriever{
		db:     db,
		gormDB: gormDB,
	}
}

// RetrieveFirstBlockNumber is used to retrieve the first block number in the db
func (ecr *CIDRetriever) RetrieveFirstBlockNumber() (int64, error) {
	var blockNumber int64
	err := ecr.db.Get(&blockNumber, "SELECT block_number FROM eth.header_cids ORDER BY block_number ASC LIMIT 1")
	return blockNumber, err
}

// RetrieveLastBlockNumber is used to retrieve the latest block number in the db
func (ecr *CIDRetriever) RetrieveLastBlockNumber() (int64, error) {
	var blockNumber int64
	err := ecr.db.Get(&blockNumber, "SELECT block_number FROM eth.header_cids ORDER BY block_number DESC LIMIT 1")
	return blockNumber, err
}

// Retrieve is used to retrieve all of the CIDs which conform to the passed StreamFilters
func (ecr *CIDRetriever) Retrieve(filter SubscriptionSettings, blockNumber int64) ([]CIDWrapper, bool, error) {
	log.Debug("retrieving cids")

	// Begin new db tx
	tx, err := ecr.db.Beginx()
	if err != nil {
		return nil, true, err
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

	// Retrieve cached header CIDs at this block height
	var headers []models.HeaderModel
	headers, err = ecr.RetrieveHeaderCIDs(tx, blockNumber)
	if err != nil {
		log.Error("header cid retrieval error", err)
		return nil, true, err
	}
	cws := make([]CIDWrapper, len(headers))
	empty := true
	for i, header := range headers {
		cw := new(CIDWrapper)
		cw.BlockNumber = big.NewInt(blockNumber)
		if !filter.HeaderFilter.Off {
			cw.Header = header
			empty = false
			if filter.HeaderFilter.Uncles {
				// Retrieve uncle cids for this header id
				var uncleCIDs []models.UncleModel
				uncleCIDs, err = ecr.RetrieveUncleCIDsByHeaderID(tx, header.BlockHash)
				if err != nil {
					log.Error("uncle cid retrieval error")
					return nil, true, err
				}
				cw.Uncles = uncleCIDs
			}
		}
		// Retrieve cached trx CIDs
		if !filter.TxFilter.Off {
			cw.Transactions, err = ecr.RetrieveTxCIDs(tx, filter.TxFilter, header.BlockHash)
			if err != nil {
				log.Error("transaction cid retrieval error")
				return nil, true, err
			}
			if len(cw.Transactions) > 0 {
				empty = false
			}
		}
		trxHashes := make([]string, len(cw.Transactions))
		for j, t := range cw.Transactions {
			trxHashes[j] = t.TxHash
		}
		// Retrieve cached receipt CIDs
		if !filter.ReceiptFilter.Off {
			cw.Receipts, err = ecr.RetrieveRctCIDs(tx, filter.ReceiptFilter, 0, header.BlockHash, trxHashes)
			if err != nil {
				log.Error("receipt cid retrieval error")
				return nil, true, err
			}
			if len(cw.Receipts) > 0 {
				empty = false
			}
		}
		// Retrieve cached state CIDs
		if !filter.StateFilter.Off {
			cw.StateNodes, err = ecr.RetrieveStateCIDs(tx, filter.StateFilter, header.BlockHash)
			if err != nil {
				log.Error("state cid retrieval error")
				return nil, true, err
			}
			if len(cw.StateNodes) > 0 {
				empty = false
			}
		}
		// Retrieve cached storage CIDs
		if !filter.StorageFilter.Off {
			cw.StorageNodes, err = ecr.RetrieveStorageCIDs(tx, filter.StorageFilter, header.BlockHash)
			if err != nil {
				log.Error("storage cid retrieval error")
				return nil, true, err
			}
			if len(cw.StorageNodes) > 0 {
				empty = false
			}
		}
		cws[i] = *cw
	}

	return cws, empty, err
}

// RetrieveHeaderCIDs retrieves and returns all of the header cids at the provided blockheight
func (ecr *CIDRetriever) RetrieveHeaderCIDs(tx *sqlx.Tx, blockNumber int64) ([]models.HeaderModel, error) {
	log.Debug("retrieving header cids for block ", blockNumber)
	headers := make([]models.HeaderModel, 0)
	pgStr := `SELECT CAST(block_number as Text), block_hash, parent_hash, cid, mh_key, CAST(td as Text), node_id,
				CAST(reward as Text), state_root, uncle_root,tx_root, receipt_root, bloom, timestamp, times_validated, coinbase
				FROM eth.header_cids
				WHERE block_number = $1`
	return headers, tx.Select(&headers, pgStr, blockNumber)
}

// RetrieveUncleCIDsByHeaderID retrieves and returns all of the uncle cids for the provided header
func (ecr *CIDRetriever) RetrieveUncleCIDsByHeaderID(tx *sqlx.Tx, headerID string) ([]models.UncleModel, error) {
	log.Debug("retrieving uncle cids for block id ", headerID)
	headers := make([]models.UncleModel, 0)
	pgStr := `SELECT CAST(block_number as Text), header_id, block_hash, parent_hash, cid, mh_key, CAST(reward as text)
				FROM eth.uncle_cids
				WHERE header_id = $1`
	return headers, tx.Select(&headers, pgStr, headerID)
}

// RetrieveTxCIDs retrieves and returns all of the trx cids at the provided blockheight that conform to the provided filter parameters
// also returns the ids for the returned transaction cids
func (ecr *CIDRetriever) RetrieveTxCIDs(tx *sqlx.Tx, txFilter TxFilter, headerID string) ([]models.TxModel, error) {
	log.Debug("retrieving transaction cids for header id ", headerID)
	args := make([]interface{}, 0, 3)
	results := make([]models.TxModel, 0)
	id := 1
	pgStr := fmt.Sprintf(`SELECT CAST(transaction_cids.block_number as Text), transaction_cids.tx_hash,
				transaction_cids.header_id, transaction_cids.cid, transaction_cids.mh_key, transaction_cids.dst,
				transaction_cids.src, transaction_cids.index, transaction_cids.tx_data, transaction_cids.tx_type
				FROM eth.transaction_cids
				INNER JOIN eth.header_cids ON (
						transaction_cids.header_id = header_cids.block_hash
						AND transaction_cids.block_number = header_cids.block_number
					)
				WHERE header_cids.block_hash = $%d`, id)
	args = append(args, headerID)
	id++
	if len(txFilter.Dst) > 0 {
		pgStr += fmt.Sprintf(` AND transaction_cids.dst = ANY($%d::VARCHAR(66)[])`, id)
		args = append(args, pq.Array(txFilter.Dst))
		id++
	}
	if len(txFilter.Src) > 0 {
		pgStr += fmt.Sprintf(` AND transaction_cids.src = ANY($%d::VARCHAR(66)[])`, id)
		args = append(args, pq.Array(txFilter.Src))
	}
	pgStr += ` ORDER BY transaction_cids.index`
	return results, tx.Select(&results, pgStr, args...)
}

func topicFilterCondition(id *int, topics [][]string, args []interface{}, pgStr string, first bool) (string, []interface{}) {
	for i, topicSet := range topics {
		if len(topicSet) == 0 {
			continue
		}

		if !first {
			pgStr += " AND"
		} else {
			first = false
		}
		pgStr += fmt.Sprintf(` eth.log_cids.topic%d = ANY ($%d)`, i, *id)
		args = append(args, pq.Array(topicSet))
		*id++
	}
	return pgStr, args
}

func logFilterCondition(id *int, pgStr string, args []interface{}, rctFilter ReceiptFilter) (string, []interface{}) {
	if len(rctFilter.LogAddresses) > 0 {
		pgStr += fmt.Sprintf(` AND eth.log_cids.address = ANY ($%d)`, *id)
		args = append(args, pq.Array(rctFilter.LogAddresses))
		*id++
	}

	// Filter on topics if there are any
	if hasTopics(rctFilter.Topics) {
		pgStr, args = topicFilterCondition(id, rctFilter.Topics, args, pgStr, false)
	}

	return pgStr, args
}

func receiptFilterConditions(id *int, pgStr string, args []interface{}, rctFilter ReceiptFilter, txHashes []string) (string, []interface{}) {
	rctCond := " AND (receipt_cids.tx_id = ANY ( "
	logQuery := "SELECT rct_id FROM eth.log_cids WHERE"
	if len(rctFilter.LogAddresses) > 0 {
		// Filter on log contract addresses if there are any
		pgStr += fmt.Sprintf(`%s %s eth.log_cids.address = ANY ($%d)`, rctCond, logQuery, *id)
		args = append(args, pq.Array(rctFilter.LogAddresses))
		*id++

		// Filter on topics if there are any
		if hasTopics(rctFilter.Topics) {
			pgStr, args = topicFilterCondition(id, rctFilter.Topics, args, pgStr, false)
		}

		pgStr += ")"

		// Filter on txHashes if there are any, and we are matching txs
		if rctFilter.MatchTxs && len(txHashes) > 0 {
			pgStr += fmt.Sprintf(` OR receipt_cids.tx_id = ANY($%d)`, *id)
			args = append(args, pq.Array(txHashes))
		}
		pgStr += ")"
	} else { // If there are no contract addresses to filter on
		// Filter on topics if there are any
		if hasTopics(rctFilter.Topics) {
			pgStr += rctCond + logQuery
			pgStr, args = topicFilterCondition(id, rctFilter.Topics, args, pgStr, true)
			pgStr += ")"
			// Filter on txHashes if there are any, and we are matching txs
			if rctFilter.MatchTxs && len(txHashes) > 0 {
				pgStr += fmt.Sprintf(` OR receipt_cids.tx_id = ANY($%d)`, *id)
				args = append(args, pq.Array(txHashes))
			}
			pgStr += ")"
		} else if rctFilter.MatchTxs && len(txHashes) > 0 {
			// If there are no contract addresses or topics to filter on,
			// Filter on txHashes if there are any, and we are matching txs
			pgStr += fmt.Sprintf(` AND receipt_cids.tx_id = ANY($%d)`, *id)
			args = append(args, pq.Array(txHashes))
		}
	}

	return pgStr, args
}

// RetrieveFilteredGQLLogs retrieves and returns all the log CIDs provided blockHash that conform to the provided
// filter parameters.
func (ecr *CIDRetriever) RetrieveFilteredGQLLogs(tx *sqlx.Tx, rctFilter ReceiptFilter, blockHash *common.Hash, blockNumber *big.Int) ([]LogResult, error) {
	log.Debug("retrieving log cids for receipt ids with block hash", blockHash.String())
	args := make([]interface{}, 0, 4)
	id := 1
	pgStr := `SELECT CAST(eth.log_cids.block_number as Text), eth.log_cids.header_id as block_hash,
			eth.log_cids.leaf_cid, eth.log_cids.index, eth.log_cids.rct_id, eth.log_cids.address,
			eth.log_cids.topic0, eth.log_cids.topic1, eth.log_cids.topic2, eth.log_cids.topic3, eth.log_cids.log_data,
			data, eth.receipt_cids.leaf_cid as cid, eth.receipt_cids.post_status, eth.receipt_cids.tx_id AS tx_hash
				FROM eth.log_cids, eth.receipt_cids, public.blocks
				WHERE eth.log_cids.rct_id = receipt_cids.tx_id
				AND eth.log_cids.header_id = receipt_cids.header_id
				AND eth.log_cids.block_number = receipt_cids.block_number
				AND log_cids.leaf_mh_key = blocks.key
				AND log_cids.block_number = blocks.block_number
				AND receipt_cids.header_id = $1`

	args = append(args, blockHash.String())
	id++

	if blockNumber != nil {
		pgStr += ` AND receipt_cids.block_number = $2`
		id++
		args = append(args, blockNumber.Int64())
	}

	pgStr, args = logFilterCondition(&id, pgStr, args, rctFilter)
	pgStr += ` ORDER BY log_cids.index`

	logCIDs := make([]LogResult, 0)
	err := tx.Select(&logCIDs, pgStr, args...)
	if err != nil {
		return nil, err
	}

	return logCIDs, nil
}

// RetrieveFilteredLog retrieves and returns all the log CIDs provided blockHeight or blockHash that conform to the provided
// filter parameters.
func (ecr *CIDRetriever) RetrieveFilteredLog(tx *sqlx.Tx, rctFilter ReceiptFilter, blockNumber int64, blockHash *common.Hash) ([]LogResult, error) {
	log.Debug("retrieving log cids for receipt ids")
	args := make([]interface{}, 0, 4)
	pgStr := `SELECT CAST(eth.log_cids.block_number as Text), eth.log_cids.leaf_cid, eth.log_cids.index, eth.log_cids.rct_id,
			eth.log_cids.address, eth.log_cids.topic0, eth.log_cids.topic1, eth.log_cids.topic2, eth.log_cids.topic3,
			eth.log_cids.log_data, eth.transaction_cids.tx_hash, eth.transaction_cids.index as txn_index,
			eth.receipt_cids.leaf_cid as cid, eth.receipt_cids.post_status, header_cids.block_hash
							FROM eth.log_cids, eth.receipt_cids, eth.transaction_cids, eth.header_cids
							WHERE eth.log_cids.rct_id = receipt_cids.tx_id
							AND eth.log_cids.header_id = eth.receipt_cids.header_id
							AND eth.log_cids.block_number = eth.receipt_cids.block_number
							AND receipt_cids.tx_id = transaction_cids.tx_hash
							AND receipt_cids.header_id = transaction_cids.header_id
							AND receipt_cids.block_number = transaction_cids.block_number
							AND transaction_cids.header_id = header_cids.block_hash
							AND transaction_cids.block_number = header_cids.block_number`
	id := 1
	if blockNumber > 0 {
		pgStr += fmt.Sprintf(` AND header_cids.block_number = $%d`, id)
		args = append(args, blockNumber)
		id++
	}
	if blockHash != nil {
		pgStr += fmt.Sprintf(` AND header_cids.block_hash = $%d`, id)
		args = append(args, blockHash.String())
		id++
	}

	pgStr, args = logFilterCondition(&id, pgStr, args, rctFilter)
	pgStr += ` ORDER BY log_cids.index`

	logCIDs := make([]LogResult, 0)
	err := tx.Select(&logCIDs, pgStr, args...)
	if err != nil {
		return nil, err
	}

	return logCIDs, nil
}

// RetrieveRctCIDs retrieves and returns all of the rct cids at the provided blockheight or block hash that conform to the provided
// filter parameters and correspond to the provided tx ids
func (ecr *CIDRetriever) RetrieveRctCIDs(tx *sqlx.Tx, rctFilter ReceiptFilter, blockNumber int64, blockHash string, txHashes []string) ([]models.ReceiptModel, error) {
	log.Debug("retrieving receipt cids for block ", blockNumber)
	args := make([]interface{}, 0, 5)
	pgStr := `SELECT CAST(receipt_cids.block_number as Text), receipt_cids.header_id, receipt_cids.tx_id,
			receipt_cids.leaf_cid, receipt_cids.leaf_mh_key, receipt_cids.contract, receipt_cids.contract_hash
 			FROM eth.receipt_cids, eth.transaction_cids, eth.header_cids
			WHERE receipt_cids.tx_id = transaction_cids.tx_hash
			AND receipt_cids.header_id = transaction_cids.header_id
			AND receipt_cids.block_number = transaction_cids.block_number
			AND transaction_cids.header_id = header_cids.block_hash
			AND transaction_cids.block_number = header_cids.block_number`
	id := 1
	if blockNumber > 0 {
		pgStr += fmt.Sprintf(` AND header_cids.block_number = $%d`, id)
		args = append(args, blockNumber)
		id++
	}
	if blockHash != "" {
		pgStr += fmt.Sprintf(` AND header_cids.block_hash = $%d`, id)
		args = append(args, blockHash)
		id++
	}

	pgStr, args = receiptFilterConditions(&id, pgStr, args, rctFilter, txHashes)

	pgStr += ` ORDER BY transaction_cids.index`
	receiptCIDs := make([]models.ReceiptModel, 0)
	return receiptCIDs, tx.Select(&receiptCIDs, pgStr, args...)
}

func hasTopics(topics [][]string) bool {
	for _, topicSet := range topics {
		if len(topicSet) > 0 {
			return true
		}
	}
	return false
}

// RetrieveStateCIDs retrieves and returns all of the state node cids at the provided header ID that conform to the provided filter parameters
func (ecr *CIDRetriever) RetrieveStateCIDs(tx *sqlx.Tx, stateFilter StateFilter, headerID string) ([]models.StateNodeModel, error) {
	log.Debug("retrieving state cids for header id ", headerID)
	args := make([]interface{}, 0, 2)
	pgStr := `SELECT CAST(state_cids.block_number as Text), state_cids.header_id,
			state_cids.state_leaf_key, state_cids.node_type, state_cids.cid, state_cids.mh_key, state_cids.state_path
			FROM eth.state_cids
			INNER JOIN eth.header_cids ON (
				state_cids.header_id = header_cids.block_hash
				AND state_cids.block_number = header_cids.block_number
			)
			WHERE header_cids.block_hash = $1`
	args = append(args, headerID)
	addrLen := len(stateFilter.Addresses)
	if addrLen > 0 {
		keys := make([]string, addrLen)
		for i, addr := range stateFilter.Addresses {
			keys[i] = crypto.Keccak256Hash(common.HexToAddress(addr).Bytes()).String()
		}
		pgStr += ` AND state_cids.state_leaf_key = ANY($2::VARCHAR(66)[])`
		args = append(args, pq.Array(keys))
	}
	if !stateFilter.IntermediateNodes {
		pgStr += ` AND state_cids.node_type = 2`
	}
	stateNodeCIDs := make([]models.StateNodeModel, 0)
	return stateNodeCIDs, tx.Select(&stateNodeCIDs, pgStr, args...)
}

// RetrieveStorageCIDs retrieves and returns all of the storage node cids at the provided header id that conform to the provided filter parameters
func (ecr *CIDRetriever) RetrieveStorageCIDs(tx *sqlx.Tx, storageFilter StorageFilter, headerID string) ([]models.StorageNodeWithStateKeyModel, error) {
	log.Debug("retrieving storage cids for header id ", headerID)
	args := make([]interface{}, 0, 3)
	pgStr := `SELECT CAST(storage_cids.block_number as Text), storage_cids.header_id, storage_cids.storage_leaf_key,
			storage_cids.node_type, storage_cids.cid, storage_cids.mh_key, storage_cids.storage_path, storage_cids.state_path,
			state_cids.state_leaf_key
 			FROM eth.storage_cids, eth.state_cids, eth.header_cids
			WHERE storage_cids.header_id = state_cids.header_id
			AND storage_cids.state_path = state_cids.state_path
			AND storage_cids.block_number = state_cids.block_number
			AND state_cids.header_id = header_cids.block_hash
			AND state_cids.block_number = header_cids.block_number
			AND header_cids.block_hash = $1`
	args = append(args, headerID)
	id := 2
	addrLen := len(storageFilter.Addresses)
	if addrLen > 0 {
		keys := make([]string, addrLen)
		for i, addr := range storageFilter.Addresses {
			keys[i] = crypto.Keccak256Hash(common.HexToAddress(addr).Bytes()).String()
		}
		pgStr += fmt.Sprintf(` AND state_cids.state_leaf_key = ANY($%d::VARCHAR(66)[])`, id)
		args = append(args, pq.Array(keys))
		id++
	}
	if len(storageFilter.StorageKeys) > 0 {
		pgStr += fmt.Sprintf(` AND storage_cids.storage_leaf_key = ANY($%d::VARCHAR(66)[])`, id)
		args = append(args, pq.Array(storageFilter.StorageKeys))
	}
	if !storageFilter.IntermediateNodes {
		pgStr += ` AND storage_cids.node_type = 2`
	}
	storageNodeCIDs := make([]models.StorageNodeWithStateKeyModel, 0)
	return storageNodeCIDs, tx.Select(&storageNodeCIDs, pgStr, args...)
}

// RetrieveBlockByHash returns all of the CIDs needed to compose an entire block, for a given block hash
func (ecr *CIDRetriever) RetrieveBlockByHash(blockHash common.Hash) (models.HeaderModel, []models.UncleModel, []models.TxModel, []models.ReceiptModel, error) {
	log.Debug("retrieving block cids for block hash ", blockHash.String())

	// Begin new db tx
	tx, err := ecr.db.Beginx()
	if err != nil {
		return models.HeaderModel{}, nil, nil, nil, err
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

	var headerCID models.HeaderModel
	headerCID, err = ecr.RetrieveHeaderCIDByHash(tx, blockHash)
	if err != nil {
		log.Error("header cid retrieval error")
		return models.HeaderModel{}, nil, nil, nil, err
	}
	blockNumber, err := strconv.ParseInt(headerCID.BlockNumber, 10, 64)
	if err != nil {
		return models.HeaderModel{}, nil, nil, nil, err
	}
	var uncleCIDs []models.UncleModel
	uncleCIDs, err = ecr.RetrieveUncleCIDsByHeaderID(tx, headerCID.BlockHash)
	if err != nil {
		log.Error("uncle cid retrieval error")
		return models.HeaderModel{}, nil, nil, nil, err
	}
	var txCIDs []models.TxModel
	txCIDs, err = ecr.RetrieveTxCIDsByHeaderID(tx, headerCID.BlockHash, blockNumber)
	if err != nil {
		log.Error("tx cid retrieval error")
		return models.HeaderModel{}, nil, nil, nil, err
	}
	txHashes := make([]string, len(txCIDs))
	for i, txCID := range txCIDs {
		txHashes[i] = txCID.TxHash
	}
	var rctCIDs []models.ReceiptModel
	rctCIDs, err = ecr.RetrieveReceiptCIDsByByHeaderIDAndTxIDs(tx, headerCID.BlockHash, txHashes, blockNumber)
	if err != nil {
		log.Error("rct cid retrieval error")
	}
	return headerCID, uncleCIDs, txCIDs, rctCIDs, err
}

// RetrieveBlockByNumber returns all of the CIDs needed to compose an entire block, for a given block number
func (ecr *CIDRetriever) RetrieveBlockByNumber(blockNumber int64) (models.HeaderModel, []models.UncleModel, []models.TxModel, []models.ReceiptModel, error) {
	log.Debug("retrieving block cids for block number ", blockNumber)

	// Begin new db tx
	tx, err := ecr.db.Beginx()
	if err != nil {
		return models.HeaderModel{}, nil, nil, nil, err
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

	var headerCID []models.HeaderModel
	headerCID, err = ecr.RetrieveHeaderCIDs(tx, blockNumber)
	if err != nil {
		log.Error("header cid retrieval error")
		return models.HeaderModel{}, nil, nil, nil, err
	}
	if len(headerCID) < 1 {
		return models.HeaderModel{}, nil, nil, nil, fmt.Errorf("header cid retrieval error, no header CIDs found at block %d", blockNumber)
	}
	var uncleCIDs []models.UncleModel
	uncleCIDs, err = ecr.RetrieveUncleCIDsByHeaderID(tx, headerCID[0].BlockHash)
	if err != nil {
		log.Error("uncle cid retrieval error")
		return models.HeaderModel{}, nil, nil, nil, err
	}
	var txCIDs []models.TxModel
	txCIDs, err = ecr.RetrieveTxCIDsByHeaderID(tx, headerCID[0].BlockHash, blockNumber)
	if err != nil {
		log.Error("tx cid retrieval error")
		return models.HeaderModel{}, nil, nil, nil, err
	}
	txHashes := make([]string, len(txCIDs))
	for i, txCID := range txCIDs {
		txHashes[i] = txCID.TxHash
	}
	var rctCIDs []models.ReceiptModel
	rctCIDs, err = ecr.RetrieveReceiptCIDsByByHeaderIDAndTxIDs(tx, headerCID[0].BlockHash, txHashes, blockNumber)
	if err != nil {
		log.Error("rct cid retrieval error")
	}
	return headerCID[0], uncleCIDs, txCIDs, rctCIDs, err
}

// RetrieveHeaderCIDByHash returns the header for the given block hash
func (ecr *CIDRetriever) RetrieveHeaderCIDByHash(tx *sqlx.Tx, blockHash common.Hash) (models.HeaderModel, error) {
	log.Debug("retrieving header cids for block hash ", blockHash.String())
	pgStr := `SELECT block_hash, CAST(block_number as Text), parent_hash, cid, mh_key, CAST(td as Text),
			state_root, uncle_root, tx_root, receipt_root, bloom, timestamp FROM eth.header_cids
			WHERE block_hash = $1`
	var headerCID models.HeaderModel
	return headerCID, tx.Get(&headerCID, pgStr, blockHash.String())
}

// RetrieveTxCIDsByHeaderID retrieves all tx CIDs for the given header id
func (ecr *CIDRetriever) RetrieveTxCIDsByHeaderID(tx *sqlx.Tx, headerID string, blockNumber int64) ([]models.TxModel, error) {
	log.Debug("retrieving tx cids for block id ", headerID)
	pgStr := `SELECT CAST(block_number as Text), header_id, index, tx_hash, cid, mh_key,
			dst, src, tx_data, tx_type, value
			FROM eth.transaction_cids
			WHERE header_id = $1 AND block_number = $2
			ORDER BY index`
	var txCIDs []models.TxModel
	return txCIDs, tx.Select(&txCIDs, pgStr, headerID, blockNumber)
}

// RetrieveReceiptCIDsByByHeaderIDAndTxIDs retrieves receipt CIDs by their associated tx IDs for the given header id
func (ecr *CIDRetriever) RetrieveReceiptCIDsByByHeaderIDAndTxIDs(tx *sqlx.Tx, headerID string, txHashes []string, blockNumber int64) ([]models.ReceiptModel, error) {
	log.Debugf("retrieving receipt cids for tx hashes %v", txHashes)
	pgStr := `SELECT CAST(receipt_cids.block_number as Text), receipt_cids.header_id, receipt_cids.tx_id, receipt_cids.leaf_cid,
			receipt_cids.leaf_mh_key, receipt_cids.contract, receipt_cids.contract_hash
			FROM eth.receipt_cids, eth.transaction_cids
			WHERE tx_id = ANY($2)
			AND receipt_cids.tx_id = transaction_cids.tx_hash
			AND receipt_cids.header_id = transaction_cids.header_id
			AND receipt_cids.block_number = transaction_cids.block_number
			AND transaction_cids.header_id = $1
			AND transaction_cids.block_number = $3
			ORDER BY transaction_cids.index`
	var rctCIDs []models.ReceiptModel
	return rctCIDs, tx.Select(&rctCIDs, pgStr, headerID, pq.Array(txHashes), blockNumber)
}

// RetrieveHeaderAndTxCIDsByBlockNumber retrieves header CIDs and their associated tx CIDs by block number
func (ecr *CIDRetriever) RetrieveHeaderAndTxCIDsByBlockNumber(blockNumber int64) ([]HeaderCIDRecord, error) {
	log.Debug("retrieving header cids and tx cids for block number ", blockNumber)

	var headerCIDs []HeaderCIDRecord

	// https://github.com/go-gorm/gorm/issues/4083#issuecomment-778883283
	// Will use join for TransactionCIDs once preload for 1:N is supported.
	err := ecr.gormDB.Preload("TransactionCIDs", func(tx *gorm.DB) *gorm.DB {
		return tx.Select("cid", "tx_hash", "index", "src", "dst", "header_id", "block_number")
	}).Joins("IPLD").Find(&headerCIDs, "header_cids.block_number = ?", blockNumber).Error

	if err != nil {
		log.Error("header cid retrieval error")
		return nil, err
	}

	return headerCIDs, nil
}

// RetrieveHeaderAndTxCIDsByBlockHash retrieves header CID and their associated tx CIDs by block hash (and optionally block number)
func (ecr *CIDRetriever) RetrieveHeaderAndTxCIDsByBlockHash(blockHash common.Hash, blockNumber *big.Int) (HeaderCIDRecord, error) {
	log.Debug("retrieving header cid and tx cids for block hash ", blockHash.String())

	var headerCIDs []HeaderCIDRecord

	conditions := map[string]interface{}{"block_hash": blockHash.String()}
	if blockNumber != nil {
		conditions["header_cids.block_number"] = blockNumber.Int64()
	}

	// https://github.com/go-gorm/gorm/issues/4083#issuecomment-778883283
	// Will use join for TransactionCIDs once preload for 1:N is supported.
	err := ecr.gormDB.Preload("TransactionCIDs", func(tx *gorm.DB) *gorm.DB {
		return tx.Select("cid", "tx_hash", "index", "src", "dst", "header_id", "block_number")
	}).Joins("IPLD").Find(&headerCIDs, conditions).Error

	if err != nil {
		log.Error("header cid retrieval error")
		return HeaderCIDRecord{}, err
	}

	if len(headerCIDs) == 0 {
		return HeaderCIDRecord{}, errHeaderHashNotFound
	} else if len(headerCIDs) > 1 {
		return HeaderCIDRecord{}, errMultipleHeadersForHash
	}

	return headerCIDs[0], nil
}

// RetrieveTxCIDByHash returns the tx for the given tx hash (and optionally block number)
func (ecr *CIDRetriever) RetrieveTxCIDByHash(txHash string, blockNumber *big.Int) (TransactionCIDRecord, error) {
	log.Debug("retrieving tx cid for tx hash ", txHash)

	var txCIDs []TransactionCIDRecord

	var err error
	if blockNumber != nil {
		err = ecr.gormDB.Joins("IPLD").Find(&txCIDs, "tx_hash = ? AND transaction_cids.header_id = (SELECT canonical_header_hash(transaction_cids.block_number)) AND transaction_cids.block_number = ?", txHash, blockNumber.Int64()).Error
	} else {
		err = ecr.gormDB.Joins("IPLD").Find(&txCIDs, "tx_hash = ? AND transaction_cids.header_id = (SELECT canonical_header_hash(transaction_cids.block_number))", txHash).Error
	}
	if err != nil {
		log.Error("tx retrieval error")
		return TransactionCIDRecord{}, err
	}

	if len(txCIDs) == 0 {
		return TransactionCIDRecord{}, errTxHashNotFound
	} else if len(txCIDs) > 1 {
		// a transaction can be part of a only one canonical block
		return TransactionCIDRecord{}, errTxHashInMultipleBlocks
	}

	return txCIDs[0], nil
}
