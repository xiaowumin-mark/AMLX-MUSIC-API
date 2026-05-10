package qqmusic

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"unicode/utf8"
)

// QrcDecrypt decrypts a hex-encoded QQ Music QRC lyrics string.
//
// Algorithm (from Unilyric qrc_codec.rs):
//  1. Hex decode
//  2. Custom 3DES decrypt (non-standard DES with custom S-boxes, IP, InvIP)
//  3. Zlib decompress
//  4. Strip UTF-8 BOM if present
func QrcDecrypt(encryptedHex string) (string, error) {
	data, err := hex.DecodeString(encryptedHex)
	if err != nil {
		return "", fmt.Errorf("qrc decrypt hex: %w", err)
	}
	return decryptQrcBytes(data)
}

// QrcDecryptLocal decrypts a local .qrc binary file.
//
// Algorithm:
//  1. QMC1 stream cipher decryption
//  2. Strip 11-byte header
//  3. Custom 3DES decrypt
//  4. Zlib decompress
func QrcDecryptLocal(data []byte) (string, error) {
	qmc1Decrypt(data)
	if len(data) < 11 {
		return "", fmt.Errorf("qrc local: data too short (%d bytes)", len(data))
	}
	return decryptQrcBytes(data[11:])
}

// QrcDecryptWithFallback tries Base64 decode first, then falls back to
// the custom DES decryption if Base64 produces non-UTF-8 output.
func QrcDecryptWithFallback(encryptedStr string) (string, error) {
	// Try Base64 first
	if decoded, err := base64.StdEncoding.DecodeString(encryptedStr); err == nil {
		if utf8.Valid(decoded) {
			return string(decoded), nil
		}
	}
	return QrcDecrypt(encryptedStr)
}

