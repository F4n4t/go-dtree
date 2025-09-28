package dtree

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildFileTree(t *testing.T) {
	parents := map[string]*Node{
		"release": {
			FullPath: "release",
			Info: &FileInfo{
				Name:  "release",
				IsDir: true,
			},
		},
		"release/a.mkv": {
			FullPath: "release/a.mkv",
			Info: &FileInfo{
				Name:  "a.mkv",
				IsDir: false,
			},
		},
		"release/sample": {
			FullPath: "release/sample",
			Info: &FileInfo{
				Name:  "sample",
				IsDir: true,
			},
		},
		"release/sample/sample.mkv": {
			FullPath: "release/sample/sample.mkv",
			Info: &FileInfo{
				Name:  "sample.mkv",
				IsDir: false,
			},
		},
		"release/subs": {
			FullPath: "release/subs",
			Info: &FileInfo{
				Name:  "subs",
				IsDir: true,
			},
		},
		"release/subs/sub.sub": {
			FullPath: "release/subs/sub.sub",
			Info: &FileInfo{
				Name:  "sub.sub",
				IsDir: false,
			},
		},
		"release/subs/sub.idx": {
			FullPath: "release/subs/sub.idx",
			Info: &FileInfo{
				Name:  "sub.idx",
				IsDir: false,
			},
		},
	}

	gotTree := BuildFileTree(parents)

	assert.Equal(t, "release", gotTree.FullPath)
	assert.Equal(t, "release/a.mkv", gotTree.Children[0].FullPath)
	assert.Equal(t, "release/sample", gotTree.Children[1].FullPath)
	assert.Equal(t, "release/sample/sample.mkv", gotTree.Children[1].Children[0].FullPath)
	assert.Equal(t, "release/subs", gotTree.Children[2].FullPath)
	assert.Equal(t, "release/subs/sub.idx", gotTree.Children[2].Children[0].FullPath)
	assert.Equal(t, "release/subs/sub.sub", gotTree.Children[2].Children[1].FullPath)
}
