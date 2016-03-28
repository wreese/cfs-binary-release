package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gholt/brimtime"
	"github.com/gholt/store"
	"github.com/pandemicsyn/oort/api"
	"github.com/spaolacci/murmur3"
	"golang.org/x/net/context"

	"google.golang.org/grpc"
)

// AccountWS the strurcture carrying all the extra stuff
type AccountWS struct {
	superKey string
	gaddr    string
	gopts    []grpc.DialOption
	gconn    *grpc.ClientConn
	gstore   store.GroupStore
}

// NewAccountWS function used to create a new admin grpc web service
func NewAccountWS(superkey string, gaddr string, insecureSkipVerify bool, grpcOpts ...grpc.DialOption) (*AccountWS, error) {
	// TODO: This all eventually needs to replaced with group rings
	var err error
	a := &AccountWS{
		superKey: superkey,
		gaddr:    gaddr,
		gopts:    grpcOpts,
	}
	//a.gopts = append(a.gopts, grpc.WithTransportCredentials(a.gcreds))
	a.gstore, err = api.NewGroupStore(a.gaddr, 10, a.gopts...)
	if err != nil {
		log.Fatalf("Unable to setup group store: %s", err.Error())
		return nil, err
	}
	log.Println("creating a new group store...")
	return a, nil
}

// grpc Group Store functions
// getGroupClient ...
func (aws *AccountWS) getGClient() {
	var err error
	log.Println("reconnecting to a new group store...")
	aws.gstore, err = api.NewGroupStore(aws.gaddr, 10, aws.gopts...)
	if err != nil {
		log.Fatalf("Unable to setup group store: %s", err.Error())
	}
}

// lookupAccount ...
func (aws *AccountWS) lookupGStore(g string) (string, error) {
	if aws.gconn == nil {
		aws.getGClient()
	}
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
	if aws.gconn == nil {
		aws.getGClient()
	}
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
	if aws.gconn == nil {
		aws.getGClient()
	}
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
