package persist

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
)

var lock sync.Mutex

// Marshal is a function that marshals the object into an
// io.Reader.
func Marshal(v interface{}) (io.Reader, error) {
	b, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(b), nil
}

// Save saves the representation of v to the file at path.
func Save(directory string, file string, v interface{}) error {
	lock.Lock()
	defer lock.Unlock()
	f, err := os.Create(directory + file)
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("failed to create return object file: %s", err.Error()))
	}
	defer f.Close()
	r, err := Marshal(v)
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("failed to marshal return object into io.Reader: %s", err.Error()))
	}
	_, err = io.Copy(f, r)
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("failed to copy return object into file: %s", err.Error()))
	}
	return nil
}

// Load loads the file at path into v.
func Load(path string, v interface{}) error {
	lock.Lock()
	defer lock.Unlock()
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("failed to open return object file: %s", err.Error()))
	}
	defer f.Close()
	if err = json.NewDecoder(f).Decode(&v); err != nil {
		return fmt.Errorf(fmt.Sprintf("failed to decode object file to interface: %s", err.Error()))
	}
	return nil
}

// CreatePath creates the dump files directory
func CreatePath(directory string) error {
	if err := os.Mkdir(directory, 0777); err != nil {
		return fmt.Errorf(fmt.Sprintf("failed to create return object directory: %s", err.Error()))
	}
	return nil
}

// DeletePath deletes the dump files directory
func DeletePath(directory string) error {
	if err := os.RemoveAll(directory); err != nil {
		return fmt.Errorf(fmt.Sprintf("failed to delete return object directory: %s", err.Error()))
	}
	return nil
}
