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

package graphql_test

import (
	"context"
	"fmt"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/statediff"
	"github.com/ethereum/go-ethereum/statediff/indexer/models"
	sdtypes "github.com/ethereum/go-ethereum/statediff/types"
	"github.com/jmoiron/sqlx"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vulcanize/ipld-eth-server/v3/pkg/eth"
	"github.com/vulcanize/ipld-eth-server/v3/pkg/eth/test_helpers"
	"github.com/vulcanize/ipld-eth-server/v3/pkg/graphql"
	"github.com/vulcanize/ipld-eth-server/v3/pkg/shared"
	ethServerShared "github.com/vulcanize/ipld-eth-server/v3/pkg/shared"
)

var _ = Describe("GraphQL", func() {
	const (
		gqlEndPoint = "127.0.0.1:8083"
	)
	var (
		randomAddr      = common.HexToAddress("0x1C3ab14BBaD3D99F4203bd7a11aCB94882050E6f")
		randomHash      = crypto.Keccak256Hash(randomAddr.Bytes())
		blocks          []*types.Block
		receipts        []types.Receipts
		chain           *core.BlockChain
		db              *sqlx.DB
		blockHashes     []common.Hash
		backend         *eth.Backend
		graphQLServer   *graphql.Service
		chainConfig     = params.TestChainConfig
		mockTD          = big.NewInt(1337)
		client          = graphql.NewClient(fmt.Sprintf("http://%s/graphql", gqlEndPoint))
		ctx             = context.Background()
		blockHash       common.Hash
		contractAddress common.Address
	)

	It("test init", func() {
		var err error
		db = shared.SetupDB()
		transformer := shared.SetupTestStateDiffIndexer(ctx, chainConfig, test_helpers.Genesis.Hash())

		backend, err = eth.NewEthBackend(db, &eth.Config{
			ChainConfig: chainConfig,
			VMConfig:    vm.Config{},
			RPCGasCap:   big.NewInt(10000000000),
			GroupCacheConfig: &ethServerShared.GroupCacheConfig{
				StateDB: ethServerShared.GroupConfig{
					Name:                   "graphql_test",
					CacheSizeInMB:          8,
					CacheExpiryInMins:      60,
					LogStatsIntervalInSecs: 0,
				},
			},
		})
		Expect(err).ToNot(HaveOccurred())

		// make the test blockchain (and state)
		blocks, receipts, chain = test_helpers.MakeChain(5, test_helpers.Genesis, test_helpers.TestChainGen)
		params := statediff.Params{
			IntermediateStateNodes:   true,
			IntermediateStorageNodes: true,
		}

		// iterate over the blocks, generating statediff payloads, and transforming the data into Postgres
		builder := statediff.NewBuilder(chain.StateCache())
		for i, block := range blocks {
			blockHashes = append(blockHashes, block.Hash())
			var args statediff.Args
			var rcts types.Receipts
			if i == 0 {
				args = statediff.Args{
					OldStateRoot: common.Hash{},
					NewStateRoot: block.Root(),
					BlockNumber:  block.Number(),
					BlockHash:    block.Hash(),
				}
			} else {
				args = statediff.Args{
					OldStateRoot: blocks[i-1].Root(),
					NewStateRoot: block.Root(),
					BlockNumber:  block.Number(),
					BlockHash:    block.Hash(),
				}
				rcts = receipts[i-1]
			}

			var diff sdtypes.StateObject
			diff, err = builder.BuildStateDiffObject(args, params)
			Expect(err).ToNot(HaveOccurred())

			tx, err := transformer.PushBlock(block, rcts, mockTD)
			Expect(err).ToNot(HaveOccurred())

			for _, node := range diff.Nodes {
				err = transformer.PushStateNode(tx, node, block.Hash().String())
				Expect(err).ToNot(HaveOccurred())
			}

			err = tx.Submit(err)
			Expect(err).ToNot(HaveOccurred())
		}

		// Insert some non-canonical data into the database so that we test our ability to discern canonicity
		indexAndPublisher := shared.SetupTestStateDiffIndexer(ctx, chainConfig, test_helpers.Genesis.Hash())

		blockHash = test_helpers.MockBlock.Hash()
		contractAddress = test_helpers.ContractAddr

		tx, err := indexAndPublisher.PushBlock(test_helpers.MockBlock, test_helpers.MockReceipts, test_helpers.MockBlock.Difficulty())
		Expect(err).ToNot(HaveOccurred())

		err = tx.Submit(err)
		Expect(err).ToNot(HaveOccurred())

		// The non-canonical header has a child
		tx, err = indexAndPublisher.PushBlock(test_helpers.MockChild, test_helpers.MockReceipts, test_helpers.MockChild.Difficulty())
		Expect(err).ToNot(HaveOccurred())

		ccHash := sdtypes.CodeAndCodeHash{
			Hash: test_helpers.CodeHash,
			Code: test_helpers.ContractCode,
		}

		err = indexAndPublisher.PushCodeAndCodeHash(tx, ccHash)
		Expect(err).ToNot(HaveOccurred())

		err = tx.Submit(err)
		Expect(err).ToNot(HaveOccurred())

		graphQLServer, err = graphql.New(backend, gqlEndPoint, nil, []string{"*"}, rpc.HTTPTimeouts{})
		Expect(err).ToNot(HaveOccurred())

		err = graphQLServer.Start(nil)
		Expect(err).ToNot(HaveOccurred())
	})

	defer It("test teardown", func() {
		err := graphQLServer.Stop()
		Expect(err).ToNot(HaveOccurred())
		shared.TearDownDB(db)
		chain.Stop()
	})

	Describe("eth_getLogs", func() {
		It("Retrieves logs that matches the provided blockHash and contract address", func() {
			logs, err := client.GetLogs(ctx, blockHash, &contractAddress)
			Expect(err).ToNot(HaveOccurred())

			expectedLogs := []graphql.LogResponse{
				{
					Topics:      test_helpers.MockLog1.Topics,
					Data:        hexutil.Bytes(test_helpers.MockLog1.Data),
					Transaction: graphql.TransactionResp{Hash: test_helpers.MockTransactions[0].Hash()},
					ReceiptCID:  test_helpers.Rct1CID.String(),
					Status:      int32(test_helpers.MockReceipts[0].Status),
				},
			}

			Expect(logs).To(Equal(expectedLogs))
		})

		It("Retrieves logs for the failed receipt status that matches the provided blockHash and another contract address", func() {
			logs, err := client.GetLogs(ctx, blockHash, &test_helpers.AnotherAddress2)
			Expect(err).ToNot(HaveOccurred())

			expectedLogs := []graphql.LogResponse{
				{
					Topics:      test_helpers.MockLog6.Topics,
					Data:        hexutil.Bytes(test_helpers.MockLog6.Data),
					Transaction: graphql.TransactionResp{Hash: test_helpers.MockTransactions[3].Hash()},
					ReceiptCID:  test_helpers.Rct4CID.String(),
					Status:      int32(test_helpers.MockReceipts[3].Status),
				},
			}

			Expect(logs).To(Equal(expectedLogs))
		})

		It("Retrieves all the logs for the receipt that matches the provided blockHash and nil contract address", func() {
			logs, err := client.GetLogs(ctx, blockHash, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(logs)).To(Equal(6))
		})

		It("Retrieves logs with random hash", func() {
			logs, err := client.GetLogs(ctx, randomHash, &contractAddress)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(logs)).To(Equal(0))
		})
	})

	Describe("eth_getStorageAt", func() {
		It("Retrieves the storage value at the provided contract address and storage leaf key at the block with the provided hash", func() {
			storageRes, err := client.GetStorageAt(ctx, blockHashes[2], contractAddress, test_helpers.IndexOne)
			Expect(err).ToNot(HaveOccurred())
			Expect(storageRes.Value).To(Equal(common.HexToHash("01")))

			storageRes, err = client.GetStorageAt(ctx, blockHashes[3], contractAddress, test_helpers.IndexOne)
			Expect(err).ToNot(HaveOccurred())
			Expect(storageRes.Value).To(Equal(common.HexToHash("03")))

			storageRes, err = client.GetStorageAt(ctx, blockHashes[4], contractAddress, test_helpers.IndexOne)
			Expect(err).ToNot(HaveOccurred())
			Expect(storageRes.Value).To(Equal(common.HexToHash("09")))
		})

		It("Retrieves empty data if it tries to access a contract at a blockHash which does not exist", func() {
			storageRes, err := client.GetStorageAt(ctx, blockHashes[0], contractAddress, test_helpers.IndexOne)
			Expect(err).ToNot(HaveOccurred())
			Expect(storageRes.Value).To(Equal(common.Hash{}))

			storageRes, err = client.GetStorageAt(ctx, blockHashes[1], contractAddress, test_helpers.IndexOne)
			Expect(err).ToNot(HaveOccurred())
			Expect(storageRes.Value).To(Equal(common.Hash{}))
		})

		It("Retrieves empty data if it tries to access a contract slot which does not exist", func() {
			storageRes, err := client.GetStorageAt(ctx, blockHashes[3], contractAddress, randomHash.Hex())
			Expect(err).ToNot(HaveOccurred())
			Expect(storageRes.Value).To(Equal(common.Hash{}))
		})
	})

	Describe("allEthHeaderCids", func() {
		It("Retrieves header_cids that matches the provided blockNumber", func() {
			allEthHeaderCidsResp, err := client.AllEthHeaderCids(ctx, graphql.EthHeaderCidCondition{BlockNumber: new(graphql.BigInt).SetUint64(2)})
			Expect(err).ToNot(HaveOccurred())

			headerCIDs, txCIDs, err := backend.Retriever.RetrieveHeaderAndTxCIDsByBlockNumber(2)
			Expect(err).ToNot(HaveOccurred())

			// Begin tx
			tx, err := backend.DB.Beginx()
			Expect(err).ToNot(HaveOccurred())
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

			headerIPLDs, err := backend.Fetcher.FetchHeaders(tx, headerCIDs)
			Expect(err).ToNot(HaveOccurred())

			for idx, headerCID := range headerCIDs {
				ethHeaderCid := allEthHeaderCidsResp.Nodes[idx]

				compareEthHeaderCid(ethHeaderCid, headerCID, txCIDs[idx], headerIPLDs[idx])
			}
		})

		It("Retrieves header_cids that matches the provided blockHash", func() {
			blockHash := blocks[1].Hash().String()
			allEthHeaderCidsResp, err := client.AllEthHeaderCids(ctx, graphql.EthHeaderCidCondition{BlockHash: &blockHash})
			Expect(err).ToNot(HaveOccurred())

			headerCID, txCIDs, err := backend.Retriever.RetrieveHeaderAndTxCIDsByBlockHash(blocks[1].Hash())
			Expect(err).ToNot(HaveOccurred())

			// Begin tx
			tx, err := backend.DB.Beginx()
			Expect(err).ToNot(HaveOccurred())
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

			headerIPLDs, err := backend.Fetcher.FetchHeaders(tx, []models.HeaderModel{headerCID})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(allEthHeaderCidsResp.Nodes)).To(Equal(1))

			Expect(len(allEthHeaderCidsResp.Nodes)).To(Equal(1))
			ethHeaderCid := allEthHeaderCidsResp.Nodes[0]
			compareEthHeaderCid(ethHeaderCid, headerCID, txCIDs, headerIPLDs[0])
		})
	})

	Describe("ethTransactionCidByTxHash", func() {
		It("Retrieves tx_cid that matches the provided txHash", func() {
			txHash := blocks[2].Transactions()[0].Hash().String()
			ethTransactionCidResp, err := client.EthTransactionCidByTxHash(ctx, txHash)
			Expect(err).ToNot(HaveOccurred())

			txCID, err := backend.Retriever.RetrieveTxCIDByHash(txHash)
			Expect(err).ToNot(HaveOccurred())

			compareEthTxCid(*ethTransactionCidResp, txCID)

			// Begin tx
			tx, err := backend.DB.Beginx()
			Expect(err).ToNot(HaveOccurred())
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

			txIPLDs, err := backend.Fetcher.FetchTrxs(tx, []models.TxModel{txCID})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(txIPLDs)).To(Equal(1))
			Expect(ethTransactionCidResp.BlockByMhKey.Data).To(Equal(graphql.Bytes(txIPLDs[0].Data).String()))
		})
	})
})

