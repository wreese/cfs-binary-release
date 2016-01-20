package main

import (
	"bytes"
	"fmt"
	"testing"

	"golang.org/x/net/context"

	pb "github.com/creiht/formic/proto"
)

// Minimal DirService for testing
type TestDS struct {
}

func NewTestDS() *TestDS {
	return &TestDS{}
}

func (ds *TestDS) GetAttr(inode uint64) (*pb.Attr, error) {
	return &pb.Attr{}, nil
}

func (ds *TestDS) SetAttr(inode uint64, attr *pb.Attr, valid uint32) (*pb.Attr, error) {
	return &pb.Attr{}, nil
}

func (ds *TestDS) Create(parent, inode uint64, name string, attr *pb.Attr, isdir bool) (string, *pb.Attr, error) {
	return name, attr, nil
}

func (ds *TestDS) Lookup(parent uint64, name string) (string, *pb.Attr, error) {
	return "", &pb.Attr{}, nil
}

func (ds *TestDS) ReadDirAll(inode uint64) (*pb.ReadDirAllResponse, error) {
	return &pb.ReadDirAllResponse{}, nil
}

func (ds *TestDS) Remove(parent uint64, name string) (int32, error) {
	return 1, nil
}

func (ds *TestDS) Update(inode, block, blocksize, size uint64, mtime int64) {
}

func (ds *TestDS) Symlink(parent uint64, name string, target string, attr *pb.Attr, inode uint64) (*pb.SymlinkResponse, error) {
	return &pb.SymlinkResponse{}, nil
}

func (ds *TestDS) Readlink(inode uint64) (*pb.ReadlinkResponse, error) {
	return &pb.ReadlinkResponse{}, nil
}

func (ds *TestDS) Getxattr(r *pb.GetxattrRequest) (*pb.GetxattrResponse, error) {
	return &pb.GetxattrResponse{}, nil
}

func (ds *TestDS) Setxattr(r *pb.SetxattrRequest) (*pb.SetxattrResponse, error) {
	return &pb.SetxattrResponse{}, nil
}

func (ds *TestDS) Listxattr(r *pb.ListxattrRequest) (*pb.ListxattrResponse, error) {
	return &pb.ListxattrResponse{}, nil
}

func (ds *TestDS) Removexattr(r *pb.RemovexattrRequest) (*pb.RemovexattrResponse, error) {
	return &pb.RemovexattrResponse{}, nil
}

func (ds *TestDS) Rename(r *pb.RenameRequest) (*pb.RenameResponse, error) {
	return &pb.RenameResponse{}, nil
}

// Minimal FileService for testing
type TestFS struct {
	writes [][]byte
	reads  [][]byte
}

func NewTestFS() *TestFS {
	return &TestFS{
		writes: make([][]byte, 0),
		reads:  make([][]byte, 0),
	}
}

func (fs *TestFS) GetChunk(id []byte) ([]byte, error) {
	if len(fs.reads) > 0 {
		chunk := fs.reads[0]
		fs.reads = fs.reads[1:]
		return chunk, nil
	} else {
		return []byte(""), nil
	}
}

func (fs *TestFS) WriteChunk(id, data []byte) error {
	fs.writes = append(fs.writes, data)
	return nil
}

func (fs *TestFS) clearwrites() {
	fs.writes = make([][]byte, 0)
}

func (fs *TestFS) addread(d []byte) {
	fs.reads = append(fs.reads, d)
}

func TestGetID(t *testing.T) {
	api := NewApiServer(NewTestDS(), NewTestFS())
	id1 := api.GetID(uint64(11), uint64(1), uint64(1), uint64(1))
	id2 := api.GetID(uint64(11), uint64(1), uint64(1), uint64(1))
	if !bytes.Equal(id1, id2) {
		t.Errorf("Generated IDs not equal")
	}
	id3 := api.GetID(uint64(11), uint64(1), uint64(1), uint64(2))
	if bytes.Equal(id1, id3) {
		t.Errorf("Generated IDs were equal")
	}
}

func TestCreate(t *testing.T) {
	api := NewApiServer(NewTestDS(), NewTestFS())
	_, err := api.Create(context.Background(), &pb.CreateRequest{Parent: 1, Name: "Test", Attr: &pb.Attr{Gid: 1001, Uid: 1001}})
	if err != nil {
		t.Error("Create Failed: ", err)
	}

}

func TestWrite_Basic(t *testing.T) {
	fs := NewTestFS()
	api := NewApiServer(NewTestDS(), fs)
	api.blocksize = 10
	chunk := pb.WriteRequest{
		Inode:   0,
		Offset:  0,
		Payload: []byte("1234567890"),
	}
	r, err := api.Write(context.Background(), &chunk)
	if err != nil {
		t.Error("Write Failed: ", err)
	}
	if r.Status != 0 {
		t.Error("Write status expected: 0, received: ", r.Status)
	}
	if !bytes.Equal(chunk.Payload, fs.writes[0]) {
		fmt.Println(fs.writes)
		t.Errorf("Expected write: '%s' recieved: '%s'", chunk.Payload, fs.writes[0])
	}
	chunk.Payload = []byte("1")
	fs.clearwrites()
	r, err = api.Write(context.Background(), &chunk)
	if err != nil {
		t.Error("Write Failed: ", err)
	}
	if r.Status != 0 {
		t.Error("Write status expected: 0, received: ", r.Status)
	}
	if !bytes.Equal(chunk.Payload, fs.writes[0]) {
		fmt.Println(fs.writes)
		t.Errorf("Expected write: '%s' recieved: '%s'", chunk.Payload, fs.writes[0])
	}

}

