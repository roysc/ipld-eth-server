package integration_test

import (
	"context"
	"math/big"
	"os"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cerc-io/ipld-eth-server/v4/pkg/eth"
	integration "github.com/cerc-io/ipld-eth-server/v4/test"
)

var _ = Describe("Integration test", func() {
	directProxyEthCalls, err := strconv.ParseBool(os.Getenv("ETH_FORWARD_ETH_CALLS"))
	Expect(err).To(BeNil())
	gethHttpPath := "http://127.0.0.1:8545"
	gethClient, err := ethclient.Dial(gethHttpPath)
	Expect(err).ToNot(HaveOccurred())

	ipldEthHttpPath := "http://127.0.0.1:8081"
	ipldClient, err := ethclient.Dial(ipldEthHttpPath)
	Expect(err).ToNot(HaveOccurred())

	ctx := context.Background()

	var contract *integration.ContractDeployed
	var erc20TotalSupply *big.Int
	var tx *integration.Tx
	var bigIntResult bool
	var contractErr error
	var txErr error
	sleepInterval := 2 * time.Second

	BeforeEach(func() {
		if !directProxyEthCalls {
			Skip("skipping direct-proxy-forwarding integration tests")
		}
	})

	Describe("get Block", func() {
		BeforeEach(func() {
			contract, contractErr = integration.DeployContract()
			time.Sleep(sleepInterval)
		})

		It("get not existing block by number", func() {
			Expect(contractErr).ToNot(HaveOccurred())

			blockNum := contract.BlockNumber + 100

			gethBlock, err := gethClient.BlockByNumber(ctx, big.NewInt(int64(blockNum)))
			Expect(err).To(MatchError(ethereum.NotFound))
			Expect(gethBlock).To(BeZero())

			ipldBlock, err := ipldClient.BlockByNumber(ctx, big.NewInt(int64(blockNum)))
			Expect(err).To(MatchError(ethereum.NotFound))
			Expect(ipldBlock).To(BeZero())
		})

		It("get not existing block by hash", func() {
			gethBlock, err := gethClient.BlockByHash(ctx, common.HexToHash(nonExistingBlockHash))
			Expect(err).To(MatchError(ethereum.NotFound))
			Expect(gethBlock).To(BeZero())

			ipldBlock, err := ipldClient.BlockByHash(ctx, common.HexToHash(nonExistingBlockHash))
			Expect(err).To(MatchError(ethereum.NotFound))
			Expect(ipldBlock).To(BeZero())
		})

		It("get block by number", func() {
			Expect(contractErr).ToNot(HaveOccurred())

			blockNum := contract.BlockNumber

			_, err := gethClient.BlockByNumber(ctx, big.NewInt(int64(blockNum)))
			Expect(err).ToNot(HaveOccurred())

			_, err = ipldClient.BlockByNumber(ctx, big.NewInt(int64(blockNum)))
			Expect(err).To(HaveOccurred())
		})

		It("get block by hash", func() {
			_, err := gethClient.BlockByHash(ctx, common.HexToHash(contract.BlockHash))
			Expect(err).ToNot(HaveOccurred())

			_, err = ipldClient.BlockByHash(ctx, common.HexToHash(contract.BlockHash))
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Transaction", func() {
		BeforeEach(func() {
			contract, contractErr = integration.DeployContract()
			time.Sleep(sleepInterval)
		})

		It("Get tx by hash", func() {
			Expect(contractErr).ToNot(HaveOccurred())

			_, _, err := gethClient.TransactionByHash(ctx, common.HexToHash(contract.TransactionHash))
			Expect(err).ToNot(HaveOccurred())

			_, _, err = ipldClient.TransactionByHash(ctx, common.HexToHash(contract.TransactionHash))
			Expect(err).To(HaveOccurred())
		})

		It("Get tx by block hash and index", func() {
			_, err := gethClient.TransactionInBlock(ctx, common.HexToHash(contract.BlockHash), 0)
			Expect(err).ToNot(HaveOccurred())

			_, err = ipldClient.TransactionInBlock(ctx, common.HexToHash(contract.BlockHash), 0)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Receipt", func() {
		BeforeEach(func() {
			contract, contractErr = integration.DeployContract()
			time.Sleep(sleepInterval)
		})

		It("Get tx receipt", func() {
			Expect(contractErr).ToNot(HaveOccurred())

			_, err := gethClient.TransactionReceipt(ctx, common.HexToHash(contract.TransactionHash))
			Expect(err).ToNot(HaveOccurred())

			_, err = ipldClient.TransactionReceipt(ctx, common.HexToHash(contract.TransactionHash))
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("FilterLogs", func() {
		BeforeEach(func() {
			contract, contractErr = integration.DeployContract()
			time.Sleep(sleepInterval)
		})

		It("with blockhash", func() {
			Expect(contractErr).ToNot(HaveOccurred())

			blockHash := common.HexToHash(contract.BlockHash)
			filterQuery := ethereum.FilterQuery{
				//Addresses: addresses,
				BlockHash: &blockHash,
				Topics:    [][]common.Hash{},
			}

			gethLogs, err := gethClient.FilterLogs(ctx, filterQuery)
			Expect(err).ToNot(HaveOccurred())

			ipldLogs, err := ipldClient.FilterLogs(ctx, filterQuery)
			Expect(err).ToNot(HaveOccurred())

			// not empty list
			Expect(gethLogs).ToNot(BeEmpty())
			// empty list
			Expect(ipldLogs).To(BeEmpty())
		})
	})

	Describe("CodeAt", func() {
		BeforeEach(func() {
			contract, contractErr = integration.DeployContract()
			time.Sleep(sleepInterval)
		})

		It("gets code at non-existing address without block number", func() {
			Expect(contractErr).ToNot(HaveOccurred())

			gethCode, err := gethClient.CodeAt(ctx, common.HexToAddress(nonExistingAddress), nil)
			Expect(err).ToNot(HaveOccurred())

			ipldCode, err := ipldClient.CodeAt(ctx, common.HexToAddress(nonExistingAddress), nil)
			Expect(err).ToNot(HaveOccurred())

			Expect(gethCode).To(BeEmpty())
			Expect(gethCode).To(Equal(ipldCode))
		})
		It("gets code of deployed contract without block number", func() {
			_, err := gethClient.CodeAt(ctx, common.HexToAddress(contract.Address), nil)
			Expect(err).ToNot(HaveOccurred())

			ipldCode, err := ipldClient.CodeAt(ctx, common.HexToAddress(contract.Address), nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(ipldCode).To(BeEmpty())
		})
		It("gets code of deployed contract with block number", func() {
			_, err := gethClient.CodeAt(ctx, common.HexToAddress(contract.Address), big.NewInt(int64(contract.BlockNumber)))
			Expect(err).ToNot(HaveOccurred())

			ipldCode, err := ipldClient.CodeAt(ctx, common.HexToAddress(contract.Address), big.NewInt(int64(contract.BlockNumber)))
			Expect(err).ToNot(HaveOccurred())
			Expect(ipldCode).To(BeEmpty())
		})
		It("gets code of contract that doesn't exist at this height", func() {
			gethCode, err := gethClient.CodeAt(ctx, common.HexToAddress(contract.Address), big.NewInt(int64(contract.BlockNumber-1)))
			Expect(err).ToNot(HaveOccurred())

			ipldCode, err := ipldClient.CodeAt(ctx, common.HexToAddress(contract.Address), big.NewInt(int64(contract.BlockNumber-1)))
			Expect(err).ToNot(HaveOccurred())

			Expect(gethCode).To(BeEmpty())
			Expect(gethCode).To(Equal(ipldCode))
		})
	})

	Describe("Get balance", func() {
		address := "0x1111111111111111111111111111111111111112"
		BeforeEach(func() {
			tx, txErr = integration.SendEth(address, "0.01")
			time.Sleep(sleepInterval)
		})

		It("gets balance for an account with eth without block number", func() {
			Expect(txErr).ToNot(HaveOccurred())

			gethBalance, err := gethClient.BalanceAt(ctx, common.HexToAddress(address), nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(gethBalance.String()).To(Equal(big.NewInt(10000000000000000).String()))

			ipldBalance, err := ipldClient.BalanceAt(ctx, common.HexToAddress(address), nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(ipldBalance.String()).To(Equal(big.NewInt(0).String()))
		})
		It("gets balance for an account with eth with block number", func() {
			Expect(txErr).ToNot(HaveOccurred())

			_, err := gethClient.BalanceAt(ctx, common.HexToAddress(address), big.NewInt(int64(tx.BlockNumber)))
			Expect(err).ToNot(HaveOccurred())

			_, err = ipldClient.BalanceAt(ctx, common.HexToAddress(address), big.NewInt(int64(tx.BlockNumber)))
			Expect(err).To(HaveOccurred())
		})
		It("gets historical balance for an account with eth with block number", func() {
			Expect(txErr).ToNot(HaveOccurred())

			_, err := gethClient.BalanceAt(ctx, common.HexToAddress(address), big.NewInt(int64(tx.BlockNumber-1)))
			Expect(err).ToNot(HaveOccurred())

			_, err = ipldClient.BalanceAt(ctx, common.HexToAddress(address), big.NewInt(int64(tx.BlockNumber-1)))
			Expect(err).To(HaveOccurred())
		})
		It("gets balance for a non-existing account without block number", func() {
			Expect(txErr).ToNot(HaveOccurred())

			gethBalance, err := gethClient.BalanceAt(ctx, common.HexToAddress(nonExistingAddress), nil)
			Expect(err).ToNot(HaveOccurred())

			ipldBalance, err := ipldClient.BalanceAt(ctx, common.HexToAddress(nonExistingAddress), nil)
			Expect(err).ToNot(HaveOccurred())

			Expect(gethBalance).To(Equal(ipldBalance))
		})
		It("gets balance for an non-existing block number", func() {
			Expect(txErr).ToNot(HaveOccurred())

			gethBalance, err := gethClient.BalanceAt(ctx, common.HexToAddress(address), big.NewInt(int64(tx.BlockNumber+3)))
			Expect(err).To(MatchError("header not found"))

			ipldBalance, err := ipldClient.BalanceAt(ctx, common.HexToAddress(nonExistingAddress), big.NewInt(int64(tx.BlockNumber+3)))
			Expect(err).To(MatchError("header not found"))

			Expect(gethBalance).To(Equal(ipldBalance))
		})
	})

	Describe("Get Storage", func() {
		BeforeEach(func() {
			contract, contractErr = integration.DeployContract()
			erc20TotalSupply, bigIntResult = new(big.Int).SetString("1000000000000000000000", 10)

			time.Sleep(sleepInterval)
		})

		It("gets ERC20 total supply (without block number)", func() {
			Expect(contractErr).ToNot(HaveOccurred())
			Expect(bigIntResult).To(Equal(true))

			totalSupplyIndex := "0x2"

			gethStorage, err := gethClient.StorageAt(ctx, common.HexToAddress(contract.Address), common.HexToHash(totalSupplyIndex), nil)
			Expect(err).ToNot(HaveOccurred())

			gethTotalSupply := new(big.Int).SetBytes(gethStorage)
			Expect(gethTotalSupply).To(Equal(erc20TotalSupply))

			ipldStorage, err := ipldClient.StorageAt(ctx, common.HexToAddress(contract.Address), common.HexToHash(totalSupplyIndex), nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(ipldStorage).To(Equal(make([]byte, 32)))
		})

		It("gets ERC20 total supply (with block number)", func() {
			totalSupplyIndex := "0x2"

			gethStorage, err := gethClient.StorageAt(ctx, common.HexToAddress(contract.Address), common.HexToHash(totalSupplyIndex), big.NewInt(int64(contract.BlockNumber)))
			Expect(err).ToNot(HaveOccurred())

			gethTotalSupply := new(big.Int).SetBytes(gethStorage)
			Expect(gethTotalSupply).To(Equal(erc20TotalSupply))

			_, err = ipldClient.StorageAt(ctx, common.HexToAddress(contract.Address), common.HexToHash(totalSupplyIndex), big.NewInt(int64(contract.BlockNumber)))
			Expect(err).To(HaveOccurred())
		})
		It("gets storage for non-existing account", func() {
			totalSupplyIndex := "0x2"

			_, err := gethClient.StorageAt(ctx, common.HexToAddress(nonExistingAddress), common.HexToHash(totalSupplyIndex), big.NewInt(int64(contract.BlockNumber)))
			Expect(err).ToNot(HaveOccurred())

			_, err = ipldClient.StorageAt(ctx, common.HexToAddress(nonExistingAddress), common.HexToHash(totalSupplyIndex), big.NewInt(int64(contract.BlockNumber)))
			Expect(err).To(MatchError("header not found"))
		})
		It("gets storage for non-existing contract slot", func() {
			_, err := gethClient.StorageAt(ctx, common.HexToAddress(contract.Address), randomHash, big.NewInt(int64(contract.BlockNumber)))
			Expect(err).ToNot(HaveOccurred())

			_, err = ipldClient.StorageAt(ctx, common.HexToAddress(contract.Address), randomHash, big.NewInt(int64(contract.BlockNumber)))
			Expect(err).To(MatchError("header not found"))
		})
		It("gets storage for non-existing contract", func() {
			totalSupplyIndex := "0x2"
			gethStorage, err := gethClient.StorageAt(ctx, common.HexToAddress(contract.Address), common.HexToHash(totalSupplyIndex), big.NewInt(0))
			Expect(err).ToNot(HaveOccurred())

			ipldStorage, err := ipldClient.StorageAt(ctx, common.HexToAddress(contract.Address), common.HexToHash(totalSupplyIndex), big.NewInt(0))
			Expect(err).ToNot(HaveOccurred())
			Expect(gethStorage).To(Equal(ipldStorage))
		})
		It("gets storage for non-existing block number", func() {
			blockNum := contract.BlockNumber + 100
			totalSupplyIndex := "0x2"

			gethStorage, err := gethClient.StorageAt(ctx, common.HexToAddress(contract.Address), common.HexToHash(totalSupplyIndex), big.NewInt(int64(blockNum)))
			Expect(err).To(MatchError("header not found"))

			ipldStorage, err := ipldClient.StorageAt(ctx, common.HexToAddress(contract.Address), common.HexToHash(totalSupplyIndex), big.NewInt(int64(blockNum)))
			Expect(err).To(MatchError("header not found"))
			Expect(gethStorage).To(Equal(ipldStorage))
		})

		It("get storage after self destruct", func() {
			totalSupplyIndex := "0x2"

			tx, err := integration.DestroyContract(contract.Address)
			Expect(err).ToNot(HaveOccurred())

			time.Sleep(sleepInterval)

			gethStorage1, err := gethClient.StorageAt(ctx, common.HexToAddress(contract.Address), common.HexToHash(totalSupplyIndex), big.NewInt(tx.BlockNumber-1))
			Expect(err).ToNot(HaveOccurred())
			gethStorage2, err := gethClient.StorageAt(ctx, common.HexToAddress(contract.Address), common.HexToHash(totalSupplyIndex), big.NewInt(tx.BlockNumber))
			Expect(err).ToNot(HaveOccurred())

			Expect(gethStorage1).NotTo(Equal(gethStorage2))
			Expect(gethStorage2).To(Equal(eth.EmptyNodeValue))

			_, err = ipldClient.StorageAt(ctx, common.HexToAddress(contract.Address), common.HexToHash(totalSupplyIndex), big.NewInt(tx.BlockNumber-1))
			Expect(err).To(HaveOccurred())
			_, err = ipldClient.StorageAt(ctx, common.HexToAddress(contract.Address), common.HexToHash(totalSupplyIndex), big.NewInt(tx.BlockNumber))
			Expect(err).To(MatchError("header not found"))

			// Query the current block
			ipldStorage3, err := ipldClient.StorageAt(ctx, common.HexToAddress(contract.Address), common.HexToHash(totalSupplyIndex), nil)
			Expect(err).ToNot(HaveOccurred())

			Expect(eth.EmptyNodeValue).To(Equal(ipldStorage3))
		})
	})

	Describe("eth call", func() {
		BeforeEach(func() {
			contract, contractErr = integration.DeployContract()
			erc20TotalSupply, bigIntResult = new(big.Int).SetString("1000000000000000000000", 10)

			time.Sleep(sleepInterval)
		})

		It("calls totalSupply() without block number", func() {
			Expect(contractErr).ToNot(HaveOccurred())
			Expect(bigIntResult).To(Equal(true))

			contractAddress := common.HexToAddress(contract.Address)

			msg := ethereum.CallMsg{
				To:   &contractAddress,
				Data: common.Hex2Bytes("18160ddd"), // totalSupply()
			}
			gethResult, err := gethClient.CallContract(ctx, msg, nil)
			Expect(err).ToNot(HaveOccurred())

			gethTotalSupply := new(big.Int).SetBytes(gethResult)
			Expect(gethTotalSupply).To(Equal(erc20TotalSupply))

			ipldResult, err := ipldClient.CallContract(ctx, msg, nil)
			Expect(err).ToNot(HaveOccurred())

			Expect(gethResult).To(Equal(ipldResult))
		})

		It("calls totalSupply() with block number", func() {
			contractAddress := common.HexToAddress(contract.Address)

			msg := ethereum.CallMsg{
				To:   &contractAddress,
				Data: common.Hex2Bytes("18160ddd"), // totalSupply()
			}
			gethResult, err := gethClient.CallContract(ctx, msg, big.NewInt(int64(contract.BlockNumber)))
			Expect(err).ToNot(HaveOccurred())

			gethTotalSupply := new(big.Int).SetBytes(gethResult)
			Expect(gethTotalSupply).To(Equal(erc20TotalSupply))

			ipldResult, err := ipldClient.CallContract(ctx, msg, big.NewInt(int64(contract.BlockNumber)))
			Expect(err).ToNot(HaveOccurred())

			Expect(gethResult).To(Equal(ipldResult))
		})
	})

	Describe("Chain ID", func() {
		It("Check chain id", func() {
			_, err := gethClient.ChainID(ctx)
			Expect(err).ToNot(HaveOccurred())

			_, err = ipldClient.ChainID(ctx)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
