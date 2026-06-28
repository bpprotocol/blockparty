package derive

import (
	"encoding/hex"
	"strconv"

	"github.com/bpprotocol/blockparty/sdk/crypto"
)

// GetBlockID derives the content-binding identifier for a block:
//
//	dataHash = hex(Keccak256(data))
//	message  = "block.v"+version+":"+ts+"/"+address+"/"+typeCode+"/"+dataHash
//	id       = hex(SHA256(HMAC_SHA256(key=audienceCode, message)))
//
// Including the payload hash makes the ID change whenever data changes, so the
// ID commits to the block's contents.
func GetBlockID(version uint32, timestamp int64, audienceCode Code, address Address, typeCode Code, data []byte) string {
	dataHash := hex.EncodeToString(crypto.Keccak256(data))
	message := "block.v" + strconv.FormatUint(uint64(version), 10) +
		":" + strconv.FormatInt(timestamp, 10) +
		"/" + string(address) +
		"/" + typeCode.Hex() +
		"/" + dataHash
	seed := crypto.HMACSHA256([]byte(audienceCode.Hex()), []byte(message))
	return hex.EncodeToString(crypto.Sha256(seed))
}