func TestWrite_Chunk(t *testing.T) {
	fs := NewTestFS()
	api := NewApiServer(NewTestDS(), fs)
	api.blocksize = 5
	chunk := pb.WriteRequest{
		Inode:   0,
		Offset:  0,
		Payload: []byte("1234567890"),
	}
	r, err := api.Write(context.Background(), &chunk)
	if err != nil {
		t.Error("Write Failed: ", err)
	}
	if r.Status != 0 {
		t.Error("Write status expected: 0, received: ", r.Status)
	}
	if !bytes.Equal(chunk.Payload[:5], fs.writes[0]) {
		fmt.Println(fs.writes)
		t.Errorf("Expected write: '%s' recieved: '%s'", chunk.Payload[:5], fs.writes[0])
	}
	if !bytes.Equal(chunk.Payload[5:], fs.writes[1]) {
		fmt.Println(fs.writes)
		t.Errorf("Expected write: '%s' recieved: '%s'", chunk.Payload[5:], fs.writes[1])
	}
}

func TestWrite_Offset(t *testing.T) {
	fs := NewTestFS()
	api := NewApiServer(NewTestDS(), fs)
	api.blocksize = 10
	chunk := pb.WriteRequest{
		Offset:  5,
		Payload: []byte("12345"),
		Inode:   0,
	}
	r, err := api.Write(context.Background(), &chunk)
	if err != nil {
		t.Error("Write Failed: ", err)
	}
	if r.Status != 0 {
		t.Error("Write status expected: 0, received: ", r.Status)
	}
	if !bytes.Equal(chunk.Payload, fs.writes[0][5:]) {
		fmt.Println(fs.writes)
		t.Errorf("Expected write: '%s' recieved: '%s'", chunk.Payload, fs.writes[0][5:])
	}
	fs.clearwrites()
	chunk = pb.WriteRequest{
		Offset:  2,
		Payload: []byte("12345"),
		Inode:   0,
	}
	r, err = api.Write(context.Background(), &chunk)
	if err != nil {
		t.Error("Write Failed: ", err)
	}
	if r.Status != 0 {
		t.Error("Write status expected: 0, received: ", r.Status)
	}
	if !bytes.Equal(chunk.Payload, fs.writes[0][2:7]) {
		fmt.Println(fs.writes)
		t.Errorf("Expected write: '%s' recieved: '%s'", chunk.Payload, fs.writes[0][5:])
	}

}

func TestWrite_MultiOffset(t *testing.T) {
	fs := NewTestFS()
	api := NewApiServer(NewTestDS(), fs)
	api.blocksize = 20
	chunk := pb.WriteRequest{
		Offset:  5,
		Payload: []byte("12345"),
		Inode:   0,
	}
	r, err := api.Write(context.Background(), &chunk)
	if err != nil {
		t.Error("Write Failed: ", err)
	}
	if r.Status != 0 {
		t.Error("Write status expected: 0, received: ", r.Status)
	}
	if !bytes.Equal(chunk.Payload, fs.writes[0][5:]) {
		fmt.Println(fs.writes)
		t.Errorf("Expected write: '%s' recieved: '%s'", chunk.Payload, fs.writes[0][5:])
	}
}

func TestRead_Basic(t *testing.T) {
	fs := NewTestFS()
	api := NewApiServer(NewTestDS(), fs)
	api.blocksize = 10
	write := []byte("0123456789")
	fs.addread(write)
	data, err := api.Read(context.Background(), &pb.ReadRequest{Inode: 0, Offset: 0, Size: 10})
	if err != nil {
		t.Error("Read Failed: ", err)
	}
	if !bytes.Equal(data.Payload, write) {
		t.Errorf("Expected read: '%s' received: '%s'", write, data)
	}
}

func TestRead_Offset(t *testing.T) {
	fs := NewTestFS()
	api := NewApiServer(NewTestDS(), fs)
	api.blocksize = 10
	write := []byte("0123456789")
	fs.addread(write)
	data, err := api.Read(context.Background(), &pb.ReadRequest{Inode: 0, Offset: 5, Size: 5})
	if err != nil {
		t.Error("Read Failed: ", err)
	}
	if !bytes.Equal(data.Payload, write[5:]) {
		t.Errorf("Expected read: '%s' received: '%s'", write[5:], data)
	}
}

func TestRead_Chunk(t *testing.T) {
	fs := NewTestFS()
	api := NewApiServer(NewTestDS(), fs)
	api.blocksize = 10
	write1 := []byte("0123456789")
	write2 := []byte("9876543210")
	fs.addread(write1)
	fs.addread(write2)
	data, err := api.Read(context.Background(), &pb.ReadRequest{Inode: 0, Offset: 0, Size: 20})
	if err != nil {
		t.Error("Read Failed: ", err)
	}
	if !bytes.Equal(data.Payload[:10], write1) {
		t.Errorf("Expected read: '%s' received: '%s'", write1, data.Payload[:10])
	}
	if !bytes.Equal(data.Payload[10:], write2) {
		t.Errorf("Expected read: '%s' received: '%s'", write2, data.Payload[10:])
	}
}
