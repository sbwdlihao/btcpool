package util

import (
	"github.com/btcsuite/btcd/btcjson"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
)

func HashBlockTemplateResult(result *btcjson.GetBlockTemplateResult) (string, error) {
	b, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	hash.Write(b)
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func ReserveHash(hash string) (string, error)  {
	hb, err := hex.DecodeString(hash)
	if err != nil {
		return "", err
	}
	if len(hb) != 32 {
		return "", errors.New("hash lenght must be 64")
	}
	for j := 0; j < 4; j++ {
		i := j * 4
		hb[i], hb[i+1], hb[i+2], hb[i+3], hb[28-i], hb[29-i], hb[30-i], hb[31-i] = hb[28-i], hb[29-i], hb[30-i], hb[31-i], hb[i], hb[i+1], hb[i+2], hb[i+3]
	}
	return hex.EncodeToString(hb), nil
}
