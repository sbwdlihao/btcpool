package sserver

import (
	"github.com/btcpool/gbtmaker"
	"github.com/btcsuite/btcutil"
	"math/big"
	"github.com/btcsuite/btcd/blockchain"
	"strconv"
	"encoding/hex"
	"container/list"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcd/txscript"
	"errors"
	"bytes"
	"github.com/btcpool/util"
)

type StratumJob struct {
	JobId         string
	Height        int64
	PrevHash      string
	PrevHashBe    string              // big endian
	CoinbaseValue *int64
	MerkleBranch  []string
	Version       int32
	Target        *big.Int `json:"-"` // omit
	NBits         int64
	NTime         int64
	MinTime       int64
	Coinbase1     string
	Coinbase2     string
}

func InitFromGbt(rawgbt *gbtmaker.RawGbt, poolCoinbaseInfo string, poolPayoutAddr btcutil.Address, blockVersion int32) (*StratumJob, error) {
	job := &StratumJob{}
	job.JobId = rawgbt.Hash
	job.Height = rawgbt.BlockTemplate.Height
	job.PrevHash = rawgbt.BlockTemplate.PreviousHash
	hr, err := util.ReserveHash(job.PrevHash)
	if err != nil {
		return nil, err
	}
	job.PrevHashBe = hr
	//fmt.Println(job.PrevHash)
	//fmt.Println(job.PrevHashBe)
	if blockVersion != 0 {
		job.Version = blockVersion
	} else {
		job.Version = rawgbt.BlockTemplate.Version
	}
	job.NBits, _ = strconv.ParseInt(rawgbt.BlockTemplate.Bits, 16, 64)
	//fmt.Println(job.NBits) 545259519
	job.NTime = rawgbt.BlockTemplate.CurTime
	job.MinTime = rawgbt.BlockTemplate.MinTime
	job.CoinbaseValue = rawgbt.BlockTemplate.CoinbaseValue
	job.Target = blockchain.CompactToBig(uint32(job.NBits))
	//fmt.Println(job.Target) 57896037716911750921221705069588091649609539881711309849342236841432341020672
	txHashes := make([]*chainhash.Hash, len(rawgbt.BlockTemplate.Transactions))
	for i, tx := range rawgbt.BlockTemplate.Transactions {
		txHashes[i], _ = chainhash.NewHashFromStr(tx.Hash)
	}
	job.MerkleBranch = makeMerkleBranch(txHashes)
	//fmt.Println(job.MerkleBranch) []

	// make coinbase1 & coinbase2
	var tx *wire.MsgTx = new(wire.MsgTx)
	tx.Version = wire.TxVersion
	tx.LockTime = 0
	//
	// block height, 4 bytes in script: 0x03xxxxxx
	// https://github.com/bitcoin/bips/blob/master/bip-0034.mediawiki
	// https://github.com/bitcoin/bitcoin/pull/1526
	// example 442457=>0359c006
	scriptBuild := txscript.NewScriptBuilder()
	scriptBuild.AddInt64(job.Height)
	// add current timestamp to coinbase tx input, so if the block's merkle root
	// hash is the same, there's no risk for miners to calc the same space.
	// https://github.com/btccom/btcpool/issues/5
	//
	// 5 bytes in script: 0x04xxxxxxxx.
	// eg. 0x0402363d58 -> 0x583d3602 = 1480406530 = 2016-11-29 16:02:10
	//
	scriptBuild.AddInt64(rawgbt.BlockTemplate.CurTime)
	txInScript, err := scriptBuild.Script()
	if err != nil {
		return nil, err
	}
	//fmt.Println(txInScript) [1 103 4 231 82 73 88]
	// pool's info, example /BTC.COM/ => 2f4254432e434f4d2f
	txInScript = append(txInScript, []byte(poolCoinbaseInfo)...)
	//
	// bitcoind/src/main.cpp: CheckTransaction()
	//   if (tx.IsCoinBase())
	//   {
	//     if (tx.vin[0].scriptSig.size() < 2 || tx.vin[0].scriptSig.size() > 100)
	//       return state.DoS(100, false, REJECT_INVALID, "bad-cb-length");
	//   }
	//
	// 100: coinbase script sig max len, range: (2, 100).
	//  12: extra nonce1 (4bytes) + extra nonce2 (8bytes)
	//
	placeHolder := [12]byte{0xEE,0xEE,0xEE,0xEE,0xEE,0xEE,0xEE,0xEE,0xEE,0xEE,0xEE,0xEE}
	maxScriptSigLen := 100 - len(placeHolder)
	if len(txInScript) > maxScriptSigLen {
		txInScript = txInScript[:maxScriptSigLen]
	}
	txInScript = append(txInScript, placeHolder[:]...)
	txInPreOutPoint := wire.NewOutPoint(&chainhash.Hash{}, wire.MaxTxInSequenceNum)
	txIn := wire.NewTxIn(txInPreOutPoint, txInScript)
	tx.AddTxIn(txIn)
	pkScript, err := txscript.PayToAddrScript(poolPayoutAddr)
	if err != nil {
		return nil, err
	}
	txOut := wire.NewTxOut(*job.CoinbaseValue, pkScript)
	tx.AddTxOut(txOut)

	var buf bytes.Buffer
	if err := tx.Serialize(&buf); err != nil {
		return nil, err
	}
	bufBytes := buf.Bytes()

	extraNonceStart := bytes.LastIndex(bufBytes, placeHolder[:])
	if extraNonceStart == -1 {
		return nil, errors.New("placeHolder find failed")
	}
	job.Coinbase1 = hex.EncodeToString(bufBytes[:extraNonceStart])
	job.Coinbase2 = hex.EncodeToString(bufBytes[extraNonceStart+len(placeHolder):])
	return job, nil
}

func makeMerkleBranch(txHashes []*chainhash.Hash) []string {
	if len(txHashes) == 0 {
		return nil
	}
	branches_list := list.New()
	for len(txHashes) > 1 {
		branches_list.PushBack(txHashes[0])
		if len(txHashes) % 2 == 0 {
			txHashes = append(txHashes, txHashes[len(txHashes)-1])
		}
		for i := 0; i < (len(txHashes) - 1) / 2; i++ {
			txHashes[i] = blockchain.HashMerkleBranches(txHashes[i*2+1], txHashes[i*2+2])
		}
		txHashes = txHashes[:(len(txHashes) - 1) / 2]
	}
	branches_list.PushBack(txHashes[0])
	branches := make([]string, branches_list.Len())
	i := 0
	for e := branches_list.Front(); e != nil; e = e.Next() {
		branches[i] = e.Value.(*chainhash.Hash).String()
		i++
	}
	return branches
}
