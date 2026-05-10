package carrier

import (
	projectpath "TidyFS/project_path"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

type TidyFSNode struct {
	ID       uint64
	Name     string
	Path     string
	IsDir    bool
	children []*TidyFSNode
}

type TidyFileHandle struct {
	file *os.File
}

type FuseNode struct {
	fs.Inode
	node *TidyFSNode
}

func (n *FuseNode) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	if !n.node.IsDir {
		return nil, syscall.ENOTDIR
	}

	entries := make([]fuse.DirEntry, 0, len(n.node.children))

	for _, child := range n.node.children {
		mode := uint32(fuse.S_IFREG)

		if child.IsDir {
			mode = fuse.S_IFDIR
		}
		entries = append(entries, fuse.DirEntry{Name: child.Name, Mode: mode})
	}

	return fs.NewListDirStream(entries), 0
}

func (n *FuseNode) Getattr(
	ctx context.Context,
	fh fs.FileHandle,
	out *fuse.AttrOut,
) syscall.Errno {
	if n.node.IsDir {
		out.Mode = fuse.S_IFDIR | 0755
		out.Size = 0
		return 0
	}

	out.Mode = fuse.S_IFREG | 0644

	if n.node.Path != "" {
		if st, err := os.Stat(n.node.Path); err == nil {
			out.Size = uint64(st.Size())
		}
	}

	return 0
}

func (n *FuseNode) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	if !n.node.IsDir {
		return nil, syscall.ENOTDIR
	}

	for _, child := range n.node.children {
		if child.Name != name {
			continue
		}

		mode := uint32(fuse.S_IFREG | 0644)
		if child.IsDir {
			mode = fuse.S_IFDIR | 0755
		}

		var size uint64
		if !child.IsDir && child.Path != "" {
			if st, err := os.Stat(child.Path); err == nil {
				size = uint64(st.Size())
			}
		}

		out.Attr.Mode = mode
		out.Attr.Size = size

		stableMode := uint32(fuse.S_IFREG)
		if child.IsDir {
			stableMode = fuse.S_IFDIR
		}

		childNode := &FuseNode{
			node: child,
		}

		inode := n.NewInode(ctx, childNode, fs.StableAttr{
			Mode: stableMode,
			Ino:  child.ID,
		})

		return inode, 0
	}

	return nil, syscall.ENOENT
}

func (h *TidyFileHandle) Release(ctx context.Context) syscall.Errno {
	if err := h.file.Close(); err != nil {
		return errnoFromErr(err)
	}

	return 0
}

func (h *TidyFileHandle) Write(
	ctx context.Context,
	data []byte,
	off int64,
) (uint32, syscall.Errno) {
	n, err := h.file.WriteAt(data, off)
	if err != nil {
		return uint32(n), errnoFromErr(err)
	}

	return uint32(n), 0
}

func (n *FuseNode) Open(
	ctx context.Context,
	flags uint32,
) (fs.FileHandle, uint32, syscall.Errno) {
	if n.node.IsDir {
		return nil, 0, syscall.EISDIR
	}

	if n.node.Path == "" {
		return nil, 0, syscall.ENOENT
	}

	var openFlags int

	switch flags & syscall.O_ACCMODE {
	case syscall.O_RDONLY:
		openFlags = os.O_RDONLY
	case syscall.O_WRONLY:
		openFlags = os.O_WRONLY
	case syscall.O_RDWR:
		openFlags = os.O_RDWR
	default:
		openFlags = os.O_RDONLY
	}

	if flags&syscall.O_APPEND != 0 {
		openFlags |= os.O_APPEND
	}
	if flags&syscall.O_TRUNC != 0 {
		openFlags |= os.O_TRUNC
	}

	file, err := os.OpenFile(n.node.Path, openFlags, 0644)
	if err != nil {
		return nil, 0, errnoFromErr(err)
	}

	return &TidyFileHandle{file: file}, 0, 0
}

func (h *TidyFileHandle) Read(
	ctx context.Context,
	dest []byte,
	off int64,
) (fuse.ReadResult, syscall.Errno) {
	buf := make([]byte, len(dest))

	n, err := h.file.ReadAt(buf, off)
	if err != nil && err != io.EOF {
		return nil, errnoFromErr(err)
	}

	return fuse.ReadResultData(buf[:n]), 0
}

func FuseRun(ctx context.Context, mountPoint string) error {
	if mountPoint == "" {
		return fmt.Errorf("Mount dir is empty")
	}

	filesys, err := buildFS()

	if err != nil {
		return err
	}

	server, err := fs.Mount(mountPoint,
		&FuseNode{
			node: filesys,
		},
		&fs.Options{
			MountOptions: fuse.MountOptions{
				Debug: false,
			},
		})

	if err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		_ = server.Unmount()
	}()

	go func() {
		server.Wait()
	}()
	return nil
}

func buildFS() (*TidyFSNode, error) {
	ids := &inodeGenerator{}

	root := &TidyFSNode{
		ID:       ids.Next(),
		Name:     "",
		IsDir:    true,
		children: make([]*TidyFSNode, 0, 10),
	}

	data, err := os.ReadFile(projectpath.ClassifiedFilesJSON())
	if err != nil {
		return nil, err
	}

	var classifiedFiles []ClassifiedFile
	if err := json.Unmarshal(data, &classifiedFiles); err != nil {
		return nil, err
	}

	for _, file := range classifiedFiles {
		if file.Category == "" {
			return nil, fmt.Errorf("empty category")
		}

		names, err := ExtractCategoryNames(file.Category)
		if err != nil {
			return nil, err
		}

		current := root

		for _, categoryName := range names {
			if categoryName == "" {
				continue
			}

			current = findOrCreateDirWithID(current, categoryName, ids)
		}

		current.children = append(current.children, &TidyFSNode{
			ID:    ids.Next(),
			Name:  file.Name,
			Path:  file.Path,
			IsDir: false,
		})
	}

	return root, nil
}

func findOrCreateDirWithID(parent *TidyFSNode, name string, ids *inodeGenerator) *TidyFSNode {
	for _, child := range parent.children {
		if child.IsDir && child.Name == name {
			return child
		}
	}

	child := &TidyFSNode{
		ID:       ids.Next(),
		Name:     name,
		IsDir:    true,
		children: make([]*TidyFSNode, 0),
	}

	parent.children = append(parent.children, child)
	return child
}

func errnoFromErr(err error) syscall.Errno {
	if err == nil {
		return 0
	}

	if errno, ok := err.(syscall.Errno); ok {
		return errno
	}

	if os.IsNotExist(err) {
		return syscall.ENOENT
	}
	if os.IsPermission(err) {
		return syscall.EACCES
	}

	return syscall.EIO
}

type inodeGenerator struct {
	next uint64
}

func (g *inodeGenerator) Next() uint64 {
	g.next++
	return g.next
}
