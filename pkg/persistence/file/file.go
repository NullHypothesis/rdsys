package file

import (
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"path"
)

const (
	PersistenceMethod = "file"
)

type FilePersistence struct {
	filename string
}

// Load decodes the content of f.filename and writes the result to the given
// interface.
func (f *FilePersistence) Load(i interface{}) error {
	log.Printf("Attempting to load state from %q.", f.filename)

	fh, err := os.Open(f.filename)
	if err != nil {
		return err
	}
	defer fh.Close()

	dec := gob.NewDecoder(fh)
	return dec.Decode(i)
}

// Save encodes the given interface to f.filename.
func (f *FilePersistence) Save(i interface{}) error {
	log.Printf("Attempting to save state to %q.", f.filename)

	dirPath := path.Dir(f.filename)
	os.MkdirAll(dirPath, 0700)

	fh, err := os.Create(f.filename)
	if err != nil {
		return err
	}
	defer fh.Close()

	enc := gob.NewEncoder(fh)
	return enc.Encode(i)
}

// New returns a new FilePersistence instance.
func New(distName string, workingDir string) *FilePersistence {
	file := fmt.Sprintf("%s-%s.bin", PersistenceMethod, distName)
	filename := path.Join(workingDir, file)
	return &FilePersistence{filename: filename}
}
