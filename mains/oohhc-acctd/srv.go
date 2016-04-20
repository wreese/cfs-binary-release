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

// AccountWS the strurcture carrying all the extra stuff
type AccountWS struct {
	superKey string
	gstore   store.GroupStore
}

// NewAccountWS function used to create a new admin grpc web service
func NewAccountWS(key string, store store.GroupStore) (*AccountWS, error) {
	a := &AccountWS{
		superKey: key,
		gstore:   store,
	}
	log.Println("creating a new group store...")
	return a, nil
}

// grpc Group Store functions
// lookupAccount ...
func (aws *AccountWS) lookupGStore(g string) (string, error) {
	log.Println("Starting a Group Store Lookup")
	keyA, keyB := murmur3.Sum128([]byte(g))
	items, err := aws.gstore.LookupGroup(context.Background(), keyA, keyB)
	if store.IsNotFound(err) {
		log.Printf("Not Found: %s", "acct")
		return "", nil
	} else if err != nil {
		return "", err
	}
	// Build a list of accounts.
	log.Println("Build a list of accounts")
	m := make([]string, len(items))
	for k, v := range items {
		_, value, err := aws.gstore.Read(context.Background(), keyA, keyB, v.ChildKeyA, v.ChildKeyB, nil)
		if store.IsNotFound(err) {
			log.Printf("Detail Not Found in group list.  Key: %d, %d  ChildKey: %d, %d", keyA, keyB, v.ChildKeyA, v.ChildKeyB)
			return "", nil
		} else if err != nil {
			return "", err
		}
		m[k] = fmt.Sprintf("%s", value)
	}
	log.Println("Returning a list of accounts")
	return fmt.Sprintf(strings.Join(m, "|")), nil
}

// lookupAccount ...
func (aws *AccountWS) writeGStore(g string, m string, p []byte) (string, error) {
	// prepare groupVal and memberVal
	log.Println("Starting a Write to the Group Store")
	keyA, keyB := murmur3.Sum128([]byte(g))
	childKeyA, childKeyB := murmur3.Sum128([]byte(m))
	timestampMicro := brimtime.TimeToUnixMicro(time.Now())
	newTimestampMicro, err := aws.gstore.Write(context.Background(), keyA, keyB, childKeyA, childKeyB, timestampMicro, p)
	if err != nil {
		return "", err
	}
	log.Println("Successfully wrote something to the Group Store")
	return fmt.Sprintf("TSM: %d", newTimestampMicro), nil
}

// lookupAccount ...
func (aws *AccountWS) getGStore(g string, m string) (string, error) {
	// TODO:
	log.Println("Starting a Read from the Group Store")
	keyA, keyB := murmur3.Sum128([]byte(g))
	childKeyA, childKeyB := murmur3.Sum128([]byte(m))
	_, value, err := aws.gstore.Read(context.Background(), keyA, keyB, childKeyA, childKeyB, nil)
	if store.IsNotFound(err) {
		log.Printf("Detail Not Found in group list.  Key: %d, %d  ChildKey: %d, %d", keyA, keyB, childKeyA, childKeyB)
		return "", nil
	} else if err != nil {
		return "", err
	}
	log.Println("Successfully read an item from the Group Store")
	return fmt.Sprintf("%s", value), nil
}
