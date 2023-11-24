// Explorerfs implements a file system with cheops content
//
// After mounting, the hierarchy is the following:
// /mount/point
//	├── site1
//	|	├── id1
//	|	├── id2
//	|	├── id3
//	└── site2
//		├── id1
//		└── id2

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"cheops.com/model"
	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
	_ "github.com/anacrolix/fuse/fs/fstestutil"
	"github.com/anacrolix/fuse/fuseutil"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s MOUNTPOINT\n", os.Args[0])
	flag.PrintDefaults()
}

func run(mountpoint string) error {
	nodes := getnodes()
	c, err := fuse.Mount(
		mountpoint,
		fuse.FSName("explorer"),
		fuse.Subtype("explorerfs"),
		fuse.LocalVolume(),
		fuse.VolumeName("Explorer filesystem"),
	)
	if err != nil {
		return err
	}
	defer c.Close()

	srv := fs.New(c, nil)
	filesys := &FS{
		nodes: nodes,
	}
	if err := srv.Serve(filesys); err != nil {
		return err
	}

	// Check if the mount process has an error to report.
	<-c.Ready
	if err := c.MountError; err != nil {
		return err
	}
	return nil
}

func getnodes() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	file, err := os.Open(path.Join(home, ".oarnodes"))
	if err != nil {
		log.Fatal("Missing $HOME/.oarnodes file")
	}
	defer file.Close()

	nodes := make([]string, 0)
	scan := bufio.NewScanner(file)
	for scan.Scan() {
		nodes = append(nodes, scan.Text())
	}
	return nodes
}

func main() {
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() != 1 {
		usage()
		os.Exit(2)
	}
	mountpoint := flag.Arg(0)

	if err := run(mountpoint); err != nil {
		log.Fatal(err)
	}
}

type FS struct {
	nodes []string
}

var _ fs.FS = (*FS)(nil)

func (f *FS) Root() (fs.Node, error) {
	return &Dir{fs: f}, nil
}

// Dir implements both Node and Handle for the root directory.
type Dir struct {
	fs *FS
}

var _ fs.Node = (*Dir)(nil)

func (d *Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = 1
	a.Mode = os.ModeDir | 0o555
	return nil
}

var _ fs.NodeStringLookuper = (*Dir)(nil)

func (d *Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	for _, e := range d.fs.nodes {
		if e == name {
			return &NodeDir{
				node: name,
			}, nil
		}
	}
	return nil, syscall.ENOENT
}

var _ fs.HandleReadDirAller = (*Dir)(nil)

func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	dirs := make([]fuse.Dirent, 0)
	for _, name := range d.fs.nodes {
		dirs = append(dirs, fuse.Dirent{Inode: 1, Name: name, Type: fuse.DT_Dir})
	}
	return dirs, nil
}

type NodeDir struct {
	node string
}

func (n *NodeDir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = 0
	a.Mode = os.ModeDir | 0o555
	return nil
}

var _ fs.NodeStringLookuper = (*NodeDir)(nil)

func (n *NodeDir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	ids, err := getIds(n.node)
	if err != nil {
		return nil, err
	}

	for _, v := range ids {
		if name == v {
			return &ResourceFile{
				node: n.node,
				id:   v,
			}, nil
		}
	}
	return nil, syscall.ENOENT
}

var _ fs.HandleReadDirAller = (*NodeDir)(nil)

func (n *NodeDir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	ids, err := getIds(n.node)
	if err != nil {
		return nil, err
	}

	entries := make([]fuse.Dirent, 0)
	for _, id := range ids {
		entries = append(entries, fuse.Dirent{
			Inode: 0,
			Name:  id,
			Type:  fuse.DT_File,
		})
	}
	return entries, nil
}

func getIds(node string) ([]string, error) {
	url := fmt.Sprintf("http://%s:5984/cheops/_find", node)
	res, err := http.Post(url, "application/json", strings.NewReader(`{"selector": {"Type": "RESOURCE"}}`))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var r struct {
		Docs []model.ResourceDocument
	}
	err = json.NewDecoder(res.Body).Decode(&r)
	if err != nil {
		return nil, err
	}

	var ids []string
	for _, doc := range r.Docs {
		ids = append(ids, doc.Id)
	}
	return ids, nil
}

type ResourceFile struct {
	node    string
	id      string
	content []byte
}

var _ fs.Node = (*ResourceFile)(nil)

func (rf *ResourceFile) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = 0
	a.Mode = 0o444

	c, err := getResource(rf.node, rf.id)
	if err != nil {
		return err
	}
	a.Size = uint64(len(c))
	rf.content = c
	return nil
}

var _ fs.NodeOpener = (*File)(nil)

func (rf *ResourceFile) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	if !req.Flags.IsReadOnly() {
		return nil, fuse.Errno(syscall.EACCES)
	}
	resp.Flags |= fuse.OpenKeepCache
	return rf, nil
}

var _ fs.Handle = (*ResourceFile)(nil)
var _ fs.HandleReadAller = (*ResourceFile)(nil)

func (rf *ResourceFile) ReadAll(ctx context.Context) ([]byte, error) {
	return rf.content, nil
}

func getResource(node, id string) ([]byte, error) {
	url := fmt.Sprintf("http://%s:5984/cheops/%s", node, id)
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var r model.ResourceDocument
	err = json.NewDecoder(res.Body).Decode(&r)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(r, "", "\t")
}

type File struct {
	fuse    *fs.Server
	content atomic.Value
	count   uint64
}

var _ fs.Node = (*File)(nil)

func (f *File) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = 0
	a.Mode = 0o444
	t := f.content.Load().(string)
	a.Size = uint64(len(t))
	return nil
}

var _ fs.NodeOpener = (*File)(nil)

func (f *File) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	if !req.Flags.IsReadOnly() {
		return nil, fuse.Errno(syscall.EACCES)
	}
	resp.Flags |= fuse.OpenKeepCache
	return f, nil
}

var _ fs.Handle = (*File)(nil)

var _ fs.HandleReader = (*File)(nil)

func (f *File) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	t := f.content.Load().(string)
	fuseutil.HandleRead(req, resp, []byte(t))
	return nil
}

func (f *File) tick() {
	// Intentionally a variable-length format, to demonstrate size changes.
	f.count++
	s := fmt.Sprintf("%d\t%s\n", f.count, time.Now())
	f.content.Store(s)

	// For simplicity, this example tries to send invalidate
	// notifications even when the kernel does not hold a reference to
	// the node, so be extra sure to ignore ErrNotCached.
	if err := f.fuse.InvalidateNodeData(f); err != nil && err != fuse.ErrNotCached {
		log.Printf("invalidate error: %v", err)
	}
}

func (f *File) update() {
	tick := time.NewTicker(1 * time.Second)
	defer tick.Stop()
	for range tick.C {
		f.tick()
	}
}