func compareEthHeaderCid(ethHeaderCid graphql.EthHeaderCidResp, headerCID models.HeaderModel, txCIDs []models.TxModel, headerIPLD models.IPLDModel) {
	blockNumber, err := strconv.ParseInt(headerCID.BlockNumber, 10, 64)
	Expect(err).ToNot(HaveOccurred())

	td, err := strconv.ParseInt(headerCID.TotalDifficulty, 10, 64)
	Expect(err).ToNot(HaveOccurred())

	Expect(ethHeaderCid.Cid).To(Equal(headerCID.CID))
	Expect(ethHeaderCid.BlockNumber).To(Equal(*new(graphql.BigInt).SetUint64(uint64(blockNumber))))
	Expect(ethHeaderCid.BlockHash).To(Equal(headerCID.BlockHash))
	Expect(ethHeaderCid.ParentHash).To(Equal(headerCID.ParentHash))
	Expect(ethHeaderCid.Timestamp).To(Equal(*new(graphql.BigInt).SetUint64(headerCID.Timestamp)))
	Expect(ethHeaderCid.StateRoot).To(Equal(headerCID.StateRoot))
	Expect(ethHeaderCid.Td).To(Equal(*new(graphql.BigInt).SetUint64(uint64(td))))
	Expect(ethHeaderCid.TxRoot).To(Equal(headerCID.TxRoot))
	Expect(ethHeaderCid.ReceiptRoot).To(Equal(headerCID.RctRoot))
	Expect(ethHeaderCid.UncleRoot).To(Equal(headerCID.UncleRoot))
	Expect(ethHeaderCid.Bloom).To(Equal(graphql.Bytes(headerCID.Bloom).String()))

	for tIdx, txCID := range txCIDs {
		ethTxCid := ethHeaderCid.EthTransactionCidsByHeaderId.Nodes[tIdx]
		compareEthTxCid(ethTxCid, txCID)
	}

	Expect(ethHeaderCid.BlockByMhKey.Data).To(Equal(graphql.Bytes(headerIPLD.Data).String()))
	Expect(ethHeaderCid.BlockByMhKey.Key).To(Equal(headerIPLD.Key))
}

func compareEthTxCid(ethTxCid graphql.EthTransactionCidResp, txCID models.TxModel) {
	Expect(ethTxCid.Cid).To(Equal(txCID.CID))
	Expect(ethTxCid.TxHash).To(Equal(txCID.TxHash))
	Expect(ethTxCid.Index).To(Equal(int32(txCID.Index)))
	Expect(ethTxCid.Src).To(Equal(txCID.Src))
	Expect(ethTxCid.Dst).To(Equal(txCID.Dst))
}
