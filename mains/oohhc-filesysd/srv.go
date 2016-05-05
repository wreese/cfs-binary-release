package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gholt/brimtime"
	"github.com/gholt/store"
	"github.com/spaolacci/murmur3"
	"golang.org/x/net/context"
)

// FileSystemWS the strurcture carrying all the extra stuff
type FileSystemWS struct {
	gstore store.GroupStore
}

// NewFileSystemWS function used to create a new admin grpc web service
func NewFileSystemWS(store store.GroupStore) (*FileSystemWS, error) {
	fs := &FileSystemWS{
		gstore: store,
	}
	log.Println("creating a new group store...")
	return fs, nil
}

// grpc Group Store functions
// lookupAccount ...
func (fsws *FileSystemWS) readGroupGStore(g string) (string, error) {
	log.Println("Starting a Group Store Lookup")
	keyA, keyB := murmur3.Sum128([]byte(g))
	items, err := fsws.gstore.ReadGroup(context.Background(), keyA, keyB)
	if store.IsNotFound(err) {
		log.Printf("Not Found: %s", g)
		return "", nil
	} else if err != nil {
		return "", err
	}
	// Build a list of accounts.
	log.Println("Build a list of file systems")
	m := make([]string, len(items))
	for k, v := range items {
		m[k] = fmt.Sprintf("%s", v.Value)
	}
	log.Println("Returning a list of file systems")
	return fmt.Sprintf(strings.Join(m, "|")), nil
}

// writeGStore ...
func (fsws *FileSystemWS) writeGStore(g string, m string, p []byte) (string, error) {
	// prepare groupVal and memberVal
	log.Println("Starting a Write to the Group Store")
	keyA, keyB := murmur3.Sum128([]byte(g))
	childKeyA, childKeyB := murmur3.Sum128([]byte(m))
	timestampMicro := brimtime.TimeToUnixMicro(time.Now())
	newTimestampMicro, err := fsws.gstore.Write(context.Background(), keyA, keyB, childKeyA, childKeyB, timestampMicro, p)
	if err != nil {
		return "", err
	}
	log.Println("Successfully wrote something to the Group Store")
	return fmt.Sprintf("TSM: %d", newTimestampMicro), nil
}

// lookupAccount ...
func (fsws *FileSystemWS) getGStore(g string, m string) (string, error) {
	log.Println("Starting a Read from the Group Store")
	keyA, keyB := murmur3.Sum128([]byte(g))
	childKeyA, childKeyB := murmur3.Sum128([]byte(m))
	_, value, err := fsws.gstore.Read(context.Background(), keyA, keyB, childKeyA, childKeyB, nil)
	if store.IsNotFound(err) {
		log.Printf(" Not Found  Key: %d, %d  ChildKey: %d, %d", keyA, keyB, childKeyA, childKeyB)
		return "", nil
	} else if err != nil {
		return "", err
	}
	log.Println("Successfully read an item from the Group Store")
	return fmt.Sprintf("%s", value), nil
}

// lookupAccount ...
func (fsws *FileSystemWS) lookupItems(g string) (string, error) {
	return fmt.Sprintf("%s", "test"), nil
}
