package config

var ConfKey = []byte{}

// Implement encrypt by user.
var Encode = func(src, key []byte) []byte {
	return src
}

// Implement decrypt by user.
var Decode = func(src, key []byte) ([]byte, error) {
	return src, nil
}
