package main

import (
	"errors"
	"hash"
	"hash/crc32"
	"log"
	"os"
	"sort"
	"sync"
	"time"

	"bazil.org/fuse"

	pb "github.com/creiht/formic/proto"
	"github.com/gholt/brimtime"
	"github.com/gholt/store"
	"github.com/gogo/protobuf/proto"
	"github.com/pandemicsyn/oort/api"
	"github.com/spaolacci/murmur3"
	"google.golang.org/grpc"
)

const (
	InodeEntryVersion = 1
	DirEntryVersion   = 1
	FileBlockVersion  = 1
)

type FileService interface {
	GetAttr(id []byte) (*pb.Attr, error)
	SetAttr(id []byte, attr *pb.Attr, valid uint32) (*pb.Attr, error)
	Create(parent, id []byte, inode uint64, name string, attr *pb.Attr, isdir bool) (string, *pb.Attr, error)
	Update(id []byte, block, size, blocksize uint64, mtime int64) error
	Lookup(parent []byte, name string) (string, *pb.Attr, error)
	ReadDirAll(id []byte) (*pb.ReadDirAllResponse, error)
	Remove(parent []byte, name string) (int32, error)
	Symlink(parent, id []byte, name string, target string, attr *pb.Attr, inode uint64) (*pb.SymlinkResponse, error)
	Readlink(id []byte) (*pb.ReadlinkResponse, error)
	Getxattr(id []byte, name string) (*pb.GetxattrResponse, error)
	Setxattr(id []byte, name string, value []byte) (*pb.SetxattrResponse, error)
	Listxattr(id []byte) (*pb.ListxattrResponse, error)
	Removexattr(id []byte, name string) (*pb.RemovexattrResponse, error)
	Rename(oldParent, newParent []byte, oldName, newName string) (*pb.RenameResponse, error)
	GetChunk(id []byte) ([]byte, error)
	WriteChunk(id, data []byte) error
}

var ErrStoreHasNewerValue = errors.New("Error store already has newer value")

type OortFS struct {
	vaddr  string
	gaddr  string
	vstore store.ValueStore
	gstore store.GroupStore
	hasher func() hash.Hash32
	sync.RWMutex
}

func NewOortFS(vaddr, gaddr string, grpcOpts ...grpc.DialOption) (*OortFS, error) {
	// TODO: This all eventually needs to replaced with value and group rings
	var err error
	o := &OortFS{
		vaddr:  vaddr,
		gaddr:  gaddr,
		hasher: crc32.NewIEEE,
	}
	// TODO: These 10s here are the number of grpc streams the api can make per
	// request type; should likely be configurable somewhere along the line,
	// but hardcoded for now.
	o.vstore, err = api.NewValueStore(vaddr, 10, grpcOpts...)
	if err != nil {
		return &OortFS{}, err
	}
	o.gstore, err = api.NewGroupStore(gaddr, 10, grpcOpts...)
	if err != nil {
		return &OortFS{}, err
	}
	// TODO: This should be setup out of band when an FS is first created
	// NOTE: This also means that it is only single user until we init filesystems out of band
	// Init the root node
	id := GetID(1, 1, 1, 0)
	n, err := o.GetChunk(id)
	if len(n) == 0 {
		log.Println("Root node not found, creating new root")
		// Need to create the root node
		r := &pb.InodeEntry{
			Version: InodeEntryVersion,
			Inode:   1,
			IsDir:   true,
		}
		ts := time.Now().Unix()
		r.Attr = &pb.Attr{
			Inode:  1,
			Atime:  ts,
			Mtime:  ts,
			Ctime:  ts,
			Crtime: ts,
			Mode:   uint32(os.ModeDir | 0775),
			Uid:    1001, // TODO: need to config default user/group id
			Gid:    1001,
		}
		b, err := proto.Marshal(r)
		if err != nil {
			return &OortFS{}, err
		}
		err = o.WriteChunk(id, b)
		if err != nil {
			return &OortFS{}, err
		}
	}
	return o, nil
}

// Helper methods to get data from value and group store
func (o *OortFS) readValue(id []byte) ([]byte, error) {
	// TODO: You might want to make this whole area pass in reusable []byte to
	// lessen gc pressure.
	keyA, keyB := murmur3.Sum128(id)
	_, v, err := o.vstore.Read(keyA, keyB, nil)
	return v, err
}

