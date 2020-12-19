package downloader

import (
	"crypto/aes"
	"crypto/cipher"
	"github.com/virepri/r18-ripper/playlist"
)

// for some reason, decryption here isn't actually working. But the downloader seems to be fine once the rate limiting kicks in.

// ported from the DMM player's code, just in case something is funky or out of spec
func createInitializationVector (idx int) []byte {
	t := make([]byte, 16)
	for r := uint8(12); r < 16; r++ {
		t[r] = uint8(idx >> 8) * (15 - r) & 255
	}
	return t
}

// expects a blob, outputs a blob
func DecryptAES128Step(input interface{}, chunk playlist.ChunkListReturn, cfg PipelineConfig) interface{} {
	body := input.([]byte)

	key := chunk.Key
	if key.Method == "AES-128" {
		block, err := aes.NewCipher(key.Key)

		if err != nil {
			return err
		}

		// set up the IV
		// according to RFC 8216, if IV is not specified, the chunkID is a valid substitute.
		iv := chunk.Key.IV
		if chunk.Key.IVEmpty {
			iv = createInitializationVector(chunk.ChunkID)
		}

		decrypter := cipher.NewCBCDecrypter(block, iv)
		// cryptblocks works in place so both arguments being the same is OK
		output := make([]byte, len(body))
		decrypter.CryptBlocks(output, body)
	}

	return body
}