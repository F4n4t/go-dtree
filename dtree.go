package dtree

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

var (
	ErrNotFound = errors.New("file not found")
)

type skipType int

const (
	skipNothing skipType = iota
	skipFile
	skipDir
)

// Collect traverses a directory structure starting at the given root path and constructs a hierarchical representation.
// Exclude can be used to exclude specific files or folders (see filepath.Match for pattern usage, case is ignored).
// Matching for exclude pattern starts after root directory.
// Returns the root node of the constructed file tree or an error if traversal fails.
func Collect(root string, exclude ...string) (*Node, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("get absolute path: %w", err)
	}

	parents := make(map[string]*Node)

	walkFunc := func(path string, fileInfo fs.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if len(exclude) > 0 {
			skip, err := checkExcludeList(absRoot, path, fileInfo, exclude)
			if err != nil {
				return fmt.Errorf("check ignore list: %w", err)
			}
			switch skip {
			case skipFile:
				return nil
			case skipDir:
				return fs.SkipDir
			default:
				// skipNothing
			}
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

// checkExcludeList checks if a file or directory should be skipped based on the provided patterns.
func checkExcludeList(root, path string, fileInfo fs.FileInfo, exclude []string) (skipType, error) {
	relPath, err := filepath.Rel(root, path)
	if err != nil {
		return skipNothing, fmt.Errorf("get relative path: %w", err)
	}

	skip, err := shouldSkip(relPath, exclude, true)
	if err != nil {
		return skipNothing, fmt.Errorf("check ignore list: %w", err)
	}

	if !skip {
		return skipNothing, nil
	}

	if fileInfo.IsDir() {
		return skipDir, nil
	}

	return skipFile, nil
}

// shouldSkip is a helper function to check if the file or folder should be skipped.
func shouldSkip(path string, pattern []string, ignoreCase bool) (bool, error) {
	for _, p := range pattern {
		var (
			name  string
			match bool
			err   error
		)

		if strings.Contains(p, string(filepath.Separator)) {
			name = path
		} else {
			name = filepath.Base(path)
		}

		if ignoreCase {
			match, err = filepath.Match(strings.ToLower(p), strings.ToLower(name))
		} else {
			match, err = filepath.Match(p, name)
		}
		if err != nil {
			return false, fmt.Errorf("pattern %s: %w", p, err)
		}

		if match {
			return true, nil
		}
	}

	return false, nil
}
