package util

import (
	"github.com/btcsuite/btcd/btcjson"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