// ExtractFromQRcWrapper extracts the LyricContent attribute from an
// XML-wrapped QRC lyrics string.
func ExtractFromQRcWrapper(xmlText string) string {
	if xmlText == "" {
		return ""
	}
	if !bytes.HasPrefix([]byte(xmlText), []byte("<?xml")) {
		return xmlText
	}
	// Simple extraction: find LyricContent="..."
	start := indexOf(xmlText, `LyricContent="`)
	if start == -1 {
		return xmlText
	}
	start += len(`LyricContent="`)
	end := start
	for end < len(xmlText) && xmlText[end] != '"' {
		end++
	}
	content := xmlText[start:end]
	// Unescape XML entities
	content = unescapeXML(content)
	return content
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func unescapeXML(s string) string {
	s = replaceAll(s, "&quot;", `"`)
	s = replaceAll(s, "&amp;", "&")
	s = replaceAll(s, "&lt;", "<")
	s = replaceAll(s, "&gt;", ">")
	s = replaceAll(s, "&apos;", "'")
	return s
}

func replaceAll(s, old, new string) string {
	result := ""
	for i := 0; i < len(s); i++ {
		if i+len(old) <= len(s) && s[i:i+len(old)] == old {
			result += new
			i += len(old) - 1
		} else {
			result += string(s[i])
		}
	}
	return result
}

// ------- Internal decryption pipeline -------

func decryptQrcBytes(data []byte) (string, error) {
	if len(data)%8 != 0 {
		return "", fmt.Errorf("qrc decrypt: data length %d not multiple of 8", len(data))
	}
	decrypted := make([]byte, len(data))
	for i := 0; i < len(data); i += 8 {
		qqTripleDESDecrypt(data[i:i+8], decrypted[i:i+8])
	}
	decompressed, err := zlibDecompress(decrypted)
	if err != nil {
		return "", fmt.Errorf("qrc decrypt zlib: %w", err)
	}
	// Strip BOM
	if len(decompressed) >= 3 && decompressed[0] == 0xEF && decompressed[1] == 0xBB && decompressed[2] == 0xBF {
		decompressed = decompressed[3:]
	}
	return string(decompressed), nil
}

func zlibDecompress(data []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

// ------- Non-standard Custom DES -------

// Custom DES keys for QRC 3DES decrypt pipeline.
var (
	qrcKey1 = [8]byte{'!', '@', '#', ')', '(', '*', '$', '%'} // !@#)(*$%
	qrcKey2 = [8]byte{'1', '2', '3', 'Z', 'X', 'C', '!', '@'} // 123ZXC!@
	qrcKey3 = [8]byte{'!', '@', '#', ')', '(', 'N', 'H', 'L'} // !@#)(NHL
)

// Build decrypt key schedules. The 3DES pipeline uses:
//
//	KEY_3 in Decrypt mode, KEY_2 in Encrypt mode, KEY_1 in Decrypt mode
var (
	qrcDecryptSchedule = [3][16][6]byte{
		desKeySchedule(qrcKey3, false), // decrypt
		desKeySchedule(qrcKey2, true),  // encrypt
		desKeySchedule(qrcKey1, false), // decrypt
	}
)

func qqTripleDESDecrypt(input, output []byte) {
	var temp1, temp2 [8]byte
	desBlock(input[:], temp1[:], qrcDecryptSchedule[0])
	desBlock(temp1[:], temp2[:], qrcDecryptSchedule[1])
	desBlock(temp2[:], output, qrcDecryptSchedule[2])
}

// desBlock performs one DES block transformation.
func desBlock(input []byte, output []byte, key [16][6]byte) {
	var state [2]uint32
	initialPermutation(input, &state)
	for i := 0; i < 15; i++ {
		prevRight := state[1]
		prevLeft := state[0]
		state[1] = prevLeft ^ fFunction(prevRight, key[i])
		state[0] = prevRight
	}
	state[0] ^= fFunction(state[1], key[15])
	inversePermutation(state, output)
}

// fFunction is the DES Feistel function using custom S-boxes.
func fFunction(state uint32, key [6]byte) uint32 {
	// Expansion to 48 bits
	expanded := expandE(state)

	// XOR with round key (48 bits)
	keyU64 := uint64(key[0])<<40 | uint64(key[1])<<32 | uint64(key[2])<<24 |
		uint64(key[3])<<16 | uint64(key[4])<<8 | uint64(key[5])
	xored := expanded ^ keyU64

	// S-box substitution + P-box permutation (merged via SP tables)
	result := spTable[0][(xored>>42)&0x3F] |
		spTable[1][(xored>>36)&0x3F] |
		spTable[2][(xored>>30)&0x3F] |
		spTable[3][(xored>>24)&0x3F] |
		spTable[4][(xored>>18)&0x3F] |
		spTable[5][(xored>>12)&0x3F] |
		spTable[6][(xored>>6)&0x3F] |
		spTable[7][xored&0x3F]

	return result
}

// expandE applies the E-box expansion: 32 bits → 48 bits.
func expandE(input uint32) uint64 {
	var output uint64
	for i, pos := range eBoxTable {
		shift := uint(32 - pos)
		bit := (input >> shift) & 1
		output |= uint64(bit) << uint(47-i)
	}
	return output
}

// initialPermutation applies the non-standard IP.
func initialPermutation(input []byte, state *[2]uint32) {
	state[0] = 0
	state[1] = 0
	norm := uint64(input[0])<<56 | uint64(input[1])<<48 | uint64(input[2])<<40 |
		uint64(input[3])<<32 | uint64(input[4])<<24 | uint64(input[5])<<16 |
		uint64(input[6])<<8 | uint64(input[7])
	for i := 0; i < 64; i++ {
		srcPos := ipRule[i] - 1
		bit := (norm >> uint(63-srcPos)) & 1
		if i < 32 {
			state[0] |= uint32(bit) << uint(31-i)
		} else {
			state[1] |= uint32(bit) << uint(63-i)
		}
	}
}

// inversePermutation applies the non-standard inverse IP.
func inversePermutation(state [2]uint32, output []byte) {
	norm := uint64(state[0])<<32 | uint64(state[1])
	var result uint64
	for i := 0; i < 64; i++ {
		srcPos := invIPRule[i] - 1
		bit := (norm >> uint(63-srcPos)) & 1
		result |= bit << uint(63-i)
	}
	output[0] = byte(result >> 56)
	output[1] = byte(result >> 48)
	output[2] = byte(result >> 40)
	output[3] = byte(result >> 32)
	output[4] = byte(result >> 24)
	output[5] = byte(result >> 16)
	output[6] = byte(result >> 8)
	output[7] = byte(result)
}

// Non-standard IP rule.
var ipRule = [64]byte{
	34, 42, 50, 58, 2, 10, 18, 26,
	36, 44, 52, 60, 4, 12, 20, 28,
	38, 46, 54, 62, 6, 14, 22, 30,
	40, 48, 56, 64, 8, 16, 24, 32,
	33, 41, 49, 57, 1, 9, 17, 25,
	35, 43, 51, 59, 3, 11, 19, 27,
	37, 45, 53, 61, 5, 13, 21, 29,
	39, 47, 55, 63, 7, 15, 23, 31,
}

// Non-standard inverse IP rule.
var invIPRule = [64]byte{
	37, 5, 45, 13, 53, 21, 61, 29,
	38, 6, 46, 14, 54, 22, 62, 30,
	39, 7, 47, 15, 55, 23, 63, 31,
	40, 8, 48, 16, 56, 24, 64, 32,
	33, 1, 41, 9, 49, 17, 57, 25,
	34, 2, 42, 10, 50, 18, 58, 26,
	35, 3, 43, 11, 51, 19, 59, 27,
	36, 4, 44, 12, 52, 20, 60, 28,
}

// Standard E-box expansion table (1-based positions).
var eBoxTable = [48]byte{
	32, 1, 2, 3, 4, 5,
	4, 5, 6, 7, 8, 9,
	8, 9, 10, 11, 12, 13,
	12, 13, 14, 15, 16, 17,
	16, 17, 18, 19, 20, 21,
	20, 21, 22, 23, 24, 25,
	24, 25, 26, 27, 28, 29,
	28, 29, 30, 31, 32, 1,
}

// Standard P-box permutation table (1-based positions).
var pBoxTable = [32]byte{
	16, 7, 20, 21, 29, 12, 28, 17,
	1, 15, 23, 26, 5, 18, 31, 10,
	2, 8, 24, 14, 32, 27, 3, 9,
	19, 13, 30, 6, 22, 11, 4, 25,
}

// Custom S-boxes (from Rust reference - SBOX2/SBOX4 have duplicate values).
var sBoxes = [8][64]byte{
	// SBOX1
	{14, 4, 13, 1, 2, 15, 11, 8, 3, 10, 6, 12, 5, 9, 0, 7,
		0, 15, 7, 4, 14, 2, 13, 1, 10, 6, 12, 11, 9, 5, 3, 8,
		4, 1, 14, 8, 13, 6, 2, 11, 15, 12, 9, 7, 3, 10, 5, 0,
		15, 12, 8, 2, 4, 9, 1, 7, 5, 11, 3, 14, 10, 0, 6, 13},
	// SBOX2 (non-standard: 15 appears twice)
	{15, 1, 8, 14, 6, 11, 3, 4, 9, 7, 2, 13, 12, 0, 5, 10,
		3, 13, 4, 7, 15, 2, 8, 15, 12, 0, 1, 10, 6, 9, 11, 5,
		0, 14, 7, 11, 10, 4, 13, 1, 5, 8, 12, 6, 9, 3, 2, 15,
		13, 8, 10, 1, 3, 15, 4, 2, 11, 6, 7, 12, 0, 5, 14, 9},
	// SBOX3
	{10, 0, 9, 14, 6, 3, 15, 5, 1, 13, 12, 7, 11, 4, 2, 8,
		13, 7, 0, 9, 3, 4, 6, 10, 2, 8, 5, 14, 12, 11, 15, 1,
		13, 6, 4, 9, 8, 15, 3, 0, 11, 1, 2, 12, 5, 10, 14, 7,
		1, 10, 13, 0, 6, 9, 8, 7, 4, 15, 14, 3, 11, 5, 2, 12},
	// SBOX4 (non-standard: 10 appears twice)
	{7, 13, 14, 3, 0, 6, 9, 10, 1, 2, 8, 5, 11, 12, 4, 15,
		13, 8, 11, 5, 6, 15, 0, 3, 4, 7, 2, 12, 1, 10, 14, 9,
		10, 6, 9, 0, 12, 11, 7, 13, 15, 1, 3, 14, 5, 2, 8, 4,
		3, 15, 0, 6, 10, 10, 13, 8, 9, 4, 5, 11, 12, 7, 2, 14},
	// SBOX5
	{2, 12, 4, 1, 7, 10, 11, 6, 8, 5, 3, 15, 13, 0, 14, 9,
		14, 11, 2, 12, 4, 7, 13, 1, 5, 0, 15, 10, 3, 9, 8, 6,
		4, 2, 1, 11, 10, 13, 7, 8, 15, 9, 12, 5, 6, 3, 0, 14,
		11, 8, 12, 7, 1, 14, 2, 13, 6, 15, 0, 9, 10, 4, 5, 3},
	// SBOX6
	{12, 1, 10, 15, 9, 2, 6, 8, 0, 13, 3, 4, 14, 7, 5, 11,
		10, 15, 4, 2, 7, 12, 9, 5, 6, 1, 13, 14, 0, 11, 3, 8,
		9, 14, 15, 5, 2, 8, 12, 3, 7, 0, 4, 10, 1, 13, 11, 6,
		4, 3, 2, 12, 9, 5, 15, 10, 11, 14, 1, 7, 6, 0, 8, 13},
	// SBOX7
	{4, 11, 2, 14, 15, 0, 8, 13, 3, 12, 9, 7, 5, 10, 6, 1,
		13, 0, 11, 7, 4, 9, 1, 10, 14, 3, 5, 12, 2, 15, 8, 6,
		1, 4, 11, 13, 12, 3, 7, 14, 10, 15, 6, 8, 0, 5, 9, 2,
		6, 11, 13, 8, 1, 4, 10, 7, 9, 5, 0, 15, 14, 2, 3, 12},
	// SBOX8
	{13, 2, 8, 4, 6, 15, 11, 1, 10, 9, 3, 14, 5, 0, 12, 7,
		1, 15, 13, 8, 10, 3, 7, 4, 12, 5, 6, 11, 0, 14, 9, 2,
		7, 11, 4, 1, 9, 12, 14, 2, 0, 6, 10, 13, 15, 3, 5, 8,
		2, 1, 14, 7, 4, 10, 8, 13, 15, 12, 9, 0, 3, 5, 6, 11},
}

// Precomputed SP tables (merged S-box + P-box for efficiency).
// spTable[box_idx][6-bit input] = 32-bit result after S→P.
var spTable [8][64]uint32

func init() {
	// Build SP tables at init time.
	for boxIdx := 0; boxIdx < 8; boxIdx++ {
		for input := uint32(0); input < 64; input++ {
			sBoxIndex := calcSBoxIndex(byte(input))
			fourBit := uint32(sBoxes[boxIdx][sBoxIndex])
			preP := fourBit << (28 - uint(boxIdx*4))
			spTable[boxIdx][input] = applyPBox(preP)
		}
	}
}

func calcSBoxIndex(a byte) int {
	return int((a & 0x20) | ((a & 0x1f) >> 1) | ((a & 0x01) << 4))
}

func applyPBox(input uint32) uint32 {
	var result uint32
	for i, pos := range pBoxTable {
		bit := (input >> (32 - uint(pos))) & 1
		result |= bit << (31 - uint(i))
	}
	return result
}

// ------- Key Schedule -------

// PC-1 and PC-2 permutation tables (standard).
var keyPermC = [28]byte{
	56, 48, 40, 32, 24, 16, 8,
	0, 57, 49, 41, 33, 25, 17,
	9, 1, 58, 50, 42, 34, 26,
	18, 10, 2, 59, 51, 43, 35,
}

var keyPermD = [28]byte{
	62, 54, 46, 38, 30, 22, 14,
	6, 61, 53, 45, 37, 29, 21,
	13, 5, 60, 52, 44, 36, 28,
	20, 12, 4, 27, 19, 11, 3,
}

var keyCompression = [48]byte{
	13, 16, 10, 23, 0, 4, 2, 27,
	14, 5, 20, 9, 22, 18, 11, 3,
	25, 7, 15, 6, 26, 19, 12, 1,
	40, 51, 30, 36, 46, 54, 29, 39,
	50, 44, 32, 47, 43, 48, 38, 55,
	33, 52, 45, 41, 49, 35, 28, 31,
}

var keyRndShift = [16]uint32{1, 1, 2, 2, 2, 2, 2, 2, 1, 2, 2, 2, 2, 2, 2, 1}

// desKeySchedule generates 16 round keys from a 64-bit DES key.
// encrypt=true for encryption, false for decryption (reversed order).
func desKeySchedule(key [8]byte, encrypt bool) [16][6]byte {
	var schedule [16][6]byte

	// Apply PC-1
	c0 := permuteFromKey(key, keyPermC[:])
	d0 := permuteFromKey(key, keyPermD[:])

	// Shift left 4 bits for 28-bit alignment in uint32
	c := (uint32(c0)) << 4
	d := (uint32(d0)) << 4

	for i := 0; i < 16; i++ {
		shift := keyRndShift[i]
		c = rotateLeft28(c, shift)
		d = rotateLeft28(d, shift)

		roundIdx := i
		if !encrypt {
			roundIdx = 15 - i
		}

		// Apply PC-2
		var subkey uint64
		for k := 0; k < 48; k++ {
			pos := int(keyCompression[k])
			var bit uint64
			if pos < 28 {
				bit = uint64((c >> (31 - uint(pos))) & 1)
			} else {
				// QQ-specific quirk: pos - 27 instead of pos - 28
				bit = uint64((d >> (31 - uint(pos-27))) & 1)
			}
			if bit != 0 {
				subkey |= 1 << (47 - k)
			}
		}

		b := subkeyToBytes(subkey)
		copy(schedule[roundIdx][:], b[2:8])
	}

	return schedule
}

func permuteFromKey(key [8]byte, table []byte) uint64 {
	// Non-standard: interpret first 4 bytes as little-endian u32,
	// second 4 bytes as little-endian u32, then combine as big-endian u64.
	word1 := uint32(key[0]) | uint32(key[1])<<8 | uint32(key[2])<<16 | uint32(key[3])<<24
	word2 := uint32(key[4]) | uint32(key[5])<<8 | uint32(key[6])<<16 | uint32(key[7])<<24
	keyU64 := uint64(word1)<<32 | uint64(word2)

	var output uint64
	for i, pos := range table {
		bit := (keyU64 >> (63 - uint(pos))) & 1
		output |= bit << (uint(len(table)) - 1 - uint(i))
	}
	return output
}

func rotateLeft28(value uint32, amount uint32) uint32 {
	const mask28 uint32 = 0xFFFFFFF0
	return ((value << amount) | (value >> (28 - amount))) & mask28
}

func subkeyToBytes(v uint64) [8]byte {
	var b [8]byte
	b[0] = byte(v >> 56)
	b[1] = byte(v >> 48)
	b[2] = byte(v >> 40)
	b[3] = byte(v >> 32)
	b[4] = byte(v >> 24)
	b[5] = byte(v >> 16)
	b[6] = byte(v >> 8)
	b[7] = byte(v)
	return b
}

// ------- QMC1 Decrypt -------

// privKey is the 128-byte QMC1 stream cipher key for local QRC files.
var privKey = [128]byte{
	0xc3, 0x4a, 0xd6, 0xca, 0x90, 0x67, 0xf7, 0x52, 0xd8, 0xa1, 0x66, 0x62, 0x9f, 0x5b, 0x09, 0x00,
	0xc3, 0x5e, 0x95, 0x23, 0x9f, 0x13, 0x11, 0x7e, 0xd8, 0x92, 0x3f, 0xbc, 0x90, 0xbb, 0x74, 0x0e,
	0xc3, 0x47, 0x74, 0x3d, 0x90, 0xaa, 0x3f, 0x51, 0xd8, 0xf4, 0x11, 0x84, 0x9f, 0xde, 0x95, 0x1d,
	0xc3, 0xc6, 0x09, 0xd5, 0x9f, 0xfa, 0x66, 0xf9, 0xd8, 0xf0, 0xf7, 0xa0, 0x90, 0xa1, 0xd6, 0xf3,
	0xc3, 0xf3, 0xd6, 0xa1, 0x90, 0xa0, 0xf7, 0xf0, 0xd8, 0xf9, 0x66, 0xfa, 0x9f, 0xd5, 0x09, 0xc6,
	0xc3, 0x1d, 0x95, 0xde, 0x9f, 0x84, 0x11, 0xf4, 0xd8, 0x51, 0x3f, 0xaa, 0x90, 0x3d, 0x74, 0x47,
	0xc3, 0x0e, 0x74, 0xbb, 0x90, 0xbc, 0x3f, 0x92, 0xd8, 0x7e, 0x11, 0x13, 0x9f, 0x23, 0x95, 0x5e,
	0xc3, 0x00, 0x09, 0x5b, 0x9f, 0x62, 0x66, 0xa1, 0xd8, 0x52, 0xf7, 0x67, 0x90, 0xca, 0xd6, 0x4a,
}

func qmc1Decrypt(data []byte) {
	const threshold = 0x8000
	if len(data) < threshold {
		for i := range data {
			data[i] ^= privKey[i&0x7F]
		}
		return
	}
	for i := 0; i < threshold; i++ {
		data[i] ^= privKey[i&0x7F]
	}
	for i := threshold; i < len(data); i++ {
		data[i] ^= privKey[(i%0x7FFF)&0x7F]
	}
}
