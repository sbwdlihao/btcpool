package sserver

import (
	"testing"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcpool/gbtmaker"
	"time"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcpool/util"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"reflect"
)

type merkleBranchTest struct {
	hashes []string
	coinbase string
	merkleRoot string
}

var merkleBranchDatas = []merkleBranchTest{
	{
		// block 442428, 只有coinbase交易
		hashes: []string{},
		coinbase: "59403b96a4d72be3dce79f5bbabf16a89c4d90f8ef805165c558c68d7bc3dd7a",
		merkleRoot: "59403b96a4d72be3dce79f5bbabf16a89c4d90f8ef805165c558c68d7bc3dd7a",
	},
	{
		// block 10000， 偶数条交易
		hashes: []string{
			"fff2525b8931402dd09222c50775608f75787bd2b87e56995a7bdd30f79702c4",
			"6359f0868171b1d194cbee1af2f16ea598ae8fad666d9b012c8ed2b79a236ec4",
			"e9a66845e05d5abc0ad04ec80f774a7e585c6e8db975962d069a522137b80c1d",
		},
		coinbase: "8c14f0db3df150123e6f3dbbf30f8b955a8249b62ac1d1ff16284aefa3d06d87",
		merkleRoot: "f3e94742aca4b5ef85488dc37c06c3282295ffec960994b2c0d5ac2a25a95766",
	},
	{
		// block 100014， 奇数条交易
		hashes: []string{
			"0bcb16af267dee77ed8761662d31ee9d9a1bf1e4d268a9e7127407ebb3f9acfd",
			"48738657818e2628f216375a9d48d681e2b1b1390be1b7a028c7b810eaa3928a",
			"02981fa052f0481dbc5868f4fc2166035a10f27a03cfd2de67326471df5bc041",
			"652b0aa4cf4f17bdb31f7a1d308331bba91f3b3cbf8f39c9cb5e19d4015b9f01",
			"68d0685759c3d4f3f90a4f0e48d1b77641f06bb1f0b83a8841e8d71d5570ed41",
			"0a2a92f0bda4727d0a13eaddf4dd9ac6b5c61a1429e6b2b818f19b15df0ac154",
		},
		coinbase: "147caa76786596590baa4e98f5d9f48b86c7765e489f7a6ff3360fe5c674360b",
		merkleRoot: "8772d9d0fdf8c1303c7b1167e3c73b095fd970e33c799c6563d98b2e96c5167f",
	},
}

func TestMerkleBranch(t *testing.T) {
	for _, data := range merkleBranchDatas {
		hashes := data.hashes
		transactionHashes := make([]*chainhash.Hash, len(hashes))
		for i, hash := range hashes {
			th, _ := chainhash.NewHashFromStr(hash)
			transactionHashes[i] = th
		}
		branches := makeMerkleBranch(transactionHashes)
		coinbase, _ := chainhash.NewHashFromStr(data.coinbase)
		txHashes := make([]*chainhash.Hash, len(branches) + 1)
		txHashes[0] = coinbase
		for i, b := range branches {
			txHashes[i+1], _ = chainhash.NewHashFromStr(b)
		}
		for len(txHashes) > 1 {
			txHashes[0] = blockchain.HashMerkleBranches(txHashes[0], txHashes[1])
			for i:=1; i < len(txHashes)-1; i++ {
				txHashes[i] = txHashes[i+1]
			}
			txHashes = txHashes[:len(txHashes)-1]
		}
		if txHashes[0].String() != data.merkleRoot {
			t.Errorf("expected %s, got %s", data.merkleRoot, txHashes[0])
		}
	}
}

