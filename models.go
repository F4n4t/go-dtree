package dtree

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"
)

// Node represents a node in a directory tree.
type Node struct {
	FullPath string    `json:"path"`
	Info     *FileInfo `json:"info"`
	Children []*Node   `json:"children"`
	Parent   *Node     `json:"-"`
}

// FileInfo is a struct created from os.FileInfo interface for serialization.
type FileInfo struct {
	Name      string      `json:"name"`
	Size      int64       `json:"size"`
	IsDir     bool        `json:"is_dir"`
	Mode      os.FileMode `json:"mode"`
	ModTime   time.Time   `json:"mod_time"`
	Extension string      `json:"ext"`
}

// EqualSizes represents a collection of files that share the same size.
type EqualSizes struct {
	Count int
	Files []*Node
}

// FileInfoFromInterface is a helper function to create a local FileInfo struct from os.FileInfo interface.
func FileInfoFromInterface(v os.FileInfo) *FileInfo {
	fileInfo := &FileInfo{
		Name:    v.Name(),
		Size:    v.Size(),
		Mode:    v.Mode(),
		ModTime: v.ModTime(),
		IsDir:   v.IsDir(),
	}

	if !v.IsDir() {
		fileInfo.Extension = filepath.Ext(fileInfo.Name)
	}

	return fileInfo
}

// GetFileInfo retrieves file information for the file at the given path and returns it as a FileInfo struct.
func GetFileInfo(filePath string) (*FileInfo, error) {
	fInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("stat: %w", err)
	}

	return FileInfoFromInterface(fInfo), nil
}

// UpdateFileInfo updates the node's FileInfo by retrieving and parsing the file metadata of its associated path.
func (node *Node) UpdateFileInfo() error {
	fInfo, err := GetFileInfo(node.FullPath)
	if err != nil {
		return err
	}

	node.Info = fInfo

	return nil
}

// GetFiles returns all files or only files with a given extension.
// The extension must include the dot, e.g. ".mkv".
func (node *Node) GetFiles(ext ...string) []*Node {
	var files []*Node
	if len(node.Children) == 0 && !node.Info.IsDir {
		if len(ext) > 0 {
			if node.Info.Extension != ext[0] {
				return files
			}
		}
		files = append(files, node)
	}

	for _, child := range node.Children {
		if child.Info.IsDir {
			files = append(files, child.GetFiles(ext...)...)
		} else {
			if len(ext) > 0 {
				if child.Info.Extension != ext[0] {
					continue
				}
			}
			files = append(files, child)
		}
	}

	return files
}

// GetFile returns a single file by its name.
func (node *Node) GetFile(name string) (*Node, error) {
	for _, f := range node.GetFiles() {
		if f.Info.Name == name {
			return f, nil
		}
	}

	return nil, fmt.Errorf("%s: %w", name, ErrNotFound)
}

// GetFileByPath returns a single file by its path.
func (node *Node) GetFileByPath(filePath string) (*Node, error) {
	if filePath == "" {
		// we can't find the file by path if the path is empty
		return nil, fmt.Errorf("path: %w", ErrNotFound)
	}

	for _, f := range node.GetFiles() {
		if strings.Contains(f.FullPath, filePath) {
			return f, nil
		}
	}

	return nil, fmt.Errorf("%s: %w", filePath, ErrNotFound)
}

// GetFileByAbsolutePath returns a single file by its absolute path.
func (node *Node) GetFileByAbsolutePath(path string) (*Node, error) {
	for _, f := range node.GetFiles() {
		if f.FullPath == path {
			return f, nil
		}
	}
	return nil, fmt.Errorf("%s: %w", path, ErrNotFound)
}

// GetFileBySizeAndExt returns a single file by its size and extension.
func (node *Node) GetFileBySizeAndExt(size int64, ext string, exclude ...string) (*Node, error) {
	for _, f := range node.GetFiles() {
		if f.Info.Size == size && strings.EqualFold(f.Info.Extension, ext) && !slices.Contains(exclude, f.Info.Name) {
			return f, nil
		}
	}

	return nil, fmt.Errorf("size %d, extension %s: %w", size, ext, ErrNotFound)
}

// GetFileByPatternAndExt returns the first match for a given pattern and extension.
func (node *Node) GetFileByPatternAndExt(pattern string, ext string) (*Node, error) {
	pattern = strings.ToLower(pattern)

	for _, f := range node.GetFiles(ext) {
		if strings.Contains(strings.ToLower(f.Info.Name), pattern) {
			return f, nil
		}
	}

	return nil, fmt.Errorf("pattern %s, extension %s: %w", pattern, ext, ErrNotFound)
}

// GetBiggest traverses the node hierarchy to find the file with the largest size. Returns the largest file encountered.
func (node *Node) GetBiggest(biggest *Node) *Node {
	if !node.Info.IsDir {
		return node
	}

	for _, f := range node.Children {
		if f.Info.IsDir {
			biggest = f.GetBiggest(biggest)
		} else if biggest == nil {
			biggest = f
		} else if biggest.Info.Size < f.Info.Size {
			biggest = f
		}
	}
	return biggest
}

// Remove removes a file
func (node *Node) Remove() error {
	if node.Info.IsDir {
		return fmt.Errorf("can't remove directory: %s", node.FullPath)
	}

	if err := os.Remove(node.FullPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// remove the node
			removeNode(node)
		}

		return fmt.Errorf("remove file: %w", err)
	}

	removeNode(node)

	return nil
}

func removeNode(node *Node) {
	children := node.Parent.Children

	for i, c := range node.Parent.Children {
		if c == node {
			if len(children) == 1 {
				children = nil
			} else {
				children[i] = children[len(children)-1]
			}
			break
		}
	}

	if children == nil {
		node.Parent.Children = nil
	} else {
		node.Parent.Children = children[:len(children)-1]
	}
}

// EqualSizes returns a map of files with the same size.
func (node *Node) EqualSizes() map[int64]*EqualSizes {
	files := make(map[int64]*EqualSizes)

	for _, file := range node.GetFiles() {
		size := file.Info.Size
		if _, ok := files[size]; ok {
			files[size].Count++
			files[size].Files = append(files[size].Files, file)
		} else {
			files[size] = &EqualSizes{Count: 1, Files: []*Node{file}}
		}
	}

	return files
}

// sortNodesRecursively sorts the children of a node alphabetically by name and applies the sort recursively to
// directories.
func (node *Node) sortNodesRecursively() {
	if node == nil || len(node.Children) == 0 {
		return
	}

	sort.Slice(node.Children, func(i, j int) bool {
		return node.Children[i].Info.Name < node.Children[j].Info.Name
	})

	for _, child := range node.Children {
		if child.Info.IsDir {
			child.sortNodesRecursively()
		}
	}
}
