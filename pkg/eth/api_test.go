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

package eth_test

import (
	"context"
	"strconv"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	eth2 "github.com/vulcanize/ipld-eth-indexer/pkg/eth"
	"github.com/vulcanize/ipld-eth-indexer/pkg/postgres"

	"github.com/vulcanize/ipld-eth-server/pkg/eth"
	"github.com/vulcanize/ipld-eth-server/pkg/eth/test_helpers"
	"github.com/vulcanize/ipld-eth-server/pkg/shared"
)

var (
	expectedBlock = map[string]interface{}{
		"number":           (*hexutil.Big)(test_helpers.MockBlock.Number()),
		"hash":             test_helpers.MockBlock.Hash(),
		"parentHash":       test_helpers.MockBlock.ParentHash(),
		"nonce":            test_helpers.MockBlock.Header().Nonce,
		"mixHash":          test_helpers.MockBlock.MixDigest(),
		"sha3Uncles":       test_helpers.MockBlock.UncleHash(),
		"logsBloom":        test_helpers.MockBlock.Bloom(),
		"stateRoot":        test_helpers.MockBlock.Root(),
		"miner":            test_helpers.MockBlock.Coinbase(),
		"difficulty":       (*hexutil.Big)(test_helpers.MockBlock.Difficulty()),
		"extraData":        hexutil.Bytes(test_helpers.MockBlock.Header().Extra),
		"gasLimit":         hexutil.Uint64(test_helpers.MockBlock.GasLimit()),
		"gasUsed":          hexutil.Uint64(test_helpers.MockBlock.GasUsed()),
		"timestamp":        hexutil.Uint64(test_helpers.MockBlock.Time()),
		"transactionsRoot": test_helpers.MockBlock.TxHash(),
		"receiptsRoot":     test_helpers.MockBlock.ReceiptHash(),
		"totalDifficulty":  (*hexutil.Big)(test_helpers.MockBlock.Difficulty()),
		"size":             hexutil.Uint64(test_helpers.MockBlock.Size()),
	}
	expectedHeader = map[string]interface{}{
		"number":           (*hexutil.Big)(test_helpers.MockBlock.Header().Number),
		"hash":             test_helpers.MockBlock.Header().Hash(),
		"parentHash":       test_helpers.MockBlock.Header().ParentHash,
		"nonce":            test_helpers.MockBlock.Header().Nonce,
		"mixHash":          test_helpers.MockBlock.Header().MixDigest,
		"sha3Uncles":       test_helpers.MockBlock.Header().UncleHash,
		"logsBloom":        test_helpers.MockBlock.Header().Bloom,
		"stateRoot":        test_helpers.MockBlock.Header().Root,
		"miner":            test_helpers.MockBlock.Header().Coinbase,
		"difficulty":       (*hexutil.Big)(test_helpers.MockBlock.Header().Difficulty),
		"extraData":        hexutil.Bytes(test_helpers.MockBlock.Header().Extra),
		"size":             hexutil.Uint64(test_helpers.MockBlock.Header().Size()),
		"gasLimit":         hexutil.Uint64(test_helpers.MockBlock.Header().GasLimit),
		"gasUsed":          hexutil.Uint64(test_helpers.MockBlock.Header().GasUsed),
		"timestamp":        hexutil.Uint64(test_helpers.MockBlock.Header().Time),
		"transactionsRoot": test_helpers.MockBlock.Header().TxHash,
		"receiptsRoot":     test_helpers.MockBlock.Header().ReceiptHash,
		"totalDifficulty":  (*hexutil.Big)(test_helpers.MockBlock.Header().Difficulty),
	}
	expectedUncle1 = map[string]interface{}{
		"number":           (*hexutil.Big)(test_helpers.MockUncles[0].Number),
		"hash":             test_helpers.MockUncles[0].Hash(),
		"parentHash":       test_helpers.MockUncles[0].ParentHash,
		"nonce":            test_helpers.MockUncles[0].Nonce,
		"mixHash":          test_helpers.MockUncles[0].MixDigest,
		"sha3Uncles":       test_helpers.MockUncles[0].UncleHash,
		"logsBloom":        test_helpers.MockUncles[0].Bloom,
		"stateRoot":        test_helpers.MockUncles[0].Root,
		"miner":            test_helpers.MockUncles[0].Coinbase,
		"difficulty":       (*hexutil.Big)(test_helpers.MockUncles[0].Difficulty),
		"extraData":        hexutil.Bytes(test_helpers.MockUncles[0].Extra),
		"size":             hexutil.Uint64(types.NewBlockWithHeader(test_helpers.MockUncles[0]).Size()),
		"gasLimit":         hexutil.Uint64(test_helpers.MockUncles[0].GasLimit),
		"gasUsed":          hexutil.Uint64(test_helpers.MockUncles[0].GasUsed),
		"timestamp":        hexutil.Uint64(test_helpers.MockUncles[0].Time),
		"transactionsRoot": test_helpers.MockUncles[0].TxHash,
		"receiptsRoot":     test_helpers.MockUncles[0].ReceiptHash,
		"uncles":           []common.Hash{},
	}
	expectedUncle2 = map[string]interface{}{
		"number":           (*hexutil.Big)(test_helpers.MockUncles[1].Number),
		"hash":             test_helpers.MockUncles[1].Hash(),
		"parentHash":       test_helpers.MockUncles[1].ParentHash,
		"nonce":            test_helpers.MockUncles[1].Nonce,
		"mixHash":          test_helpers.MockUncles[1].MixDigest,
		"sha3Uncles":       test_helpers.MockUncles[1].UncleHash,
		"logsBloom":        test_helpers.MockUncles[1].Bloom,
		"stateRoot":        test_helpers.MockUncles[1].Root,
		"miner":            test_helpers.MockUncles[1].Coinbase,
		"difficulty":       (*hexutil.Big)(test_helpers.MockUncles[1].Difficulty),
		"extraData":        hexutil.Bytes(test_helpers.MockUncles[1].Extra),
		"size":             hexutil.Uint64(types.NewBlockWithHeader(test_helpers.MockUncles[1]).Size()),
		"gasLimit":         hexutil.Uint64(test_helpers.MockUncles[1].GasLimit),
		"gasUsed":          hexutil.Uint64(test_helpers.MockUncles[1].GasUsed),
		"timestamp":        hexutil.Uint64(test_helpers.MockUncles[1].Time),
		"transactionsRoot": test_helpers.MockUncles[1].TxHash,
		"receiptsRoot":     test_helpers.MockUncles[1].ReceiptHash,
		"uncles":           []common.Hash{},
	}
	expectedTransaction = eth.NewRPCTransaction(test_helpers.MockTransactions[0], test_helpers.MockBlock.Hash(), test_helpers.MockBlock.NumberU64(), 0)
)