func (o *OortFS) writeValue(id, data []byte) error {
	keyA, keyB := murmur3.Sum128(id)
	timestampMicro := brimtime.TimeToUnixMicro(time.Now())
	oldTimestampMicro, err := o.vstore.Write(keyA, keyB, timestampMicro, data)
	if err != nil {
		return err
	}
	if oldTimestampMicro >= timestampMicro {
		return ErrStoreHasNewerValue
	}
	return nil
}

func (o *OortFS) deleteValue(id []byte) error {
	keyA, keyB := murmur3.Sum128(id)
	timestampMicro := brimtime.TimeToUnixMicro(time.Now())
	oldTimestampMicro, err := o.vstore.Delete(keyA, keyB, timestampMicro)
	if oldTimestampMicro >= timestampMicro {
		return ErrStoreHasNewerValue
	}
	return err
}

func (o *OortFS) writeGroup(key, childKey, value []byte) error {
	keyA, keyB := murmur3.Sum128(key)
	childKeyA, childKeyB := murmur3.Sum128(childKey)
	timestampMicro := brimtime.TimeToUnixMicro(time.Now())
	oldTimestampMicro, err := o.gstore.Write(keyA, keyB, childKeyA, childKeyB, timestampMicro, value)
	if err != nil {
		return nil
	}
	if oldTimestampMicro >= timestampMicro {
		return ErrStoreHasNewerValue
	}
	return nil
}

func (o *OortFS) readGroupItem(key, childKey []byte) ([]byte, error) {
	childKeyA, childKeyB := murmur3.Sum128(childKey)
	return o.readGroupItemByKey(key, childKeyA, childKeyB)
}

func (o *OortFS) readGroupItemByKey(key []byte, childKeyA, childKeyB uint64) ([]byte, error) {
	keyA, keyB := murmur3.Sum128(key)
	_, v, err := o.gstore.Read(keyA, keyB, childKeyA, childKeyB, nil)
	return v, err
}

func (o *OortFS) deleteGroupItem(key, childKey []byte) error {
	keyA, keyB := murmur3.Sum128(key)
	childKeyA, childKeyB := murmur3.Sum128(childKey)
	timestampMicro := brimtime.TimeToUnixMicro(time.Now())
	oldTimestampMicro, err := o.gstore.Delete(keyA, keyB, childKeyA, childKeyB, timestampMicro)
	if err != nil {
		return err
	}
	if oldTimestampMicro >= timestampMicro {
		return ErrStoreHasNewerValue
	}
	return nil
	/*
		r := &gp.DeleteRequest{}
		r.KeyA, r.KeyB = murmur3.Sum128(key)
		r.ChildKeyA, r.ChildKeyB = murmur3.Sum128(childKey)
		r.TimestampMicro = brimtime.TimeToUnixMicro(time.Now())
		ctx, _ := context.WithTimeout(context.Background(), time.Second*10)
		_, err := o.gclient.Delete(ctx, r)
		return err
	*/
}

// FileService methods
func (o *OortFS) GetChunk(id []byte) ([]byte, error) {
	b, err := o.readValue(id)
	if err != nil {
		return nil, err
	}
	fb := &pb.FileBlock{}
	err = proto.Unmarshal(b, fb)
	if err != nil {
		return nil, err
	}
	// TODO: Validate checksum and handle errors
	return fb.Data, nil
}

func (o *OortFS) WriteChunk(id, data []byte) error {
	crc := o.hasher()
	crc.Write(data)
	fb := &pb.FileBlock{
		Version:  FileBlockVersion,
		Data:     data,
		Checksum: crc.Sum32(),
	}
	b, err := proto.Marshal(fb)
	if err != nil {
		return err
	}
	return o.writeValue(id, b)
}

func (o *OortFS) GetAttr(id []byte) (*pb.Attr, error) {
	b, err := o.GetChunk(id)
	if err != nil {
		return &pb.Attr{}, err
	}
	n := &pb.InodeEntry{}
	err = proto.Unmarshal(b, n)
	if err != nil {
		return &pb.Attr{}, err
	}
	return n.Attr, nil
}

