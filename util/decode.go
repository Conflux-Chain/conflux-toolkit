package util

import "math/big"

// MustDecodeUint256 decodes data as uint256.
func MustDecodeUint256(encoded string) *big.Int {
	value, ok := new(big.Int).SetString(encoded[2:], 16)
	if !ok {
		panic("failed to decode uint256, data = " + encoded)
	}

	return value
}

// DecodeAddress decodes data as address.
func DecodeAddress(encoded string) string {
	return "0x" + encoded[26:]
}
