package dtree_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/f4n4t/go-dtree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDir(t *testing.T, baseDir string, testFiles map[string][]byte) {
	for name, content := range testFiles {
		dir, file := filepath.Split(name)
		if dir != "" {
			require.NoError(t, os.MkdirAll(filepath.Join(baseDir, dir), 0755))
		}
		require.NoError(t, os.WriteFile(filepath.Join(baseDir, dir, file), content, 0666))
	}
}

func createFileNode(path string, isDir bool, size int64) *dtree.Node {
	name := filepath.Base(path)
	ext := ""
	if !isDir {
		ext = filepath.Ext(path)
	}

	return &dtree.Node{
		FullPath: path,
		Info: &dtree.FileInfo{
			Name:      name,
			IsDir:     isDir,
			Size:      size,
			Extension: ext,
		},
	}
}

// equalNode is a helper function to verify a node without ModTime and Extension.
func equalNode(t *testing.T, expected *dtree.Node, actual *dtree.Node) {
	assert.Equal(t, expected.FullPath, actual.FullPath)
	assert.Equal(t, expected.Info.Name, actual.Info.Name)
	assert.Equal(t, expected.Info.IsDir, actual.Info.IsDir)
	assert.Equal(t, expected.Info.Size, actual.Info.Size)
	assert.Equal(t, expected.Info.Extension, actual.Info.Extension)
	for i, expectedChild := range expected.Children {
		equalNode(t, expectedChild, actual.Children[i])
	}
}

