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

package mocks

import (
	"strconv"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/vulcanize/ipfs-blockchain-watcher/pkg/btc"
	"github.com/vulcanize/ipfs-blockchain-watcher/pkg/shared"
)

var (
	MockHeaderCID         = shared.TestCID([]byte("MockHeaderCID"))
	MockTrxCID1           = shared.TestCID([]byte("MockTrxCID1"))
	MockTrxCID2           = shared.TestCID([]byte("MockTrxCID2"))
	MockTrxCID3           = shared.TestCID([]byte("MockTrxCID3"))
	MockHeaderMhKey       = shared.MultihashKeyFromCID(MockHeaderCID)
	MockTrxMhKey1         = shared.MultihashKeyFromCID(MockTrxCID1)
	MockTrxMhKey2         = shared.MultihashKeyFromCID(MockTrxCID2)
	MockTrxMhKey3         = shared.MultihashKeyFromCID(MockTrxCID3)
	MockBlockHeight int64 = 1337
	MockBlock             = wire.MsgBlock{
		Header: wire.BlockHeader{
			Version: 1,
			PrevBlock: chainhash.Hash([32]byte{ // Make go vet happy.
				0x50, 0x12, 0x01, 0x19, 0x17, 0x2a, 0x61, 0x04,
				0x21, 0xa6, 0xc3, 0x01, 0x1d, 0xd3, 0x30, 0xd9,
				0xdf, 0x07, 0xb6, 0x36, 0x16, 0xc2, 0xcc, 0x1f,
				0x1c, 0xd0, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00,
			}), // 000000000002d01c1fccc21636b607dfd930d31d01c3a62104612a1719011250
			MerkleRoot: chainhash.Hash([32]byte{ // Make go vet happy.
				0x66, 0x57, 0xa9, 0x25, 0x2a, 0xac, 0xd5, 0xc0,
				0xb2, 0x94, 0x09, 0x96, 0xec, 0xff, 0x95, 0x22,
				0x28, 0xc3, 0x06, 0x7c, 0xc3, 0x8d, 0x48, 0x85,
				0xef, 0xb5, 0xa4, 0xac, 0x42, 0x47, 0xe9, 0xf3,
			}), // f3e94742aca4b5ef85488dc37c06c3282295ffec960994b2c0d5ac2a25a95766
			Timestamp: time.Unix(1293623863, 0), // 2010-12-29 11:57:43 +0000 UTC
			Bits:      0x1b04864c,               // 453281356
			Nonce:     0x10572b0f,               // 274148111
		},
		Transactions: []*wire.MsgTx{
			{
				Version: 1,
				TxIn: []*wire.TxIn{
					{
						PreviousOutPoint: wire.OutPoint{
							Hash:  chainhash.Hash{},
							Index: 0xffffffff,
						},
						SignatureScript: []byte{
							0x04, 0x4c, 0x86, 0x04, 0x1b, 0x02, 0x06, 0x02,
						},
						Sequence: 0xffffffff,
					},
				},
				TxOut: []*wire.TxOut{
					{
						Value: 0x12a05f200, // 5000000000
						PkScript: []byte{
							0x41, // OP_DATA_65
							0x04, 0x1b, 0x0e, 0x8c, 0x25, 0x67, 0xc1, 0x25,
							0x36, 0xaa, 0x13, 0x35, 0x7b, 0x79, 0xa0, 0x73,
							0xdc, 0x44, 0x44, 0xac, 0xb8, 0x3c, 0x4e, 0xc7,
							0xa0, 0xe2, 0xf9, 0x9d, 0xd7, 0x45, 0x75, 0x16,
							0xc5, 0x81, 0x72, 0x42, 0xda, 0x79, 0x69, 0x24,
							0xca, 0x4e, 0x99, 0x94, 0x7d, 0x08, 0x7f, 0xed,
							0xf9, 0xce, 0x46, 0x7c, 0xb9, 0xf7, 0xc6, 0x28,
							0x70, 0x78, 0xf8, 0x01, 0xdf, 0x27, 0x6f, 0xdf,
							0x84, // 65-byte signature
							0xac, // OP_CHECKSIG
						},
					},
				},
				LockTime: 0,
			},
			{
				Version: 1,
				TxIn: []*wire.TxIn{
					{
						PreviousOutPoint: wire.OutPoint{
							Hash: chainhash.Hash([32]byte{ // Make go vet happy.
								0x03, 0x2e, 0x38, 0xe9, 0xc0, 0xa8, 0x4c, 0x60,
								0x46, 0xd6, 0x87, 0xd1, 0x05, 0x56, 0xdc, 0xac,
								0xc4, 0x1d, 0x27, 0x5e, 0xc5, 0x5f, 0xc0, 0x07,
								0x79, 0xac, 0x88, 0xfd, 0xf3, 0x57, 0xa1, 0x87,
							}), // 87a157f3fd88ac7907c05fc55e271dc4acdc5605d187d646604ca8c0e9382e03
							Index: 0,
						},
						SignatureScript: []byte{
							0x49, // OP_DATA_73
							0x30, 0x46, 0x02, 0x21, 0x00, 0xc3, 0x52, 0xd3,
							0xdd, 0x99, 0x3a, 0x98, 0x1b, 0xeb, 0xa4, 0xa6,
							0x3a, 0xd1, 0x5c, 0x20, 0x92, 0x75, 0xca, 0x94,
							0x70, 0xab, 0xfc, 0xd5, 0x7d, 0xa9, 0x3b, 0x58,
							0xe4, 0xeb, 0x5d, 0xce, 0x82, 0x02, 0x21, 0x00,
							0x84, 0x07, 0x92, 0xbc, 0x1f, 0x45, 0x60, 0x62,
							0x81, 0x9f, 0x15, 0xd3, 0x3e, 0xe7, 0x05, 0x5c,
							0xf7, 0xb5, 0xee, 0x1a, 0xf1, 0xeb, 0xcc, 0x60,
							0x28, 0xd9, 0xcd, 0xb1, 0xc3, 0xaf, 0x77, 0x48,
							0x01, // 73-byte signature
							0x41, // OP_DATA_65
							0x04, 0xf4, 0x6d, 0xb5, 0xe9, 0xd6, 0x1a, 0x9d,
							0xc2, 0x7b, 0x8d, 0x64, 0xad, 0x23, 0xe7, 0x38,
							0x3a, 0x4e, 0x6c, 0xa1, 0x64, 0x59, 0x3c, 0x25,
							0x27, 0xc0, 0x38, 0xc0, 0x85, 0x7e, 0xb6, 0x7e,
							0xe8, 0xe8, 0x25, 0xdc, 0xa6, 0x50, 0x46, 0xb8,
							0x2c, 0x93, 0x31, 0x58, 0x6c, 0x82, 0xe0, 0xfd,
							0x1f, 0x63, 0x3f, 0x25, 0xf8, 0x7c, 0x16, 0x1b,
							0xc6, 0xf8, 0xa6, 0x30, 0x12, 0x1d, 0xf2, 0xb3,
							0xd3, // 65-byte pubkey
						},
						Sequence: 0xffffffff,
					},
				},
				TxOut: []*wire.TxOut{
					{
						Value: 0x2123e300, // 556000000
						PkScript: []byte{
							0x76, // OP_DUP
							0xa9, // OP_HASH160
							0x14, // OP_DATA_20
							0xc3, 0x98, 0xef, 0xa9, 0xc3, 0x92, 0xba, 0x60,
							0x13, 0xc5, 0xe0, 0x4e, 0xe7, 0x29, 0x75, 0x5e,
							0xf7, 0xf5, 0x8b, 0x32,
							0x88, // OP_EQUALVERIFY
							0xac, // OP_CHECKSIG
						},
					},
					{
						Value: 0x108e20f00, // 4444000000
						PkScript: []byte{
							0x76, // OP_DUP
							0xa9, // OP_HASH160
							0x14, // OP_DATA_20
							0x94, 0x8c, 0x76, 0x5a, 0x69, 0x14, 0xd4, 0x3f,
							0x2a, 0x7a, 0xc1, 0x77, 0xda, 0x2c, 0x2f, 0x6b,
							0x52, 0xde, 0x3d, 0x7c,
							0x88, // OP_EQUALVERIFY
							0xac, // OP_CHECKSIG
						},
					},
				},
				LockTime: 0,
			},
			{
				Version: 1,
				TxIn: []*wire.TxIn{
					{
						PreviousOutPoint: wire.OutPoint{
							Hash: chainhash.Hash([32]byte{ // Make go vet happy.
								0xc3, 0x3e, 0xbf, 0xf2, 0xa7, 0x09, 0xf1, 0x3d,
								0x9f, 0x9a, 0x75, 0x69, 0xab, 0x16, 0xa3, 0x27,
								0x86, 0xaf, 0x7d, 0x7e, 0x2d, 0xe0, 0x92, 0x65,
								0xe4, 0x1c, 0x61, 0xd0, 0x78, 0x29, 0x4e, 0xcf,
							}), // cf4e2978d0611ce46592e02d7e7daf8627a316ab69759a9f3df109a7f2bf3ec3
							Index: 1,
						},
						SignatureScript: []byte{
							0x47, // OP_DATA_71
							0x30, 0x44, 0x02, 0x20, 0x03, 0x2d, 0x30, 0xdf,
							0x5e, 0xe6, 0xf5, 0x7f, 0xa4, 0x6c, 0xdd, 0xb5,
							0xeb, 0x8d, 0x0d, 0x9f, 0xe8, 0xde, 0x6b, 0x34,
							0x2d, 0x27, 0x94, 0x2a, 0xe9, 0x0a, 0x32, 0x31,
							0xe0, 0xba, 0x33, 0x3e, 0x02, 0x20, 0x3d, 0xee,
							0xe8, 0x06, 0x0f, 0xdc, 0x70, 0x23, 0x0a, 0x7f,
							0x5b, 0x4a, 0xd7, 0xd7, 0xbc, 0x3e, 0x62, 0x8c,
							0xbe, 0x21, 0x9a, 0x88, 0x6b, 0x84, 0x26, 0x9e,
							0xae, 0xb8, 0x1e, 0x26, 0xb4, 0xfe, 0x01,
							0x41, // OP_DATA_65
							0x04, 0xae, 0x31, 0xc3, 0x1b, 0xf9, 0x12, 0x78,
							0xd9, 0x9b, 0x83, 0x77, 0xa3, 0x5b, 0xbc, 0xe5,
							0xb2, 0x7d, 0x9f, 0xff, 0x15, 0x45, 0x68, 0x39,
							0xe9, 0x19, 0x45, 0x3f, 0xc7, 0xb3, 0xf7, 0x21,
							0xf0, 0xba, 0x40, 0x3f, 0xf9, 0x6c, 0x9d, 0xee,
							0xb6, 0x80, 0xe5, 0xfd, 0x34, 0x1c, 0x0f, 0xc3,
							0xa7, 0xb9, 0x0d, 0xa4, 0x63, 0x1e, 0xe3, 0x95,
							0x60, 0x63, 0x9d, 0xb4, 0x62, 0xe9, 0xcb, 0x85,
							0x0f, // 65-byte pubkey
						},
						Sequence: 0xffffffff,
					},
				},
				TxOut: []*wire.TxOut{
					{
						Value: 0xf4240, // 1000000
						PkScript: []byte{
							0x76, // OP_DUP
							0xa9, // OP_HASH160
							0x14, // OP_DATA_20
							0xb0, 0xdc, 0xbf, 0x97, 0xea, 0xbf, 0x44, 0x04,
							0xe3, 0x1d, 0x95, 0x24, 0x77, 0xce, 0x82, 0x2d,
							0xad, 0xbe, 0x7e, 0x10,
							0x88, // OP_EQUALVERIFY
							0xac, // OP_CHECKSIG
						},
					},
					{
						Value: 0x11d260c0, // 299000000
						PkScript: []byte{
							0x76, // OP_DUP
							0xa9, // OP_HASH160
							0x14, // OP_DATA_20
							0x6b, 0x12, 0x81, 0xee, 0xc2, 0x5a, 0xb4, 0xe1,
							0xe0, 0x79, 0x3f, 0xf4, 0xe0, 0x8a, 0xb1, 0xab,
							0xb3, 0x40, 0x9c, 0xd9,
							0x88, // OP_EQUALVERIFY
							0xac, // OP_CHECKSIG
						},
					},
				},
				LockTime: 0,
			},
		},
	}
	MockTransactions = []*btcutil.Tx{
		btcutil.NewTx(MockBlock.Transactions[0]),
		btcutil.NewTx(MockBlock.Transactions[1]),
		btcutil.NewTx(MockBlock.Transactions[2]),
	}
	MockBlockPayload = btc.BlockPayload{
		Header:      &MockBlock.Header,
		Txs:         MockTransactions,
		BlockHeight: MockBlockHeight,
	}
	sClass1, addresses1, numOfSigs1, _ = txscript.ExtractPkScriptAddrs([]byte{
		0x41, // OP_DATA_65
		0x04, 0x1b, 0x0e, 0x8c, 0x25, 0x67, 0xc1, 0x25,
		0x36, 0xaa, 0x13, 0x35, 0x7b, 0x79, 0xa0, 0x73,
		0xdc, 0x44, 0x44, 0xac, 0xb8, 0x3c, 0x4e, 0xc7,
		0xa0, 0xe2, 0xf9, 0x9d, 0xd7, 0x45, 0x75, 0x16,
		0xc5, 0x81, 0x72, 0x42, 0xda, 0x79, 0x69, 0x24,
		0xca, 0x4e, 0x99, 0x94, 0x7d, 0x08, 0x7f, 0xed,
		0xf9, 0xce, 0x46, 0x7c, 0xb9, 0xf7, 0xc6, 0x28,
		0x70, 0x78, 0xf8, 0x01, 0xdf, 0x27, 0x6f, 0xdf,
		0x84, // 65-byte signature
		0xac, // OP_CHECKSIG
	}, &chaincfg.MainNetParams)
	sClass2a, addresses2a, numOfSigs2a, _ = txscript.ExtractPkScriptAddrs([]byte{
		0x76, // OP_DUP
		0xa9, // OP_HASH160
		0x14, // OP_DATA_20
		0xc3, 0x98, 0xef, 0xa9, 0xc3, 0x92, 0xba, 0x60,
		0x13, 0xc5, 0xe0, 0x4e, 0xe7, 0x29, 0x75, 0x5e,
		0xf7, 0xf5, 0x8b, 0x32,
		0x88, // OP_EQUALVERIFY
		0xac, // OP_CHECKSIG
	}, &chaincfg.MainNetParams)
	sClass2b, addresses2b, numOfSigs2b, _ = txscript.ExtractPkScriptAddrs([]byte{
		0x76, // OP_DUP
		0xa9, // OP_HASH160
		0x14, // OP_DATA_20
		0x94, 0x8c, 0x76, 0x5a, 0x69, 0x14, 0xd4, 0x3f,
		0x2a, 0x7a, 0xc1, 0x77, 0xda, 0x2c, 0x2f, 0x6b,
		0x52, 0xde, 0x3d, 0x7c,
		0x88, // OP_EQUALVERIFY
		0xac, // OP_CHECKSIG
	}, &chaincfg.MainNetParams)
	sClass3a, addresses3a, numOfSigs3a, _ = txscript.ExtractPkScriptAddrs([]byte{
		0x76, // OP_DUP
		0xa9, // OP_HASH160
		0x14, // OP_DATA_20
		0xb0, 0xdc, 0xbf, 0x97, 0xea, 0xbf, 0x44, 0x04,
		0xe3, 0x1d, 0x95, 0x24, 0x77, 0xce, 0x82, 0x2d,
		0xad, 0xbe, 0x7e, 0x10,
		0x88, // OP_EQUALVERIFY
		0xac, // OP_CHECKSIG
	}, &chaincfg.MainNetParams)
	sClass3b, addresses3b, numOfSigs3b, _ = txscript.ExtractPkScriptAddrs([]byte{
		0x76, // OP_DUP
		0xa9, // OP_HASH160
		0x14, // OP_DATA_20
		0x6b, 0x12, 0x81, 0xee, 0xc2, 0x5a, 0xb4, 0xe1,
		0xe0, 0x79, 0x3f, 0xf4, 0xe0, 0x8a, 0xb1, 0xab,
		0xb3, 0x40, 0x9c, 0xd9,
		0x88, // OP_EQUALVERIFY
		0xac, // OP_CHECKSIG
	}, &chaincfg.MainNetParams)
	MockTxsMetaData = []btc.TxModelWithInsAndOuts{
		{
			TxHash: MockBlock.Transactions[0].TxHash().String(),
			Index:  0,
			SegWit: MockBlock.Transactions[0].HasWitness(),
			TxInputs: []btc.TxInput{
				{
					Index: 0,
					SignatureScript: []byte{
						0x04, 0x4c, 0x86, 0x04, 0x1b, 0x02, 0x06, 0x02,
					},
					PreviousOutPointHash:  chainhash.Hash{}.String(),
					PreviousOutPointIndex: 0xffffffff,
				},
			},
			TxOutputs: []btc.TxOutput{
				{
					Value: 5000000000,
					Index: 0,
					PkScript: []byte{
						0x41, // OP_DATA_65
						0x04, 0x1b, 0x0e, 0x8c, 0x25, 0x67, 0xc1, 0x25,
						0x36, 0xaa, 0x13, 0x35, 0x7b, 0x79, 0xa0, 0x73,
						0xdc, 0x44, 0x44, 0xac, 0xb8, 0x3c, 0x4e, 0xc7,
						0xa0, 0xe2, 0xf9, 0x9d, 0xd7, 0x45, 0x75, 0x16,
						0xc5, 0x81, 0x72, 0x42, 0xda, 0x79, 0x69, 0x24,
						0xca, 0x4e, 0x99, 0x94, 0x7d, 0x08, 0x7f, 0xed,
						0xf9, 0xce, 0x46, 0x7c, 0xb9, 0xf7, 0xc6, 0x28,
						0x70, 0x78, 0xf8, 0x01, 0xdf, 0x27, 0x6f, 0xdf,
						0x84, // 65-byte signature
						0xac, // OP_CHECKSIG
					},
					ScriptClass:  uint8(sClass1),
					RequiredSigs: int64(numOfSigs1),
					Addresses:    stringSliceFromAddresses(addresses1),
				},
			},
		},
		{
			TxHash: MockBlock.Transactions[1].TxHash().String(),
			Index:  1,
			SegWit: MockBlock.Transactions[1].HasWitness(),
			TxInputs: []btc.TxInput{
				{
					Index: 0,
					PreviousOutPointHash: chainhash.Hash([32]byte{ // Make go vet happy.
						0x03, 0x2e, 0x38, 0xe9, 0xc0, 0xa8, 0x4c, 0x60,
						0x46, 0xd6, 0x87, 0xd1, 0x05, 0x56, 0xdc, 0xac,
						0xc4, 0x1d, 0x27, 0x5e, 0xc5, 0x5f, 0xc0, 0x07,
						0x79, 0xac, 0x88, 0xfd, 0xf3, 0x57, 0xa1, 0x87,
					}).String(),
					PreviousOutPointIndex: 0,
					SignatureScript: []byte{
						0x49, // OP_DATA_73
						0x30, 0x46, 0x02, 0x21, 0x00, 0xc3, 0x52, 0xd3,
						0xdd, 0x99, 0x3a, 0x98, 0x1b, 0xeb, 0xa4, 0xa6,
						0x3a, 0xd1, 0x5c, 0x20, 0x92, 0x75, 0xca, 0x94,
						0x70, 0xab, 0xfc, 0xd5, 0x7d, 0xa9, 0x3b, 0x58,
						0xe4, 0xeb, 0x5d, 0xce, 0x82, 0x02, 0x21, 0x00,
						0x84, 0x07, 0x92, 0xbc, 0x1f, 0x45, 0x60, 0x62,
						0x81, 0x9f, 0x15, 0xd3, 0x3e, 0xe7, 0x05, 0x5c,
						0xf7, 0xb5, 0xee, 0x1a, 0xf1, 0xeb, 0xcc, 0x60,
						0x28, 0xd9, 0xcd, 0xb1, 0xc3, 0xaf, 0x77, 0x48,
						0x01, // 73-byte signature
						0x41, // OP_DATA_65
						0x04, 0xf4, 0x6d, 0xb5, 0xe9, 0xd6, 0x1a, 0x9d,
						0xc2, 0x7b, 0x8d, 0x64, 0xad, 0x23, 0xe7, 0x38,
						0x3a, 0x4e, 0x6c, 0xa1, 0x64, 0x59, 0x3c, 0x25,
						0x27, 0xc0, 0x38, 0xc0, 0x85, 0x7e, 0xb6, 0x7e,
						0xe8, 0xe8, 0x25, 0xdc, 0xa6, 0x50, 0x46, 0xb8,
						0x2c, 0x93, 0x31, 0x58, 0x6c, 0x82, 0xe0, 0xfd,
						0x1f, 0x63, 0x3f, 0x25, 0xf8, 0x7c, 0x16, 0x1b,
						0xc6, 0xf8, 0xa6, 0x30, 0x12, 0x1d, 0xf2, 0xb3,
						0xd3, // 65-byte pubkey
					},
				},
			},
			TxOutputs: []btc.TxOutput{
				{
					Index: 0,
					Value: 556000000,
					PkScript: []byte{
						0x76, // OP_DUP
						0xa9, // OP_HASH160
						0x14, // OP_DATA_20
						0xc3, 0x98, 0xef, 0xa9, 0xc3, 0x92, 0xba, 0x60,
						0x13, 0xc5, 0xe0, 0x4e, 0xe7, 0x29, 0x75, 0x5e,
						0xf7, 0xf5, 0x8b, 0x32,
						0x88, // OP_EQUALVERIFY
						0xac, // OP_CHECKSIG
					},
					ScriptClass:  uint8(sClass2a),
					RequiredSigs: int64(numOfSigs2a),
					Addresses:    stringSliceFromAddresses(addresses2a),
				},
				{
					Index: 1,
					Value: 4444000000,
					PkScript: []byte{
						0x76, // OP_DUP
						0xa9, // OP_HASH160
						0x14, // OP_DATA_20
						0x94, 0x8c, 0x76, 0x5a, 0x69, 0x14, 0xd4, 0x3f,
						0x2a, 0x7a, 0xc1, 0x77, 0xda, 0x2c, 0x2f, 0x6b,
						0x52, 0xde, 0x3d, 0x7c,
						0x88, // OP_EQUALVERIFY
						0xac, // OP_CHECKSIG
					},
					ScriptClass:  uint8(sClass2b),
					RequiredSigs: int64(numOfSigs2b),
					Addresses:    stringSliceFromAddresses(addresses2b),
				},
			},
		},
		{
			TxHash: MockBlock.Transactions[2].TxHash().String(),
			Index:  2,
			SegWit: MockBlock.Transactions[2].HasWitness(),
			TxInputs: []btc.TxInput{
				{
					Index: 0,
					PreviousOutPointHash: chainhash.Hash([32]byte{ // Make go vet happy.
						0xc3, 0x3e, 0xbf, 0xf2, 0xa7, 0x09, 0xf1, 0x3d,
						0x9f, 0x9a, 0x75, 0x69, 0xab, 0x16, 0xa3, 0x27,
						0x86, 0xaf, 0x7d, 0x7e, 0x2d, 0xe0, 0x92, 0x65,
						0xe4, 0x1c, 0x61, 0xd0, 0x78, 0x29, 0x4e, 0xcf,
					}).String(),
					PreviousOutPointIndex: 1,
					SignatureScript: []byte{
						0x47, // OP_DATA_71
						0x30, 0x44, 0x02, 0x20, 0x03, 0x2d, 0x30, 0xdf,
						0x5e, 0xe6, 0xf5, 0x7f, 0xa4, 0x6c, 0xdd, 0xb5,
						0xeb, 0x8d, 0x0d, 0x9f, 0xe8, 0xde, 0x6b, 0x34,
						0x2d, 0x27, 0x94, 0x2a, 0xe9, 0x0a, 0x32, 0x31,
						0xe0, 0xba, 0x33, 0x3e, 0x02, 0x20, 0x3d, 0xee,
						0xe8, 0x06, 0x0f, 0xdc, 0x70, 0x23, 0x0a, 0x7f,
						0x5b, 0x4a, 0xd7, 0xd7, 0xbc, 0x3e, 0x62, 0x8c,
						0xbe, 0x21, 0x9a, 0x88, 0x6b, 0x84, 0x26, 0x9e,
						0xae, 0xb8, 0x1e, 0x26, 0xb4, 0xfe, 0x01,
						0x41, // OP_DATA_65
						0x04, 0xae, 0x31, 0xc3, 0x1b, 0xf9, 0x12, 0x78,
						0xd9, 0x9b, 0x83, 0x77, 0xa3, 0x5b, 0xbc, 0xe5,
						0xb2, 0x7d, 0x9f, 0xff, 0x15, 0x45, 0x68, 0x39,
						0xe9, 0x19, 0x45, 0x3f, 0xc7, 0xb3, 0xf7, 0x21,
						0xf0, 0xba, 0x40, 0x3f, 0xf9, 0x6c, 0x9d, 0xee,
						0xb6, 0x80, 0xe5, 0xfd, 0x34, 0x1c, 0x0f, 0xc3,
						0xa7, 0xb9, 0x0d, 0xa4, 0x63, 0x1e, 0xe3, 0x95,
						0x60, 0x63, 0x9d, 0xb4, 0x62, 0xe9, 0xcb, 0x85,
						0x0f, // 65-byte pubkey
					},
				},
			},
			TxOutputs: []btc.TxOutput{
				{
					Index: 0,
					Value: 1000000,
					PkScript: []byte{
						0x76, // OP_DUP
						0xa9, // OP_HASH160
						0x14, // OP_DATA_20
						0xb0, 0xdc, 0xbf, 0x97, 0xea, 0xbf, 0x44, 0x04,
						0xe3, 0x1d, 0x95, 0x24, 0x77, 0xce, 0x82, 0x2d,
						0xad, 0xbe, 0x7e, 0x10,
						0x88, // OP_EQUALVERIFY
						0xac, // OP_CHECKSIG
					},
					ScriptClass:  uint8(sClass3a),
					RequiredSigs: int64(numOfSigs3a),
					Addresses:    stringSliceFromAddresses(addresses3a),
				},
				{
					Index: 1,
					Value: 299000000,
					PkScript: []byte{
						0x76, // OP_DUP
						0xa9, // OP_HASH160
						0x14, // OP_DATA_20
						0x6b, 0x12, 0x81, 0xee, 0xc2, 0x5a, 0xb4, 0xe1,
						0xe0, 0x79, 0x3f, 0xf4, 0xe0, 0x8a, 0xb1, 0xab,
						0xb3, 0x40, 0x9c, 0xd9,
						0x88, // OP_EQUALVERIFY
						0xac, // OP_CHECKSIG
					},
					ScriptClass:  uint8(sClass3b),
					RequiredSigs: int64(numOfSigs3b),
					Addresses:    stringSliceFromAddresses(addresses3b),
				},
			},
		},
	}
	MockTxsMetaDataPostPublish = []btc.TxModelWithInsAndOuts{
		{
			CID:    MockTrxCID1.String(),
			MhKey:  MockTrxMhKey1,
			TxHash: MockBlock.Transactions[0].TxHash().String(),
			Index:  0,
			SegWit: MockBlock.Transactions[0].HasWitness(),
			TxInputs: []btc.TxInput{
				{
					Index: 0,
					SignatureScript: []byte{
						0x04, 0x4c, 0x86, 0x04, 0x1b, 0x02, 0x06, 0x02,
					},
					PreviousOutPointHash:  chainhash.Hash{}.String(),
					PreviousOutPointIndex: 0xffffffff,
				},
			},
			TxOutputs: []btc.TxOutput{
				{
					Value: 5000000000,
					Index: 0,
					PkScript: []byte{
						0x41, // OP_DATA_65
						0x04, 0x1b, 0x0e, 0x8c, 0x25, 0x67, 0xc1, 0x25,
						0x36, 0xaa, 0x13, 0x35, 0x7b, 0x79, 0xa0, 0x73,
						0xdc, 0x44, 0x44, 0xac, 0xb8, 0x3c, 0x4e, 0xc7,
						0xa0, 0xe2, 0xf9, 0x9d, 0xd7, 0x45, 0x75, 0x16,
						0xc5, 0x81, 0x72, 0x42, 0xda, 0x79, 0x69, 0x24,
						0xca, 0x4e, 0x99, 0x94, 0x7d, 0x08, 0x7f, 0xed,
						0xf9, 0xce, 0x46, 0x7c, 0xb9, 0xf7, 0xc6, 0x28,
						0x70, 0x78, 0xf8, 0x01, 0xdf, 0x27, 0x6f, 0xdf,
						0x84, // 65-byte signature
						0xac, // OP_CHECKSIG
					},
					ScriptClass:  uint8(sClass1),
					RequiredSigs: int64(numOfSigs1),
					Addresses:    stringSliceFromAddresses(addresses1),
				},
			},
		},
		{
			CID:    MockTrxCID2.String(),
			MhKey:  MockTrxMhKey2,
			TxHash: MockBlock.Transactions[1].TxHash().String(),
			Index:  1,
			SegWit: MockBlock.Transactions[1].HasWitness(),
			TxInputs: []btc.TxInput{
				{
					Index: 0,
					PreviousOutPointHash: chainhash.Hash([32]byte{ // Make go vet happy.
						0x03, 0x2e, 0x38, 0xe9, 0xc0, 0xa8, 0x4c, 0x60,
						0x46, 0xd6, 0x87, 0xd1, 0x05, 0x56, 0xdc, 0xac,
						0xc4, 0x1d, 0x27, 0x5e, 0xc5, 0x5f, 0xc0, 0x07,
						0x79, 0xac, 0x88, 0xfd, 0xf3, 0x57, 0xa1, 0x87,
					}).String(),
					PreviousOutPointIndex: 0,
					SignatureScript: []byte{
						0x49, // OP_DATA_73
						0x30, 0x46, 0x02, 0x21, 0x00, 0xc3, 0x52, 0xd3,
						0xdd, 0x99, 0x3a, 0x98, 0x1b, 0xeb, 0xa4, 0xa6,
						0x3a, 0xd1, 0x5c, 0x20, 0x92, 0x75, 0xca, 0x94,
						0x70, 0xab, 0xfc, 0xd5, 0x7d, 0xa9, 0x3b, 0x58,
						0xe4, 0xeb, 0x5d, 0xce, 0x82, 0x02, 0x21, 0x00,
						0x84, 0x07, 0x92, 0xbc, 0x1f, 0x45, 0x60, 0x62,
						0x81, 0x9f, 0x15, 0xd3, 0x3e, 0xe7, 0x05, 0x5c,
						0xf7, 0xb5, 0xee, 0x1a, 0xf1, 0xeb, 0xcc, 0x60,
						0x28, 0xd9, 0xcd, 0xb1, 0xc3, 0xaf, 0x77, 0x48,
						0x01, // 73-byte signature
						0x41, // OP_DATA_65
						0x04, 0xf4, 0x6d, 0xb5, 0xe9, 0xd6, 0x1a, 0x9d,
						0xc2, 0x7b, 0x8d, 0x64, 0xad, 0x23, 0xe7, 0x38,
						0x3a, 0x4e, 0x6c, 0xa1, 0x64, 0x59, 0x3c, 0x25,
						0x27, 0xc0, 0x38, 0xc0, 0x85, 0x7e, 0xb6, 0x7e,
						0xe8, 0xe8, 0x25, 0xdc, 0xa6, 0x50, 0x46, 0xb8,
						0x2c, 0x93, 0x31, 0x58, 0x6c, 0x82, 0xe0, 0xfd,
						0x1f, 0x63, 0x3f, 0x25, 0xf8, 0x7c, 0x16, 0x1b,
						0xc6, 0xf8, 0xa6, 0x30, 0x12, 0x1d, 0xf2, 0xb3,
						0xd3, // 65-byte pubkey
					},
				},
			},
			TxOutputs: []btc.TxOutput{
				{
					Index: 0,
					Value: 556000000,
					PkScript: []byte{
						0x76, // OP_DUP
						0xa9, // OP_HASH160
						0x14, // OP_DATA_20
						0xc3, 0x98, 0xef, 0xa9, 0xc3, 0x92, 0xba, 0x60,
						0x13, 0xc5, 0xe0, 0x4e, 0xe7, 0x29, 0x75, 0x5e,
						0xf7, 0xf5, 0x8b, 0x32,
						0x88, // OP_EQUALVERIFY
						0xac, // OP_CHECKSIG
					},
					ScriptClass:  uint8(sClass2a),
					RequiredSigs: int64(numOfSigs2a),
					Addresses:    stringSliceFromAddresses(addresses2a),
				},
				{
					Index: 1,
					Value: 4444000000,
					PkScript: []byte{
						0x76, // OP_DUP
						0xa9, // OP_HASH160
						0x14, // OP_DATA_20
						0x94, 0x8c, 0x76, 0x5a, 0x69, 0x14, 0xd4, 0x3f,
						0x2a, 0x7a, 0xc1, 0x77, 0xda, 0x2c, 0x2f, 0x6b,
						0x52, 0xde, 0x3d, 0x7c,
						0x88, // OP_EQUALVERIFY
						0xac, // OP_CHECKSIG
					},
					ScriptClass:  uint8(sClass2b),
					RequiredSigs: int64(numOfSigs2b),
					Addresses:    stringSliceFromAddresses(addresses2b),
				},
			},
		},
		{
			CID:    MockTrxCID3.String(),
			MhKey:  MockTrxMhKey3,
			TxHash: MockBlock.Transactions[2].TxHash().String(),
			Index:  2,
			SegWit: MockBlock.Transactions[2].HasWitness(),
			TxInputs: []btc.TxInput{
				{
					Index: 0,
					PreviousOutPointHash: chainhash.Hash([32]byte{ // Make go vet happy.
						0xc3, 0x3e, 0xbf, 0xf2, 0xa7, 0x09, 0xf1, 0x3d,
						0x9f, 0x9a, 0x75, 0x69, 0xab, 0x16, 0xa3, 0x27,
						0x86, 0xaf, 0x7d, 0x7e, 0x2d, 0xe0, 0x92, 0x65,
						0xe4, 0x1c, 0x61, 0xd0, 0x78, 0x29, 0x4e, 0xcf,
					}).String(),
					PreviousOutPointIndex: 1,
					SignatureScript: []byte{
						0x47, // OP_DATA_71
						0x30, 0x44, 0x02, 0x20, 0x03, 0x2d, 0x30, 0xdf,
						0x5e, 0xe6, 0xf5, 0x7f, 0xa4, 0x6c, 0xdd, 0xb5,
						0xeb, 0x8d, 0x0d, 0x9f, 0xe8, 0xde, 0x6b, 0x34,
						0x2d, 0x27, 0x94, 0x2a, 0xe9, 0x0a, 0x32, 0x31,
						0xe0, 0xba, 0x33, 0x3e, 0x02, 0x20, 0x3d, 0xee,
						0xe8, 0x06, 0x0f, 0xdc, 0x70, 0x23, 0x0a, 0x7f,
						0x5b, 0x4a, 0xd7, 0xd7, 0xbc, 0x3e, 0x62, 0x8c,
						0xbe, 0x21, 0x9a, 0x88, 0x6b, 0x84, 0x26, 0x9e,
						0xae, 0xb8, 0x1e, 0x26, 0xb4, 0xfe, 0x01,
						0x41, // OP_DATA_65
						0x04, 0xae, 0x31, 0xc3, 0x1b, 0xf9, 0x12, 0x78,
						0xd9, 0x9b, 0x83, 0x77, 0xa3, 0x5b, 0xbc, 0xe5,
						0xb2, 0x7d, 0x9f, 0xff, 0x15, 0x45, 0x68, 0x39,
						0xe9, 0x19, 0x45, 0x3f, 0xc7, 0xb3, 0xf7, 0x21,
						0xf0, 0xba, 0x40, 0x3f, 0xf9, 0x6c, 0x9d, 0xee,
						0xb6, 0x80, 0xe5, 0xfd, 0x34, 0x1c, 0x0f, 0xc3,
						0xa7, 0xb9, 0x0d, 0xa4, 0x63, 0x1e, 0xe3, 0x95,
						0x60, 0x63, 0x9d, 0xb4, 0x62, 0xe9, 0xcb, 0x85,
						0x0f, // 65-byte pubkey
					},
				},
			},
			TxOutputs: []btc.TxOutput{
				{
					Index: 0,
					Value: 1000000,
					PkScript: []byte{
						0x76, // OP_DUP
						0xa9, // OP_HASH160
						0x14, // OP_DATA_20
						0xb0, 0xdc, 0xbf, 0x97, 0xea, 0xbf, 0x44, 0x04,
						0xe3, 0x1d, 0x95, 0x24, 0x77, 0xce, 0x82, 0x2d,
						0xad, 0xbe, 0x7e, 0x10,
						0x88, // OP_EQUALVERIFY
						0xac, // OP_CHECKSIG
					},
					ScriptClass:  uint8(sClass3a),
					RequiredSigs: int64(numOfSigs3a),
					Addresses:    stringSliceFromAddresses(addresses3a),
				},
				{
					Index: 1,
					Value: 299000000,
					PkScript: []byte{
						0x76, // OP_DUP
						0xa9, // OP_HASH160
						0x14, // OP_DATA_20
						0x6b, 0x12, 0x81, 0xee, 0xc2, 0x5a, 0xb4, 0xe1,
						0xe0, 0x79, 0x3f, 0xf4, 0xe0, 0x8a, 0xb1, 0xab,
						0xb3, 0x40, 0x9c, 0xd9,
						0x88, // OP_EQUALVERIFY
						0xac, // OP_CHECKSIG
					},
					ScriptClass:  uint8(sClass3b),
					RequiredSigs: int64(numOfSigs3b),
					Addresses:    stringSliceFromAddresses(addresses3b),
				},
			},
		},
	}
	MockHeaderMetaData = btc.HeaderModel{
		CID:         MockHeaderCID.String(),
		MhKey:       MockHeaderMhKey,
		ParentHash:  MockBlock.Header.PrevBlock.String(),
		BlockNumber: strconv.Itoa(int(MockBlockHeight)),
		BlockHash:   MockBlock.Header.BlockHash().String(),
		Timestamp:   MockBlock.Header.Timestamp.UnixNano(),
		Bits:        MockBlock.Header.Bits,
	}
	MockConvertedPayload = btc.ConvertedPayload{
		BlockPayload: MockBlockPayload,
		TxMetaData:   MockTxsMetaData,
	}
	MockCIDPayload = btc.CIDPayload{
		HeaderCID:       MockHeaderMetaData,
		TransactionCIDs: MockTxsMetaDataPostPublish,
	}
)

func stringSliceFromAddresses(addrs []btcutil.Address) []string {
	strs := make([]string, len(addrs))
	for i, addr := range addrs {
		strs[i] = addr.EncodeAddress()
	}
	return strs
}
