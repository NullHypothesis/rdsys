package internal

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/gob"
	"os"
)

// GetRandBase32 takes as input the number of desired bytes and returns a
// Base32-encoded string consisting of the given number of cryptographically
// secure random bytes.  If anything went wrong, an error is returned.
func GetRandBase32(numBytes int) (string, error) {

	rawStr := make([]byte, numBytes)
	_, err := rand.Read(rawStr)
	if err != nil {
		return "", err
	}
	str := base32.StdEncoding.EncodeToString(rawStr)

	return str, nil
}

func Serialise(filename string, object interface{}) error {

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	enc := gob.NewEncoder(file)
	enc.Encode(object)

	return nil
}

func Deserialise(filename string, object interface{}) error {

	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	dec := gob.NewDecoder(file)
	return dec.Decode(object)
}
