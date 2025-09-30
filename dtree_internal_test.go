package dtree

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestShouldSkip(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		pattern       []string
		ignoreCase    bool
		want          bool
		expectedError error
	}{
		{
			name:       "skip by name (ignore case)",
			path:       "/dir/skip_mE",
			pattern:    []string{"skip_me"},
			ignoreCase: true,
			want:       true,
		},
		{
			name:       "skip by name (case-sensitive)",
			path:       "/dir/skip_mE",
			pattern:    []string{"skip_me"},
			ignoreCase: false,
			want:       false,
		},
		{
			name:       "skip by pattern (ignore case)",
			path:       "/dir/skip_mE",
			pattern:    []string{"skip?me"},
			ignoreCase: true,
			want:       true,
		},
		{
			name:       "skip by path (ignore case)",
			path:       "/dir/subdir/skip_mE",
			pattern:    []string{"/dir/subdir/skip_me"},
			ignoreCase: true,
			want:       true,
		},
		{
			name:       "skip by char range",
			path:       "/dir/subdir/skip_me",
			pattern:    []string{"skip_m[a-e]"},
			ignoreCase: true,
			want:       true,
		},
		{
			name:          "invalid range",
			path:          "/dir/subdir/skip_me",
			pattern:       []string{"skip_m[a--]"},
			ignoreCase:    true,
			want:          false,
			expectedError: filepath.ErrBadPattern,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := shouldSkip(tt.path, tt.pattern, tt.ignoreCase)
			if tt.expectedError != nil {
				assert.ErrorIs(t, gotErr, tt.expectedError)
				return
			}
			require.NoError(t, gotErr)

			assert.Equal(t, tt.want, got)
		})
	}
}