var _ = Describe("API", func() {
	var (
		db                *postgres.DB
		indexAndPublisher *eth2.IPLDPublisher
		backend           *eth.Backend
		api               *eth.PublicEthAPI
	)
	BeforeEach(func() {
		var err error
		db, err = shared.SetupDB()
		Expect(err).ToNot(HaveOccurred())
		indexAndPublisher = eth2.NewIPLDPublisher(db)
		backend, err = eth.NewEthBackend(db, &eth.Config{})
		Expect(err).ToNot(HaveOccurred())
		api = eth.NewPublicEthAPI(backend, nil)
		err = indexAndPublisher.Publish(test_helpers.MockConvertedPayload)
		Expect(err).ToNot(HaveOccurred())
		uncles := test_helpers.MockBlock.Uncles()
		uncleHashes := make([]common.Hash, len(uncles))
		for i, uncle := range uncles {
			uncleHashes[i] = uncle.Hash()
		}
		expectedBlock["uncles"] = uncleHashes
	})
	AfterEach(func() {
		eth.TearDownDB(db)
	})
	/*

	   Headers and blocks

	*/
	Describe("GetHeaderByHash", func() {
		It("Retrieves a header by hash", func() {
			hash := test_helpers.MockBlock.Header().Hash()
			header := api.GetHeaderByHash(context.Background(), hash)
			Expect(header).To(Equal(expectedHeader))
		})
	})

	Describe("GetHeaderByNumber", func() {
		It("Retrieves a header by number", func() {
			number, err := strconv.ParseInt(test_helpers.BlockNumber.String(), 10, 64)
			Expect(err).ToNot(HaveOccurred())
			header, err := api.GetHeaderByNumber(context.Background(), rpc.BlockNumber(number))
			Expect(err).ToNot(HaveOccurred())
			Expect(header).To(Equal(expectedHeader))
		})

		It("Throws an error if a header cannot be found", func() {
			number, err := strconv.ParseInt(test_helpers.BlockNumber.String(), 10, 64)
			Expect(err).ToNot(HaveOccurred())
			header, err := api.GetHeaderByNumber(context.Background(), rpc.BlockNumber(number+1))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("sql: no rows in result set"))
			Expect(header).To(BeNil())
			_, err = api.B.DB.Beginx()
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("BlockNumber", func() {
		It("Retrieves the head block number", func() {
			bn := api.BlockNumber()
			ubn := (uint64)(bn)
			subn := strconv.FormatUint(ubn, 10)
			Expect(subn).To(Equal(test_helpers.BlockNumber.String()))
		})
	})

	Describe("GetBlockByNumber", func() {
		It("Retrieves a block by number", func() {
			// without full txs
			number, err := strconv.ParseInt(test_helpers.BlockNumber.String(), 10, 64)
			Expect(err).ToNot(HaveOccurred())
			block, err := api.GetBlockByNumber(context.Background(), rpc.BlockNumber(number), false)
			Expect(err).ToNot(HaveOccurred())
			transactionHashes := make([]interface{}, len(test_helpers.MockBlock.Transactions()))
			for i, trx := range test_helpers.MockBlock.Transactions() {
				transactionHashes[i] = trx.Hash()
			}
			expectedBlock["transactions"] = transactionHashes
			for key, val := range expectedBlock {
				Expect(val).To(Equal(block[key]))
			}
			// with full txs
			block, err = api.GetBlockByNumber(context.Background(), rpc.BlockNumber(number), true)
			Expect(err).ToNot(HaveOccurred())
			transactions := make([]interface{}, len(test_helpers.MockBlock.Transactions()))
			for i, trx := range test_helpers.MockBlock.Transactions() {
				transactions[i] = eth.NewRPCTransactionFromBlockHash(test_helpers.MockBlock, trx.Hash())
			}
			expectedBlock["transactions"] = transactions
			for key, val := range expectedBlock {
				Expect(val).To(Equal(block[key]))
			}
		})
	})

	Describe("GetBlockByHash", func() {
		It("Retrieves a block by hash", func() {
			// without full txs
			block, err := api.GetBlockByHash(context.Background(), test_helpers.MockBlock.Hash(), false)
			Expect(err).ToNot(HaveOccurred())
			transactionHashes := make([]interface{}, len(test_helpers.MockBlock.Transactions()))
			for i, trx := range test_helpers.MockBlock.Transactions() {
				transactionHashes[i] = trx.Hash()
			}
			expectedBlock["transactions"] = transactionHashes
			for key, val := range expectedBlock {
				Expect(val).To(Equal(block[key]))
			}
			// with full txs
			block, err = api.GetBlockByHash(context.Background(), test_helpers.MockBlock.Hash(), true)
			Expect(err).ToNot(HaveOccurred())
			transactions := make([]interface{}, len(test_helpers.MockBlock.Transactions()))
			for i, trx := range test_helpers.MockBlock.Transactions() {
				transactions[i] = eth.NewRPCTransactionFromBlockHash(test_helpers.MockBlock, trx.Hash())
			}
			expectedBlock["transactions"] = transactions
			for key, val := range expectedBlock {
				Expect(val).To(Equal(block[key]))
			}
		})
	})

	/*

	   Uncles

	*/

	Describe("GetUncleByBlockNumberAndIndex", func() {
		It("Retrieves the uncle at the provided index in the canoncial block with the provided hash", func() {
			number, err := strconv.ParseInt(test_helpers.BlockNumber.String(), 10, 64)
			Expect(err).ToNot(HaveOccurred())
			uncle1, err := api.GetUncleByBlockNumberAndIndex(context.Background(), rpc.BlockNumber(number), 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(uncle1).To(Equal(expectedUncle1))
			uncle2, err := api.GetUncleByBlockNumberAndIndex(context.Background(), rpc.BlockNumber(number), 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(uncle2).To(Equal(expectedUncle2))
		})
	})

	Describe("GetUncleByBlockHashAndIndex", func() {
		It("Retrieves the uncle at the provided index in the block with the provided hash", func() {
			hash := test_helpers.MockBlock.Header().Hash()
			uncle1, err := api.GetUncleByBlockHashAndIndex(context.Background(), hash, 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(uncle1).To(Equal(expectedUncle1))
			uncle2, err := api.GetUncleByBlockHashAndIndex(context.Background(), hash, 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(uncle2).To(Equal(expectedUncle2))
		})
	})

	Describe("GetUncleCountByBlockNumber", func() {
		It("Retrieves the number of uncles for the canonical block with the provided number", func() {
			number, err := strconv.ParseInt(test_helpers.BlockNumber.String(), 10, 64)
			Expect(err).ToNot(HaveOccurred())
			count := api.GetUncleCountByBlockNumber(context.Background(), rpc.BlockNumber(number))
			Expect(uint64(*count)).To(Equal(uint64(2)))
		})
	})

	Describe("GetUncleCountByBlockHash", func() {
		It("Retrieves the number of uncles for the block with the provided hash", func() {
			hash := test_helpers.MockBlock.Header().Hash()
			count := api.GetUncleCountByBlockHash(context.Background(), hash)
			Expect(uint64(*count)).To(Equal(uint64(2)))
		})
	})

	/*

	   Transactions

	*/

	Describe("GetTransactionCount", func() {
		It("Retrieves the number of transactions the given address has sent for the given block number or block hash", func() {

		})
	})

	Describe("GetBlockTransactionCountByNumber", func() {
		It("Retrieves the number of transactions in the canonical block with the provided number", func() {

		})
	})

	Describe("GetBlockTransactionCountByHash", func() {
		It("Retrieves the number of transactions in the block with the provided hash ", func() {

		})
	})

	Describe("GetTransactionByBlockNumberAndIndex", func() {
		It("Retrieves the tx with the provided index in the canonical block with the provided block number", func() {

		})
	})

	Describe("GetTransactionByBlockHashAndIndex", func() {
		It("Retrieves the tx with the provided index in the block with the provided hash", func() {

		})
	})

	Describe("GetRawTransactionByBlockNumberAndIndex", func() {
		It("Retrieves the raw tx with the provided index in the canonical block with the provided block number", func() {

		})
	})

	Describe("GetRawTransactionByBlockHashAndIndex", func() {
		It("Retrieves the raw tx with the provided index in the block with the provided hash", func() {

		})
	})

	Describe("GetTransactionByHash", func() {
		It("Retrieves a transaction by hash", func() {
			hash := test_helpers.MockTransactions[0].Hash()
			tx, err := api.GetTransactionByHash(context.Background(), hash)
			Expect(err).ToNot(HaveOccurred())
			Expect(tx).To(Equal(expectedTransaction))
		})
	})

	Describe("GetRawTransactionByHash", func() {
		It("Retrieves a raw transaction by hash", func() {

		})
	})

	/*

	   Receipts and logs

	*/

	Describe("GetTransactionReceipt", func() {
		It("Retrieves a receipt by tx hash", func() {

		})
	})

	Describe("GetLogs", func() {
		It("Retrieves receipt logs that match the provided topics within the provided range", func() {
			crit := ethereum.FilterQuery{
				Topics: [][]common.Hash{
					{
						common.HexToHash("0x04"),
					},
				},
				FromBlock: test_helpers.MockBlock.Number(),
				ToBlock:   test_helpers.MockBlock.Number(),
			}
			logs, err := api.GetLogs(context.Background(), crit)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(logs)).To(Equal(1))
			Expect(logs).To(Equal([]*types.Log{test_helpers.MockLog1}))

			crit = ethereum.FilterQuery{
				Topics: [][]common.Hash{
					{
						common.HexToHash("0x04"),
						common.HexToHash("0x05"),
					},
				},
				FromBlock: test_helpers.MockBlock.Number(),
				ToBlock:   test_helpers.MockBlock.Number(),
			}
			logs, err = api.GetLogs(context.Background(), crit)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(logs)).To(Equal(2))
			Expect(logs).To(Equal([]*types.Log{test_helpers.MockLog1, test_helpers.MockLog2}))

			crit = ethereum.FilterQuery{
				Topics: [][]common.Hash{
					{
						common.HexToHash("0x04"),
						common.HexToHash("0x06"),
					},
				},
				FromBlock: test_helpers.MockBlock.Number(),
				ToBlock:   test_helpers.MockBlock.Number(),
			}
			logs, err = api.GetLogs(context.Background(), crit)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(logs)).To(Equal(1))
			Expect(logs).To(Equal([]*types.Log{test_helpers.MockLog1}))

			crit = ethereum.FilterQuery{
				Topics: [][]common.Hash{
					{
						common.HexToHash("0x04"),
					},
					{
						common.HexToHash("0x07"),
					},
				},
				FromBlock: test_helpers.MockBlock.Number(),
				ToBlock:   test_helpers.MockBlock.Number(),
			}
			logs, err = api.GetLogs(context.Background(), crit)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(logs)).To(Equal(0))

			crit = ethereum.FilterQuery{
				Topics: [][]common.Hash{
					{
						common.HexToHash("0x04"),
					},
					{
						common.HexToHash("0x06"),
					},
				},
				FromBlock: test_helpers.MockBlock.Number(),
				ToBlock:   test_helpers.MockBlock.Number(),
			}
			logs, err = api.GetLogs(context.Background(), crit)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(logs)).To(Equal(1))
			Expect(logs).To(Equal([]*types.Log{test_helpers.MockLog1}))

			crit = ethereum.FilterQuery{
				Topics: [][]common.Hash{
					{
						common.HexToHash("0x05"),
					},
					{
						common.HexToHash("0x07"),
					},
				},
				FromBlock: test_helpers.MockBlock.Number(),
				ToBlock:   test_helpers.MockBlock.Number(),
			}
			logs, err = api.GetLogs(context.Background(), crit)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(logs)).To(Equal(1))
			Expect(logs).To(Equal([]*types.Log{test_helpers.MockLog2}))

			crit = ethereum.FilterQuery{
				Topics: [][]common.Hash{
					{
						common.HexToHash("0x05"),
					},
					{
						common.HexToHash("0x06"),
						common.HexToHash("0x07"),
					},
				},
				FromBlock: test_helpers.MockBlock.Number(),
				ToBlock:   test_helpers.MockBlock.Number(),
			}
			logs, err = api.GetLogs(context.Background(), crit)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(logs)).To(Equal(1))
			Expect(logs).To(Equal([]*types.Log{test_helpers.MockLog2}))

			crit = ethereum.FilterQuery{
				Topics: [][]common.Hash{
					{
						common.HexToHash("0x04"),
						common.HexToHash("0x05"),
					},
					{
						common.HexToHash("0x06"),
						common.HexToHash("0x07"),
					},
				},
				FromBlock: test_helpers.MockBlock.Number(),
				ToBlock:   test_helpers.MockBlock.Number(),
			}
			logs, err = api.GetLogs(context.Background(), crit)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(logs)).To(Equal(2))
			Expect(logs).To(Equal([]*types.Log{test_helpers.MockLog1, test_helpers.MockLog2}))

			crit = ethereum.FilterQuery{
				Topics: [][]common.Hash{
					{},
					{
						common.HexToHash("0x07"),
					},
				},
				FromBlock: test_helpers.MockBlock.Number(),
				ToBlock:   test_helpers.MockBlock.Number(),
			}
			logs, err = api.GetLogs(context.Background(), crit)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(logs)).To(Equal(1))
			Expect(logs).To(Equal([]*types.Log{test_helpers.MockLog2}))

			crit = ethereum.FilterQuery{
				Topics: [][]common.Hash{
					{},
					{
						common.HexToHash("0x06"),
					},
				},
				FromBlock: test_helpers.MockBlock.Number(),
				ToBlock:   test_helpers.MockBlock.Number(),
			}
			logs, err = api.GetLogs(context.Background(), crit)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(logs)).To(Equal(1))
			Expect(logs).To(Equal([]*types.Log{test_helpers.MockLog1}))

			crit = ethereum.FilterQuery{
				Topics:    [][]common.Hash{},
				FromBlock: test_helpers.MockBlock.Number(),
				ToBlock:   test_helpers.MockBlock.Number(),
			}
			logs, err = api.GetLogs(context.Background(), crit)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(logs)).To(Equal(2))
			Expect(logs).To(Equal([]*types.Log{test_helpers.MockLog1, test_helpers.MockLog2}))
		})

		It("Uses the provided blockhash if one is provided", func() {
			hash := test_helpers.MockBlock.Hash()
			crit := ethereum.FilterQuery{
				BlockHash: &hash,
				Topics: [][]common.Hash{
					{},
					{
						common.HexToHash("0x06"),
					},
				},
			}
			logs, err := api.GetLogs(context.Background(), crit)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(logs)).To(Equal(1))
			Expect(logs).To(Equal([]*types.Log{test_helpers.MockLog1}))

			crit = ethereum.FilterQuery{
				BlockHash: &hash,
				Topics: [][]common.Hash{
					{
						common.HexToHash("0x04"),
					},
					{
						common.HexToHash("0x06"),
					},
				},
			}
			logs, err = api.GetLogs(context.Background(), crit)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(logs)).To(Equal(1))
			Expect(logs).To(Equal([]*types.Log{test_helpers.MockLog1}))

			crit = ethereum.FilterQuery{
				BlockHash: &hash,
				Topics: [][]common.Hash{
					{},
					{
						common.HexToHash("0x07"),
					},
				},
			}
			logs, err = api.GetLogs(context.Background(), crit)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(logs)).To(Equal(1))
			Expect(logs).To(Equal([]*types.Log{test_helpers.MockLog2}))

			crit = ethereum.FilterQuery{
				BlockHash: &hash,
				Topics: [][]common.Hash{
					{
						common.HexToHash("0x05"),
					},
					{
						common.HexToHash("0x07"),
					},
				},
			}
			logs, err = api.GetLogs(context.Background(), crit)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(logs)).To(Equal(1))
			Expect(logs).To(Equal([]*types.Log{test_helpers.MockLog2}))

			crit = ethereum.FilterQuery{
				BlockHash: &hash,
				Topics: [][]common.Hash{
					{
						common.HexToHash("0x04"),
					},
					{
						common.HexToHash("0x07"),
					},
				},
			}
			logs, err = api.GetLogs(context.Background(), crit)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(logs)).To(Equal(0))

			crit = ethereum.FilterQuery{
				BlockHash: &hash,
				Topics: [][]common.Hash{
					{
						common.HexToHash("0x04"),
						common.HexToHash("0x05"),
					},
					{
						common.HexToHash("0x07"),
					},
				},
			}
			logs, err = api.GetLogs(context.Background(), crit)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(logs)).To(Equal(1))
			Expect(logs).To(Equal([]*types.Log{test_helpers.MockLog2}))

			crit = ethereum.FilterQuery{
				BlockHash: &hash,
				Topics: [][]common.Hash{
					{
						common.HexToHash("0x04"),
						common.HexToHash("0x05"),
					},
				},
			}
			logs, err = api.GetLogs(context.Background(), crit)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(logs)).To(Equal(2))
			Expect(logs).To(Equal([]*types.Log{test_helpers.MockLog1, test_helpers.MockLog2}))

			crit = ethereum.FilterQuery{
				BlockHash: &hash,
				Topics: [][]common.Hash{
					{
						common.HexToHash("0x04"),
						common.HexToHash("0x05"),
					},
					{
						common.HexToHash("0x06"),
						common.HexToHash("0x07"),
					},
				},
			}
			logs, err = api.GetLogs(context.Background(), crit)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(logs)).To(Equal(2))
			Expect(logs).To(Equal([]*types.Log{test_helpers.MockLog1, test_helpers.MockLog2}))

			crit = ethereum.FilterQuery{
				BlockHash: &hash,
				Topics:    [][]common.Hash{},
			}
			logs, err = api.GetLogs(context.Background(), crit)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(logs)).To(Equal(2))
			Expect(logs).To(Equal([]*types.Log{test_helpers.MockLog1, test_helpers.MockLog2}))
		})

		It("Filters on contract address if any are provided", func() {
			hash := test_helpers.MockBlock.Hash()
			crit := ethereum.FilterQuery{
				BlockHash: &hash,
				Addresses: []common.Address{
					test_helpers.Address,
				},
				Topics: [][]common.Hash{
					{
						common.HexToHash("0x04"),
						common.HexToHash("0x05"),
					},
					{
						common.HexToHash("0x06"),
						common.HexToHash("0x07"),
					},
				},
			}
			logs, err := api.GetLogs(context.Background(), crit)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(logs)).To(Equal(1))
			Expect(logs).To(Equal([]*types.Log{test_helpers.MockLog1}))

			hash = test_helpers.MockBlock.Hash()
			crit = ethereum.FilterQuery{
				BlockHash: &hash,
				Addresses: []common.Address{
					test_helpers.Address,
					test_helpers.AnotherAddress,
				},
				Topics: [][]common.Hash{
					{
						common.HexToHash("0x04"),
						common.HexToHash("0x05"),
					},
					{
						common.HexToHash("0x06"),
						common.HexToHash("0x07"),
					},
				},
			}
			logs, err = api.GetLogs(context.Background(), crit)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(logs)).To(Equal(2))
			Expect(logs).To(Equal([]*types.Log{test_helpers.MockLog1, test_helpers.MockLog2}))

			hash = test_helpers.MockBlock.Hash()
			crit = ethereum.FilterQuery{
				BlockHash: &hash,
				Addresses: []common.Address{
					test_helpers.Address,
					test_helpers.AnotherAddress,
				},
			}
			logs, err = api.GetLogs(context.Background(), crit)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(logs)).To(Equal(2))
			Expect(logs).To(Equal([]*types.Log{test_helpers.MockLog1, test_helpers.MockLog2}))
		})
	})

	/*

	   State and storage

	*/

	Describe("GetBalance", func() {
		It("Retrieves the eth balance for the provided account address at the block with the provided hash or number", func() {

		})
	})

	Describe("GetStorageAt", func() {
		It("Retrieves the storage value at the provided contract address and storage leaf key at the block with the provided hash or number", func() {

		})
	})

	Describe("GetCode", func() {
		It("Retrieves the code for the provided contract address at the block with the provied hash or number", func() {

		})
	})

	Describe("GetProof", func() {
		It("Retrieves the Merkle-proof for a given account and optionally some storage keys at the block with the provided hash or number", func() {

		})
	})

})