var cv int64 = 5000000000
type jobTest struct {
	bk *btcjson.GetBlockTemplateResult
	version int32
	targetVersion int32
	preHashBe string
	merkleBranch []string
	coinbase1 string
	coinbase2 string
}
var jobTests = []jobTest{
	// 空块
	{
		bk: &btcjson.GetBlockTemplateResult{
			Bits: "207fffff",
			CurTime: 1481198367,
			Height: 442457,
			PreviousHash: "53dae1de0cf115cbaec8d48893b891231f3907ff5878ede3cde6959000f52076",
			SigOpLimit: 20000,
			SizeLimit: 1000000,
			Transactions: []btcjson.GetBlockTemplateResultTx{},
			Version: 536870912,
			CoinbaseAux: &btcjson.GetBlockTemplateResultAux{
				Flags: "0a2f454231362f4144342f",
			},
			CoinbaseValue: &cv,
			LongPollID: "53dae1de0cf115cbaec8d48893b891231f3907ff5878ede3cde6959000f5207623",
			Target: "7fffff0000000000000000000000000000000000000000000000000000000000",
			MinTime: 1480851746,
			Mutable: []string{"time","transactions","prevblock"},
			NonceRange: "00000000ffffffff",
			Capabilities: []string{"proposal"},
		},
		version: 2,
		targetVersion: 2,
		preHashBe: "00f52076cde695905878ede31f3907ff93b89123aec8d4880cf115cb53dae1de",
		merkleBranch: nil,
		coinbase1: "01000000010000000000000000000000000000000000000000000000000000000000000000ffffffff1e0359c006041f4b49582f4254432e434f4d2f",
		coinbase2: "ffffffff0100f2052a0100000017a914e083685a1097ce1ea9e91987ab9e94eae33d8a138700000000",
	},
	// 奇数条交易
	{
		bk: &btcjson.GetBlockTemplateResult{
			Bits: "207fffff",
			CurTime: 1481205390,
			Height: 442457,
			PreviousHash: "53dae1de0cf115cbaec8d48893b891231f3907ff5878ede3cde6959000f52076",
			SigOpLimit: 20000,
			SizeLimit: 1000000,
			Transactions: []btcjson.GetBlockTemplateResultTx{
				{
					Data: "01000000013d6171c20312373e7e2bb7ea07d3f2176abc1dd114af92666e0570c8f7192177000000004948304502210092b7951fde9c5fa4161abb22265a9c7647d3774a22874ff251d9fa99a21e894c022010d6c2f0eae1d701bda97ff9cbfe29f55a5e982b337d5ae9a3d8379693d7f36501feffffff0200021024010000001976a914f8b500f578e602cd9021e6809ac9f22687a363b888ac00e1f505000000001976a914991d0461842b386b74616f79011979f4e666ff3788ac42000000",
					Hash: "605adc0cfd155da6c9902753123160bb7a8934d7105d6d346eb619f39098195f",
					Depends: []int64{},
					Fee: 3840,
					SigOps: 2,
				},
			},
			Version: 536870912,
			CoinbaseAux: &btcjson.GetBlockTemplateResultAux{
				Flags: "0a2f454231362f4144342f",
			},
			CoinbaseValue: &cv,
			LongPollID: "53dae1de0cf115cbaec8d48893b891231f3907ff5878ede3cde6959000f5207623",
			Target: "7fffff0000000000000000000000000000000000000000000000000000000000",
			MinTime: 1480851746,
			Mutable: []string{"time","transactions","prevblock"},
			NonceRange: "00000000ffffffff",
			Capabilities: []string{"proposal"},
		},
		version: 0,
		targetVersion: 536870912,
		preHashBe: "00f52076cde695905878ede31f3907ff93b89123aec8d4880cf115cb53dae1de",
		merkleBranch: []string{
			"605adc0cfd155da6c9902753123160bb7a8934d7105d6d346eb619f39098195f",
		},
		coinbase1: "01000000010000000000000000000000000000000000000000000000000000000000000000ffffffff1e0359c006048e6649582f4254432e434f4d2f",
		coinbase2: "ffffffff0100f2052a0100000017a914e083685a1097ce1ea9e91987ab9e94eae33d8a138700000000",
	},
	// 偶数条交易
	{
		bk: &btcjson.GetBlockTemplateResult{
			Bits: "207fffff",
			CurTime: 1481205390,
			Height: 442457,
			PreviousHash: "53dae1de0cf115cbaec8d48893b891231f3907ff5878ede3cde6959000f52076",
			SigOpLimit: 20000,
			SizeLimit: 1000000,
			Transactions: []btcjson.GetBlockTemplateResultTx{
				{
					Data: "01000000013d6171c20312373e7e2bb7ea07d3f2176abc1dd114af92666e0570c8f7192177000000004948304502210092b7951fde9c5fa4161abb22265a9c7647d3774a22874ff251d9fa99a21e894c022010d6c2f0eae1d701bda97ff9cbfe29f55a5e982b337d5ae9a3d8379693d7f36501feffffff0200021024010000001976a914f8b500f578e602cd9021e6809ac9f22687a363b888ac00e1f505000000001976a914991d0461842b386b74616f79011979f4e666ff3788ac42000000",
					Hash: "605adc0cfd155da6c9902753123160bb7a8934d7105d6d346eb619f39098195f",
					Depends: []int64{},
					Fee: 3840,
					SigOps: 2,
				},
				{
					Data: "01000000015f199890f319b66e346d5d10d734897abb603112532790c9a65d15fd0cdc5a60000000006a47304402201dba47fb8ab224f0eb8192975d54118f897e47fe433e4921040281d71ecc570702205ddd3ff8c5849131829ca60a07dff80e2a859123a6f14e3924aae11ebcb653df0121026b8aa8e9305fd900d487b71eb4467f35c680da6d5f9244b60a265574ddd02d5efeffffff026c0f1a1e010000001976a914bf1fd67e3bfa4fa7257e0adfbc068b19d95da86488ac00e1f505000000001976a914991d0461842b386b74616f79011979f4e666ff3788ac66000000",
					Hash: "6aa04f0d94d2a7965c3d036eb03a0e90ef7cc62aa9e4bb80efa72b9796f0d513",
					Depends: []int64{1},
					Fee: 4500,
					SigOps: 2,
				},
			},
			Version: 536870912,
			CoinbaseAux: &btcjson.GetBlockTemplateResultAux{
				Flags: "0a2f454231362f4144342f",
			},
			CoinbaseValue: &cv,
			LongPollID: "53dae1de0cf115cbaec8d48893b891231f3907ff5878ede3cde6959000f5207623",
			Target: "7fffff0000000000000000000000000000000000000000000000000000000000",
			MinTime: 1480851746,
			Mutable: []string{"time","transactions","prevblock"},
			NonceRange: "00000000ffffffff",
			Capabilities: []string{"proposal"},
		},
		version: 0,
		targetVersion: 536870912,
		preHashBe: "00f52076cde695905878ede31f3907ff93b89123aec8d4880cf115cb53dae1de",
		merkleBranch: []string{
			"605adc0cfd155da6c9902753123160bb7a8934d7105d6d346eb619f39098195f",
			"1dca84e954428844fad7daefefe1623727f9ce48ad29d023b54715e97bdd6896",
		},
		coinbase1: "01000000010000000000000000000000000000000000000000000000000000000000000000ffffffff1e0359c006048e6649582f4254432e434f4d2f",
		coinbase2: "ffffffff0100f2052a0100000017a914e083685a1097ce1ea9e91987ab9e94eae33d8a138700000000",
	},
}