func TestCollect(t *testing.T) {
	// setup test files
	testFiles := map[string][]byte{
		filepath.Join("test", "test.nfo"):  []byte("asd"),
		filepath.Join("test", "test.mkv"):  []byte("asdf"),
		filepath.Join("test", "test2.mkv"): []byte("asdf"),
	}
	tempDir := t.TempDir()
	setupTestDir(t, tempDir, testFiles)

	// collect structure
	rootNode, err := dtree.Collect(tempDir)
	require.NoError(t, err)

	// create the expected structure
	expectedParent := createFileNode(tempDir, true, 4096)
	child := createFileNode(filepath.Join(tempDir, "test"), true, 4096)
	totalSize := 11 // sum of all the files
	subChild1 := createFileNode(filepath.Join(child.FullPath, "test.mkv"), false, 4)
	subChild2 := createFileNode(filepath.Join(child.FullPath, "test.nfo"), false, 3)
	subChild3 := createFileNode(filepath.Join(child.FullPath, "test2.mkv"), false, 4)
	child.Children = append(child.Children, subChild1, subChild2, subChild3)
	expectedParent.Children = append(expectedParent.Children, child)

	t.Run("VerifyCollectedStructure", func(t *testing.T) {
		equalNode(t, expectedParent, rootNode)
	})

	t.Run("GetFiles", func(t *testing.T) {
		t.Run("WithExtension", func(t *testing.T) {
			foundNodes := rootNode.GetFiles(".mkv")
			require.Len(t, foundNodes, 2)
			equalNode(t, foundNodes[0], subChild1)
			equalNode(t, foundNodes[1], subChild3)
		})

		t.Run("WithExtensionOnSingleFile", func(t *testing.T) {
			gotNode, gotErr := rootNode.GetFileByPath(subChild1.FullPath)
			require.NoError(t, gotErr)

			// right extension
			foundNodes := gotNode.GetFiles(".mkv")

			require.Len(t, foundNodes, 1)
			equalNode(t, foundNodes[0], gotNode)

			// wrong extension
			foundNodes = gotNode.GetFiles(".wrong")
			require.Len(t, foundNodes, 0)
		})

		t.Run("WithoutExtension", func(t *testing.T) {
			foundNodes := rootNode.GetFiles()
			require.Len(t, foundNodes, 3)
			equalNode(t, foundNodes[0], subChild1)
			equalNode(t, foundNodes[1], subChild2)
			equalNode(t, foundNodes[2], subChild3)
		})

		t.Run("WithWrongExtension", func(t *testing.T) {
			foundNodes := rootNode.GetFiles(".txt")
			assert.Len(t, foundNodes, 0)
		})
	})

	t.Run("GetFile", func(t *testing.T) {
		t.Run("ExistingFile", func(t *testing.T) {
			foundNode, gotErr := rootNode.GetFile("test.mkv")
			require.NoError(t, gotErr)
			equalNode(t, subChild1, foundNode)
		})

		t.Run("NonExistingFile", func(t *testing.T) {
			_, gotErr := rootNode.GetFile("does_not_exist.txt")
			assert.ErrorIs(t, gotErr, dtree.ErrNotFound)
		})
	})

	t.Run("GetFileByPath", func(t *testing.T) {
		testCases := []struct {
			name     string
			path     string
			expected *dtree.Node
			wantErr  bool
		}{
			{
				name:     "ExistingRelativePath",
				path:     filepath.Join("test", "test.mkv"),
				expected: subChild1,
				wantErr:  false,
			},
			{
				name:     "ExistingAbsolutePath",
				path:     subChild1.FullPath,
				expected: subChild1,
				wantErr:  false,
			},
			{
				name:     "NonExistingPath",
				path:     filepath.Join("asd", "blub", "does_not_exist.txt"),
				expected: nil,
				wantErr:  true,
			},
			{
				name:     "EmptyPath",
				path:     "",
				expected: nil,
				wantErr:  true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				foundNode, gotErr := rootNode.GetFileByPath(tc.path)
				if tc.wantErr {
					assert.ErrorIs(t, gotErr, dtree.ErrNotFound)
				} else {
					require.NoError(t, gotErr)
					equalNode(t, tc.expected, foundNode)
				}
			})
		}
	})

	t.Run("GetFileByAbsolutePath", func(t *testing.T) {
		testCases := []struct {
			name     string
			path     string
			expected *dtree.Node
			wantErr  bool
		}{
			{
				name:     "ExistingAbsolutePath",
				path:     subChild1.FullPath,
				expected: subChild1,
				wantErr:  false,
			},
			{
				name:     "RelativePath",
				path:     filepath.Join("test", "test.mkv"),
				expected: nil,
				wantErr:  true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				foundNode, gotErr := rootNode.GetFileByAbsolutePath(tc.path)
				if tc.wantErr {
					assert.ErrorIs(t, gotErr, dtree.ErrNotFound)
				} else {
					require.NoError(t, gotErr)
					equalNode(t, tc.expected, foundNode)
				}
			})
		}
	})

	t.Run("GetFileBySizeAndExt", func(t *testing.T) {
		testCases := []struct {
			name     string
			size     int64
			ext      string
			excludes []string
			expected *dtree.Node
			wantErr  bool
		}{
			{
				name:     "MatchingSizeAndExt",
				size:     4,
				ext:      ".mkv",
				excludes: nil,
				expected: subChild1,
				wantErr:  false,
			},
			{
				name:     "ExcludedName",
				size:     4,
				ext:      ".mkv",
				excludes: []string{"test.mkv", "test2.mkv"},
				expected: nil,
				wantErr:  true,
			},
			{
				name:     "CorrectSizeWrongExt",
				size:     4,
				ext:      ".nfo",
				excludes: nil,
				expected: nil,
				wantErr:  true,
			},
			{
				name:     "WrongSizeCorrectExt",
				size:     1337,
				ext:      ".mkv",
				excludes: nil,
				expected: nil,
				wantErr:  true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				foundNode, gotErr := rootNode.GetFileBySizeAndExt(tc.size, tc.ext, tc.excludes...)
				if tc.wantErr {
					assert.ErrorIs(t, gotErr, dtree.ErrNotFound)
				} else {
					require.NoError(t, gotErr)
					equalNode(t, tc.expected, foundNode)
				}
			})
		}
	})

	t.Run("GetFileByPatternAndExt", func(t *testing.T) {
		testCases := []struct {
			name     string
			pattern  string
			ext      string
			expected *dtree.Node
			wantErr  bool
		}{
			{
				name:     "MatchingPatternAndExt",
				pattern:  "test",
				ext:      ".mkv",
				expected: subChild1,
				wantErr:  false,
			},
			{
				name:     "NonExistingPattern",
				pattern:  "does_not_exist",
				ext:      ".mkv",
				expected: nil,
				wantErr:  true,
			},
			{
				name:     "MatchingPatternWrongExt",
				pattern:  "test",
				ext:      ".txt",
				expected: nil,
				wantErr:  true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				foundNode, gotErr := rootNode.GetFileByPatternAndExt(tc.pattern, tc.ext)
				if tc.wantErr {
					assert.ErrorIs(t, gotErr, dtree.ErrNotFound)
				} else {
					require.NoError(t, gotErr)
					equalNode(t, tc.expected, foundNode)
				}
			})
		}
	})

	t.Run("GetFileInfo", func(t *testing.T) {
		t.Run("NonExistingPath", func(t *testing.T) {
			_, gotErr := dtree.GetFileInfo("/path/that/doesnt/exist")
			assert.ErrorIs(t, gotErr, os.ErrNotExist)
		})

		t.Run("ValidPath", func(t *testing.T) {
			fInfo, gotErr := dtree.GetFileInfo(subChild1.FullPath)
			require.NoError(t, gotErr)
			expectedInfo, gotErr := rootNode.GetFileByPath(subChild1.FullPath)
			require.NoError(t, gotErr)
			assert.Equal(t, expectedInfo.Info, fInfo)
		})
	})

	t.Run("GetBiggest", func(t *testing.T) {
		t.Run("FromRoot", func(t *testing.T) {
			foundNode := rootNode.GetBiggest(nil)
			equalNode(t, subChild1, foundNode)
		})

		t.Run("FromSubChild", func(t *testing.T) {
			foundNode := subChild2.GetBiggest(nil)
			equalNode(t, subChild2, foundNode)
		})
	})

	t.Run("EqualSizes", func(t *testing.T) {
		gotMap := rootNode.EqualSizes()
		require.Len(t, gotMap, 2)

		assert.Equal(t, gotMap[4].Count, 2)
		require.Len(t, gotMap[4].Files, 2)
		equalNode(t, gotMap[4].Files[0], subChild1)
		equalNode(t, gotMap[4].Files[1], subChild3)

		assert.Equal(t, gotMap[3].Count, 1)
		require.Len(t, gotMap[3].Files, 1)
		equalNode(t, gotMap[3].Files[0], subChild2)
	})

	t.Run("TotalSize", func(t *testing.T) {
		assert.EqualValues(t, totalSize, rootNode.GetTotalSize())
	})

	// this must come before TotalSize, because it changes the size.
	t.Run("UpdateFileInfo", func(t *testing.T) {
		node, gotErr := rootNode.GetFileByPath(subChild1.FullPath)
		require.NoError(t, gotErr)

		oldSize := node.Info.Size

		// append to the file
		f, err := os.OpenFile(subChild1.FullPath, os.O_APPEND|os.O_WRONLY, 0666)
		require.NoError(t, err)
		_, err = f.WriteString("asd")
		require.NoError(t, err)
		require.NoError(t, f.Close())

		// update the node
		assert.NoError(t, node.UpdateFileInfo())

		expectedNode := createFileNode(subChild1.FullPath, false, oldSize+3)
		equalNode(t, expectedNode, node)
	})

	// this must be the last test, because it removes files
	t.Run("Remove", func(t *testing.T) {
		// remove existing node
		toRemoveNode, gotErr := rootNode.GetFile(subChild1.Info.Name)
		require.NoError(t, gotErr)

		assert.NoError(t, toRemoveNode.Remove())
		assert.NoFileExists(t, toRemoveNode.FullPath)
		_, err := rootNode.GetFile("test.mkv")
		require.ErrorIs(t, err, dtree.ErrNotFound)

		assert.ErrorContains(t, rootNode.Remove(), "can't remove directory")

		// remove file beforehand
		gotErr = os.Remove(subChild2.FullPath)
		require.NoError(t, gotErr)

		// try to remove a non-existing file
		toRemoveNode, gotErr = rootNode.GetFile(subChild2.Info.Name)
		require.NoError(t, gotErr)
		assert.ErrorIs(t, toRemoveNode.Remove(), os.ErrNotExist)

		// verify that the node got removed
		_, gotErr = rootNode.GetFile(subChild2.Info.Name)
		require.ErrorIs(t, gotErr, dtree.ErrNotFound)

		// remove remaining file
		toRemoveNode, gotErr = rootNode.GetFile(subChild3.Info.Name)
		require.NoError(t, gotErr)

		assert.NoError(t, toRemoveNode.Remove())
		assert.NoFileExists(t, toRemoveNode.FullPath)
	})
}
