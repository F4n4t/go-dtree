package dtree

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
)

var (
	ErrNotFound = errors.New("file not found")
)

// Collect traverses a directory structure starting at the given root path and constructs a hierarchical representation.
// Returns the root node of the constructed file tree or an error if traversal fails.
func Collect(root string) (*Node, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("get absolute path: %w", err)
	}

	parents := make(map[string]*Node)

	walkFunc := func(path string, fileInfo fs.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		node := &Node{
			FullPath: path,
			Info:     FileInfoFromInterface(fileInfo),
		}

		parents[path] = node

		return nil
	}

	if err := filepath.Walk(absRoot, walkFunc); err != nil {
		return nil, fmt.Errorf("walk path: %w", err)
	}

	return BuildFileTree(parents), nil
}

// BuildFileTree constructs a hierarchical file tree from a map of nodes, identifying the root and linking children.
// It associates parent nodes with their children and sorts the tree nodes recursively.
// Panics if no root node is identified.
func BuildFileTree(parents map[string]*Node) *Node {
	var root *Node

	for path, node := range parents {
		parentPath := filepath.Dir(path)
		parent, ok := parents[parentPath]
		if ok {
			node.Parent = parent
			parent.Children = append(parent.Children, node)
		} else {
			root = node
		}
	}

	if root == nil {
		// should never happen
		panic("no root node found")
	}

	root.sortNodesRecursively()

	return root
}