func (o *OortFS) SetAttr(id []byte, attr *pb.Attr, v uint32) (*pb.Attr, error) {
	valid := fuse.SetattrValid(v)
	b, err := o.GetChunk(id)
	if err != nil {
		return &pb.Attr{}, err
	}
	n := &pb.InodeEntry{}
	err = proto.Unmarshal(b, n)
	if err != nil {
		return &pb.Attr{}, err
	}
	if valid.Mode() {
		n.Attr.Mode = attr.Mode
	}
	if valid.Size() {
		if n.Attr.Size == 0 {
			n.Blocks = 0
			n.LastBlock = 0
		}
		n.Attr.Size = attr.Size
	}
	if valid.Mtime() {
		n.Attr.Mtime = attr.Mtime
	}
	if valid.Atime() {
		n.Attr.Atime = attr.Atime
	}
	if valid.Uid() {
		n.Attr.Uid = attr.Uid
	}
	if valid.Gid() {
		n.Attr.Gid = attr.Gid
	}
	b, err = proto.Marshal(n)
	if err != nil {
		return &pb.Attr{}, err
	}
	err = o.WriteChunk(id, b)
	if err != nil {
		return &pb.Attr{}, err
	}

	return n.Attr, nil
}

func (o *OortFS) Create(parent, id []byte, inode uint64, name string, attr *pb.Attr, isdir bool) (string, *pb.Attr, error) {
	// Check to see if the name already exists
	val, err := o.readGroupItem(parent, []byte(name))
	if err != nil && !store.IsNotFound(err) {
		// TODO: Needs beter error handling
		return "", &pb.Attr{}, err
	}
	if len(val) > 0 {
		return "", &pb.Attr{}, nil
	}
	// Add the name to the group
	d := &pb.DirEntry{
		Version: DirEntryVersion,
		Name:    name,
		Id:      id,
	}
	b, err := proto.Marshal(d)
	if err != nil {
		return "", &pb.Attr{}, err
	}
	err = o.writeGroup(parent, []byte(name), b)
	if err != nil {
		return "", &pb.Attr{}, err
	}
	// Add the inode entry
	n := &pb.InodeEntry{
		Version: InodeEntryVersion,
		Inode:   inode,
		IsDir:   isdir,
		Attr:    attr,
		Blocks:  0,
	}
	b, err = proto.Marshal(n)
	if err != nil {
		return "", &pb.Attr{}, err
	}
	err = o.WriteChunk(id, b)
	if err != nil {
		return "", &pb.Attr{}, err
	}
	return name, attr, nil
}

func (o *OortFS) Lookup(parent []byte, name string) (string, *pb.Attr, error) {
	// Get the id
	b, err := o.readGroupItem(parent, []byte(name))
	if store.IsNotFound(err) {
		return "", &pb.Attr{}, nil
	} else if err != nil {
		return "", &pb.Attr{}, err
	}
	d := &pb.DirEntry{}
	err = proto.Unmarshal(b, d)
	if err != nil {
		return "", &pb.Attr{}, err
	}
	// Get the Inode entry
	b, err = o.GetChunk(d.Id)
	if err != nil {
		return "", &pb.Attr{}, err
	}
	n := &pb.InodeEntry{}
	err = proto.Unmarshal(b, n)
	if err != nil {
		return "", &pb.Attr{}, err
	}
	return d.Name, n.Attr, nil
}

// Needed to be able to sort the dirents
type ByDirent []*pb.DirEnt

func (d ByDirent) Len() int {
	return len(d)
}

func (d ByDirent) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}

func (d ByDirent) Less(i, j int) bool {
	return d[i].Name < d[j].Name
}

