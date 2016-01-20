package main

import (
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"bazil.org/fuse"

	pb "github.com/creiht/formic/proto"
)

type DirService interface {
	GetAttr(inode uint64) (*pb.Attr, error)
	SetAttr(inode uint64, attr *pb.Attr, valid uint32) (*pb.Attr, error)
	Create(parent, inode uint64, name string, attr *pb.Attr, isdir bool) (string, *pb.Attr, error)
	Update(inode, block, size, blocksize uint64, mtime int64)
	Lookup(parent uint64, name string) (string, *pb.Attr, error)
	ReadDirAll(inode uint64) (*pb.ReadDirAllResponse, error)
	Remove(parent uint64, name string) (int32, error)
	Symlink(parent uint64, name string, target string, attr *pb.Attr, inode uint64) (*pb.SymlinkResponse, error)
	Readlink(inode uint64) (*pb.ReadlinkResponse, error)
	Getxattr(*pb.GetxattrRequest) (*pb.GetxattrResponse, error)
	Setxattr(*pb.SetxattrRequest) (*pb.SetxattrResponse, error)
	Listxattr(*pb.ListxattrRequest) (*pb.ListxattrResponse, error)
	Removexattr(*pb.RemovexattrRequest) (*pb.RemovexattrResponse, error)
	Rename(*pb.RenameRequest) (*pb.RenameResponse, error)
}

// Entry describes each node in our fs.
// it also contains a list of all other entries "in this node".
// i.e. all files/directory in this directory.
type Entry struct {
	path  string // string path/name for this entry
	isdir bool
	sync.RWMutex
	attr      *pb.Attr
	parent    uint64            // inode of the parent
	inode     uint64            //the original/actual inode incase fuse stomps on the one in attr
	entries   map[string]uint64 // subdir/files by name
	ientries  map[uint64]string // subdir/files by inode
	nodeCount uint64            // uint64
	islink    bool
	target    string
	xattrs    map[string][]byte
	blocks    uint64 // For files, the number of blocks this file represents
	blocksize uint64 // The size of each block
	lastblock uint64 // The size of the last block
}

// In memory implementation of DirService
type InMemDS struct {
	sync.RWMutex
	nodes map[uint64]*Entry
}

func NewInMemDS() *InMemDS {
	ds := &InMemDS{
		nodes: make(map[uint64]*Entry),
	}
	n := &Entry{
		path:     "/",
		inode:    1,
		isdir:    true,
		entries:  make(map[string]uint64),
		ientries: make(map[uint64]string),
	}
	ts := time.Now().Unix()
	n.attr = &pb.Attr{
		Inode:  n.inode,
		Atime:  ts,
		Mtime:  ts,
		Ctime:  ts,
		Crtime: ts,
		Mode:   uint32(os.ModeDir | 0775),
		Uid:    1001, // TODO: need to config default user/group id
		Gid:    1001,
	}
	ds.nodes[n.attr.Inode] = n
	return ds
}

func (ds *InMemDS) GetAttr(inode uint64) (*pb.Attr, error) {
	ds.RLock()
	defer ds.RUnlock()
	if entry, ok := ds.nodes[inode]; ok {
		return entry.attr, nil
	}
	return &pb.Attr{}, nil
}

func (ds *InMemDS) SetAttr(inode uint64, attr *pb.Attr, v uint32) (*pb.Attr, error) {
	ds.Lock()
	defer ds.Unlock()
	valid := fuse.SetattrValid(v)
	if entry, ok := ds.nodes[inode]; ok {
		if valid.Mode() {
			entry.attr.Mode = attr.Mode
		}
		if valid.Size() {
			if attr.Size == 0 {
				entry.blocks = 0
				entry.lastblock = 0
			}
			entry.attr.Size = attr.Size
		}
		if valid.Mtime() {
			entry.attr.Mtime = attr.Mtime
		}
		if valid.Atime() {
			entry.attr.Atime = attr.Atime
		}
		if valid.Uid() {
			entry.attr.Uid = attr.Uid
		}
		if valid.Gid() {
			entry.attr.Gid = attr.Gid
		}
		return entry.attr, nil
	}
	return &pb.Attr{}, nil
}

func (ds *InMemDS) Create(parent, inode uint64, name string, attr *pb.Attr, isdir bool) (string, *pb.Attr, error) {
	ds.Lock()
	defer ds.Unlock()
	if _, exists := ds.nodes[parent].entries[name]; exists {
		return "", &pb.Attr{}, nil
	}
	entry := &Entry{
		path:   name,
		inode:  inode,
		isdir:  isdir,
		attr:   attr,
		xattrs: make(map[string][]byte),
		blocks: 0,
	}
	if isdir {
		entry.entries = make(map[string]uint64)
		entry.ientries = make(map[uint64]string)
	}
	ds.nodes[inode] = entry
	ds.nodes[parent].entries[name] = inode
	ds.nodes[parent].ientries[inode] = name
	atomic.AddUint64(&ds.nodes[parent].nodeCount, 1)
	return name, attr, nil
}

