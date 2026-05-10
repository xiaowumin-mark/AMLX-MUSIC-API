// Package qqmusic implements the MusicProvider interface for QQ Music.
//
// QRC lyrics decryption uses a non-standard, custom DES-like block cipher
// (NOT standard DES). It features:
//   - Custom S-boxes (SBOX2 and SBOX4 contain duplicate values)
//   - Non-standard initial permutation (IP) and inverse IP
//   - 3DES encrypt-decrypt-encrypt pattern with three 8-byte keys
//   - QQ-specific key schedule quirk (D half at position `pos - 27`)
//
// This implementation is derived from the Unilyric project:
//
//	lyrics_helper_rs/src/providers/qq/qrc_codec.rs
//
// Original credit:
//   - Brad Conte's DES implementation
//   - SuJiKiNen's LyricDecoder project (QQ Music adaptation)
//
// The Qimei device fingerprint mechanism generates a virtual Android device
// identity and exchanges it for a Qimei36 token via the Tencent Music API.
package qqmusic