func TestInitFromGbt(t *testing.T) {
	payoutAddress, _ := btcutil.DecodeAddress("3NA8hsjfdgVkmmVS9moHmkZsVCoLxUkvvv", &chaincfg.MainNetParams)
	for _, jt := range jobTests {
		hash, _ := util.HashBlockTemplateResult(jt.bk)
		rawGbt := &gbtmaker.RawGbt{
			CreatedAt: time.Now(),
			BlockTemplate: jt.bk,
			Hash: hash,
		}
		job, err := InitFromGbt(rawGbt, "/BTC.COM/", payoutAddress, jt.version)
		if err != nil {
			t.Error(err)
		}
		if job.Version != jt.targetVersion {
			t.Errorf("Version expected %d, got %d", jt.targetVersion, job.Version)
		}
		if job.PrevHashBe != jt.preHashBe {
			t.Errorf("PrevHashBe expected %s, got %s", jt.preHashBe, job.PrevHashBe)
		}
		if !reflect.DeepEqual(job.MerkleBranch, jt.merkleBranch) {
			t.Errorf("MerkleBranch expected %v, got %v", jt.merkleBranch, job.MerkleBranch)
		}
		if job.Coinbase1 != jt.coinbase1 {
			t.Errorf("Coinbase1 expected %s, got %s", jt.coinbase1, job.Coinbase1)
		}
		if job.Coinbase2 != jt.coinbase2 {
			t.Errorf("Coinbase2 expected %s, got %s", jt.coinbase2, job.Coinbase2)
		}
	}
}