func (o *OortFS) ReadDirAll(id []byte) (*pb.ReadDirAllResponse, error) {
	keyA, keyB := murmur3.Sum128(id)
	// Get the keys from the group
	items, err := o.gstore.LookupGroup(keyA, keyB)
	log.Println("ITEMS: ", items)
	if err != nil {
		// TODO: Needs beter error handling
		log.Println("Error looking up group: ", err)
		return &pb.ReadDirAllResponse{}, err
	}
	// Iterate over each item, getting the ID then the Inode Entry
	e := &pb.ReadDirAllResponse{}
	dirent := &pb.DirEntry{}
	for _, item := range items {
		// lookup the item in the group to get the id
		_, b, err := o.gstore.Read(keyA, keyB, item.ChildKeyA, item.ChildKeyB, nil)
		if err != nil {
			// TODO: Needs beter error handling
			log.Println("Error with lookup: ", err)
			continue
		}
		err = proto.Unmarshal(b, dirent)
		if err != nil {
			return &pb.ReadDirAllResponse{}, err
		}
		// get the inode entry
		b, err = o.GetChunk(dirent.Id)
		if err != nil {
			continue
		}
		n := &pb.InodeEntry{}
		err = proto.Unmarshal(b, n)
		if err != nil {
			continue
		}
		if n.IsDir {
			e.DirEntries = append(e.DirEntries, &pb.DirEnt{Name: dirent.Name, Attr: n.Attr})
		} else {
			e.FileEntries = append(e.FileEntries, &pb.DirEnt{Name: dirent.Name, Attr: n.Attr})
		}
	}
	sort.Sort(ByDirent(e.DirEntries))
	sort.Sort(ByDirent(e.FileEntries))
	return e, nil
}

func (o *OortFS) Remove(parent []byte, name string) (int32, error) {
	// Get the ID from the group list
	b, err := o.readGroupItem(parent, []byte(name))
	if store.IsNotFound(err) {
		return 1, nil
	} else if err != nil {
		return 1, err
	}
	d := &pb.DirEntry{}
	err = proto.Unmarshal(b, d)
	if err != nil {
		return 1, err
	}
	// Remove the inode
	// TODO: Need to delete more than just the inode
	err = o.deleteValue(d.Id)
	if err != nil {
		return 1, err
	}
	// TODO: More error handling needed
	// Remove from the group
	err = o.deleteGroupItem(parent, []byte(name))
	if err != nil {
		return 1, err // Not really sure what should be done here to try to recover from err
	}
	return 0, nil
}

func (o *OortFS) Update(id []byte, block, blocksize, size uint64, mtime int64) error {
	b, err := o.GetChunk(id)
	if err != nil {
		return err
	}
	n := &pb.InodeEntry{}
	err = proto.Unmarshal(b, n)
	if err != nil {
		return err
	}
	blocks := n.Blocks
	if block >= blocks {
		n.Blocks = block + 1
		n.LastBlock = size
		n.BlockSize = blocksize
		n.Attr.Size = blocksize*block + size
	} else if block == (blocks - 1) {
		n.LastBlock = size
		n.Attr.Size = blocksize*block + size
	}

	n.Attr.Mtime = mtime
	b, err = proto.Marshal(n)
	if err != nil {
		return err
	}
	err = o.WriteChunk(id, b)
	if err != nil {
		return err
	}
	return nil
}

func (o *OortFS) Symlink(parent, id []byte, name string, target string, attr *pb.Attr, inode uint64) (*pb.SymlinkResponse, error) {
	// Check to see if the name exists
	val, err := o.readGroupItem(parent, []byte(name))
	if err != nil && !store.IsNotFound(err) {
		// TODO: Needs beter error handling
		return &pb.SymlinkResponse{}, err
	}
	if len(val) > 1 { // Exists already
		return &pb.SymlinkResponse{}, nil
	}
	n := &pb.InodeEntry{
		Version: InodeEntryVersion,
		Inode:   inode,
		IsDir:   false,
		IsLink:  true,
		Target:  target,
		Attr:    attr,
	}
	b, err := proto.Marshal(n)
	if err != nil {
		return &pb.SymlinkResponse{}, err
	}
	err = o.WriteChunk(id, b)
	if err != nil {
		return &pb.SymlinkResponse{}, err
	}
	// Add the name to the group
	d := &pb.DirEntry{
		Version: DirEntryVersion,
		Name:    name,
		Id:      id,
	}
	b, err = proto.Marshal(d)
	if err != nil {
		return &pb.SymlinkResponse{}, err
	}
	err = o.writeGroup(parent, []byte(name), b)
	if err != nil {
		return &pb.SymlinkResponse{}, err
	}
	return &pb.SymlinkResponse{Name: name, Attr: attr}, nil
}

