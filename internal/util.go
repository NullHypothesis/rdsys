package internal

import (
	"encoding/gob"
	"os"
)

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
