package main

import (
	"crypto/tls"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/gholt/brimtime"
	vp "github.com/pandemicsyn/oort/api/valueproto"
	"github.com/spaolacci/murmur3"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type FileService interface {
	GetChunk(id []byte) ([]byte, error)
	WriteChunk(id, data []byte) error
}

var ErrStoreHasNewerValue = errors.New("Error store already has newer value")

type OortFS struct {
	addr               string
	gopts              []grpc.DialOption
	gcreds             credentials.TransportAuthenticator
	insecureSkipVerify bool
	sync.RWMutex
	conn   *grpc.ClientConn
	client vp.ValueStoreClient
}

func NewOortFS(addr string, InsecureSkipVerify bool, grpcOpts ...grpc.DialOption) (*OortFS, error) {
	var err error
	o := &OortFS{
		addr:  addr,
		gopts: grpcOpts,
		gcreds: credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: InsecureSkipVerify,
		}),
		insecureSkipVerify: InsecureSkipVerify,
	}
	o.gopts = append(o.gopts, grpc.WithTransportCredentials(o.gcreds))
	o.conn, err = grpc.Dial(o.addr, o.gopts...)
	if err != nil {
		return &OortFS{}, err
	}
	o.client = vp.NewValueStoreClient(o.conn)
	return o, nil
}

func (o *OortFS) ConnClose() error {
	o.Lock()
	defer o.Unlock()
	return o.conn.Close()
}

func (o *OortFS) ConnState() (grpc.ConnectivityState, error) {
	o.RLock()
	defer o.RUnlock()
	return o.conn.State()
}

func (o *OortFS) GetReadStream(ctx context.Context, opts ...grpc.CallOption) (vp.ValueStore_StreamReadClient, error) {
	o.RLock()
	defer o.RUnlock()
	return o.client.StreamRead(ctx)
}

func (o *OortFS) GetWriteStream(ctx context.Context, opts ...grpc.CallOption) (vp.ValueStore_StreamWriteClient, error) {
	o.RLock()
	defer o.RUnlock()
	return o.client.StreamWrite(ctx)
}

func (o *OortFS) GetChunk(id []byte) ([]byte, error) {
	stream, err := o.GetReadStream(context.Background())
	defer stream.CloseSend()
	if err != nil {
		return []byte(""), err
	}
	r := &vp.ReadRequest{}
	r.KeyA, r.KeyB = murmur3.Sum128(id)
	if err := stream.Send(r); err != nil {
		return []byte(""), err
	}
	res, err := stream.Recv()
	if err == io.EOF {
		return []byte(""), nil
	}
	if err != nil {
		return []byte(""), err
	}
	return res.Value, nil
}

func (o *OortFS) WriteChunk(id, data []byte) error {
	stream, err := o.GetWriteStream(context.Background())
	defer stream.CloseSend()
	if err != nil {
		return err
	}
	w := &vp.WriteRequest{
		Value: data,
	}
	w.KeyA, w.KeyB = murmur3.Sum128(id)
	w.Tsm = brimtime.TimeToUnixMicro(time.Now())
	if err := stream.Send(w); err != nil {
		return err
	}
	res, err := stream.Recv()
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return err
	}
	if res.Tsm > w.Tsm {
		return ErrStoreHasNewerValue
	}
	return nil
}