func (ds *InMemDS) Lookup(parent uint64, name string) (string, *pb.Attr, error) {
	ds.RLock()
	defer ds.RUnlock()
	inode, ok := ds.nodes[parent].entries[name]
	if !ok {
		return "", &pb.Attr{}, nil
	}
	entry := ds.nodes[inode]
	return entry.path, entry.attr, nil
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

func (ds *InMemDS) ReadDirAll(inode uint64) (*pb.ReadDirAllResponse, error) {
	ds.RLock()
	defer ds.RUnlock()
	e := &pb.ReadDirAllResponse{}
	for i, _ := range ds.nodes[inode].ientries {
		entry := ds.nodes[i]
		if entry.isdir {
			e.DirEntries = append(e.DirEntries, &pb.DirEnt{Name: entry.path, Attr: entry.attr})
		} else {
			e.FileEntries = append(e.FileEntries, &pb.DirEnt{Name: entry.path, Attr: entry.attr})
		}
	}
	sort.Sort(ByDirent(e.DirEntries))
	sort.Sort(ByDirent(e.FileEntries))
	return e, nil
}

func (ds *InMemDS) Remove(parent uint64, name string) (int32, error) {
	ds.Lock()
	defer ds.Unlock()
	inode, ok := ds.nodes[parent].entries[name]
	if !ok {
		return 1, nil
	}
	delete(ds.nodes, inode)
	delete(ds.nodes[parent].entries, name)
	delete(ds.nodes[parent].ientries, inode)
	atomic.AddUint64(&ds.nodes[parent].nodeCount, ^uint64(0)) // -1
	return 0, nil
}

func (ds *InMemDS) Update(inode, block, blocksize, size uint64, mtime int64) {
	// NOTE: Not sure what this function really should look like yet
	blocks := ds.nodes[inode].blocks
	if block >= blocks {
		ds.nodes[inode].blocks = block + 1
		ds.nodes[inode].lastblock = size
		ds.nodes[inode].blocksize = blocksize
		ds.nodes[inode].attr.Size = blocksize*block + size
	} else if block == (blocks - 1) {
		ds.nodes[inode].lastblock = size
		ds.nodes[inode].attr.Size = blocksize*block + size
	}

	ds.nodes[inode].attr.Mtime = mtime
}

func (ds *InMemDS) Symlink(parent uint64, name string, target string, attr *pb.Attr, inode uint64) (*pb.SymlinkResponse, error) {
	ds.Lock()
	defer ds.Unlock()
	if _, exists := ds.nodes[parent].entries[name]; exists {
		return &pb.SymlinkResponse{}, nil
	}
	entry := &Entry{
		path:   name,
		inode:  inode,
		isdir:  false,
		islink: true,
		target: target,
		attr:   attr,
		xattrs: make(map[string][]byte),
	}
	ds.nodes[inode] = entry
	ds.nodes[parent].entries[name] = inode
	ds.nodes[parent].ientries[inode] = name
	atomic.AddUint64(&ds.nodes[parent].nodeCount, 1)
	return &pb.SymlinkResponse{Name: name, Attr: attr}, nil
}

func (ds *InMemDS) Readlink(inode uint64) (*pb.ReadlinkResponse, error) {
	ds.RLock()
	defer ds.RUnlock()
	return &pb.ReadlinkResponse{Target: ds.nodes[inode].target}, nil
}

func (ds *InMemDS) Getxattr(r *pb.GetxattrRequest) (*pb.GetxattrResponse, error) {
	ds.RLock()
	defer ds.RUnlock()
	if xattr, ok := ds.nodes[r.Inode].xattrs[r.Name]; ok {
		return &pb.GetxattrResponse{Xattr: xattr}, nil
	}
	return &pb.GetxattrResponse{}, nil
}

func (ds *InMemDS) Setxattr(r *pb.SetxattrRequest) (*pb.SetxattrResponse, error) {
	ds.Lock()
	defer ds.Unlock()
	if entry, ok := ds.nodes[r.Inode]; ok {
		entry.xattrs[r.Name] = r.Xattr
	}
	return &pb.SetxattrResponse{}, nil
}

func (ds *InMemDS) Listxattr(r *pb.ListxattrRequest) (*pb.ListxattrResponse, error) {
	ds.RLock()
	defer ds.RUnlock()
	resp := &pb.ListxattrResponse{}
	if entry, ok := ds.nodes[r.Inode]; ok {
		names := ""
		for name := range entry.xattrs {
			names += name
			names += "\x00"
		}
		resp.Xattr = []byte(names)
	}
	return resp, nil
}

func (ds *InMemDS) Removexattr(r *pb.RemovexattrRequest) (*pb.RemovexattrResponse, error) {
	ds.Lock()
	defer ds.Unlock()
	if entry, ok := ds.nodes[r.Inode]; ok {
		delete(entry.xattrs, r.Name)
	}
	return &pb.RemovexattrResponse{}, nil
}

func (ds *InMemDS) Rename(r *pb.RenameRequest) (*pb.RenameResponse, error) {
	ds.Lock()
	defer ds.Unlock()
	if inode, ok := ds.nodes[r.Parent].entries[r.OldName]; ok {
		// remove old
		delete(ds.nodes[r.Parent].entries, r.OldName)
		delete(ds.nodes[r.Parent].ientries, inode)
		atomic.AddUint64(&ds.nodes[r.Parent].nodeCount, ^uint64(0)) // -1
		// add new
		ds.nodes[inode].path = r.NewName
		ds.nodes[r.NewDir].entries[r.NewName] = inode
		ds.nodes[r.NewDir].ientries[inode] = r.NewName
		atomic.AddUint64(&ds.nodes[r.NewDir].nodeCount, 1)
	}
	return &pb.RenameResponse{}, nil
}