func (o *OortFS) Readlink(id []byte) (*pb.ReadlinkResponse, error) {
	b, err := o.GetChunk(id)
	if err != nil {
		return &pb.ReadlinkResponse{}, err
	}
	n := &pb.InodeEntry{}
	err = proto.Unmarshal(b, n)
	if err != nil {
		return &pb.ReadlinkResponse{}, err
	}
	return &pb.ReadlinkResponse{Target: n.Target}, nil
}

func (o *OortFS) Getxattr(id []byte, name string) (*pb.GetxattrResponse, error) {
	b, err := o.GetChunk(id)
	if err != nil {
		return &pb.GetxattrResponse{}, err
	}
	n := &pb.InodeEntry{}
	err = proto.Unmarshal(b, n)
	if err != nil {
		return &pb.GetxattrResponse{}, err
	}
	if xattr, ok := n.Xattr[name]; ok {
		return &pb.GetxattrResponse{Xattr: xattr}, nil
	}
	return &pb.GetxattrResponse{}, nil
}

func (o *OortFS) Setxattr(id []byte, name string, value []byte) (*pb.SetxattrResponse, error) {
	b, err := o.GetChunk(id)
	if err != nil {
		return &pb.SetxattrResponse{}, err
	}
	n := &pb.InodeEntry{}
	err = proto.Unmarshal(b, n)
	if err != nil {
		return &pb.SetxattrResponse{}, err
	}
	n.Xattr[name] = value
	b, err = proto.Marshal(n)
	if err != nil {
		return &pb.SetxattrResponse{}, err
	}
	err = o.WriteChunk(id, b)
	if err != nil {
		return &pb.SetxattrResponse{}, err
	}
	return &pb.SetxattrResponse{}, nil
}

func (o *OortFS) Listxattr(id []byte) (*pb.ListxattrResponse, error) {
	resp := &pb.ListxattrResponse{}
	b, err := o.GetChunk(id)
	if err != nil {
		return &pb.ListxattrResponse{}, err
	}
	n := &pb.InodeEntry{}
	err = proto.Unmarshal(b, n)
	if err != nil {
		return &pb.ListxattrResponse{}, err
	}
	names := ""
	for name := range n.Xattr {
		names += name
		names += "\x00"
	}
	resp.Xattr = []byte(names)
	return resp, nil
}

func (o *OortFS) Removexattr(id []byte, name string) (*pb.RemovexattrResponse, error) {
	b, err := o.GetChunk(id)
	if err != nil {
		return &pb.RemovexattrResponse{}, err
	}
	n := &pb.InodeEntry{}
	err = proto.Unmarshal(b, n)
	if err != nil {
		return &pb.RemovexattrResponse{}, err
	}
	delete(n.Xattr, name)
	b, err = proto.Marshal(n)
	if err != nil {
		return &pb.RemovexattrResponse{}, err
	}
	err = o.WriteChunk(id, b)
	if err != nil {
		return &pb.RemovexattrResponse{}, err
	}
	return &pb.RemovexattrResponse{}, nil
}

func (o *OortFS) Rename(oldParent, newParent []byte, oldName, newName string) (*pb.RenameResponse, error) {
	// Check if the new name already exists
	id, err := o.readGroupItem(newParent, []byte(newName))
	if err != nil && !store.IsNotFound(err) {
		// TODO: Needs beter error handling
		return &pb.RenameResponse{}, err
	}
	if len(id) > 0 { // New name already exists
		return &pb.RenameResponse{}, nil
	}
	// Get the ID from the group list
	b, err := o.readGroupItem(oldParent, []byte(oldName))
	if store.IsNotFound(err) {
		return &pb.RenameResponse{}, nil
	}
	if err != nil {
		return &pb.RenameResponse{}, err
	}
	d := &pb.DirEntry{}
	err = proto.Unmarshal(b, d)
	if err != nil {
		return &pb.RenameResponse{}, err
	}
	// Delete old entry
	err = o.deleteGroupItem(oldParent, []byte(oldName))
	if err != nil {
		return &pb.RenameResponse{}, err
	}
	// Create new entry
	d.Name = newName
	b, err = proto.Marshal(d)
	err = o.writeGroup(newParent, []byte(newName), b)
	if err != nil {
		return &pb.RenameResponse{}, err
	}
	return &pb.RenameResponse{}, nil
}
