package dump

import (
	"bytes"
	"context"
	"testing"

	"github.com/restic/restic/internal/archiver"
	"github.com/restic/restic/internal/backend"
	"github.com/restic/restic/internal/data"
	"github.com/restic/restic/internal/fs"
	"github.com/restic/restic/internal/repository"
	"github.com/restic/restic/internal/restic"
	rtest "github.com/restic/restic/internal/test"
)

// TestSendNodesNilSubtree tests that sendNodes handles directory nodes with nil Subtree gracefully.
// This is a regression test for a bug where sendNodes dereferenced node.Subtree without checking
// if it was nil first, which could cause a panic on corrupted repository data.
func TestSendNodesNilSubtree(t *testing.T) {
	ctx := context.Background()

	// Create a directory node with nil Subtree (simulating corrupted data)
	dirNode := &data.Node{
		Name:    "testdir",
		Type:    data.NodeTypeDir,
		Subtree: nil, // explicitly nil to test the fix
	}

	repo, _, _ := repository.TestRepositoryWithVersion(t, 0)
	ch := make(chan *data.Node, 10)

	// This should return an error, not panic
	err := sendNodes(ctx, repo, dirNode, ch)
	if err == nil {
		t.Errorf("sendNodes should return error for nil subtree, got nil")
	}

	close(ch)
}

// TestSendNodesFileNode tests that sendNodes correctly handles file nodes.
func TestSendNodesFileNode(t *testing.T) {
	ctx := context.Background()

	fileNode := &data.Node{
		Name:    "testfile",
		Type:    data.NodeTypeFile,
		Mode:    0644,
		Content: restic.IDs{},
	}

	repo, _, _ := repository.TestRepositoryWithVersion(t, 0)
	ch := make(chan *data.Node, 10)

	// This should not error for file nodes
	err := sendNodes(ctx, repo, fileNode, ch)
	if err != nil {
		t.Errorf("sendNodes returned error for file node: %v", err)
	}

	// Verify the node was sent
	node := <-ch
	if node != fileNode {
		t.Errorf("sendNodes should send the file node")
	}

	close(ch)
}

func prepareTempdirRepoSrc(t testing.TB, src archiver.TestDir) (string, restic.Repository, backend.Backend) {
	tempdir := rtest.TempDir(t)
	repo, _, be := repository.TestRepositoryWithVersion(t, 0)

	archiver.TestCreateFiles(t, tempdir, src)

	return tempdir, repo, be
}

type CheckDump func(t *testing.T, testDir string, testDump *bytes.Buffer) error

func WriteTest(t *testing.T, format string, cd CheckDump) {
	tests := []struct {
		name   string
		args   archiver.TestDir
		target string
	}{
		{
			name: "single file in root",
			args: archiver.TestDir{
				"file": archiver.TestFile{Content: "string"},
			},
			target: "/",
		},
		{
			name: "multiple files in root",
			args: archiver.TestDir{
				"file1": archiver.TestFile{Content: "string"},
				"file2": archiver.TestFile{Content: "string"},
			},
			target: "/",
		},
		{
			name: "multiple files and folders in root",
			args: archiver.TestDir{
				"file1": archiver.TestFile{Content: "string"},
				"file2": archiver.TestFile{Content: "string"},
				"firstDir": archiver.TestDir{
					"another": archiver.TestFile{Content: "string"},
				},
				"secondDir": archiver.TestDir{
					"another2": archiver.TestFile{Content: "string"},
				},
			},
			target: "/",
		},
		{
			name: "file and symlink in root",
			args: archiver.TestDir{
				"file1": archiver.TestFile{Content: "string"},
				"file2": archiver.TestSymlink{Target: "file1"},
			},
			target: "/",
		},
		{
			name: "directory only",
			args: archiver.TestDir{
				"firstDir": archiver.TestDir{
					"secondDir": archiver.TestDir{},
				},
			},
			target: "/",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			tmpdir, repo, be := prepareTempdirRepoSrc(t, tt.args)
			arch := archiver.New(repo, fs.Track{FS: fs.Local{}}, archiver.Options{})

			back := rtest.Chdir(t, tmpdir)
			defer back()

			sn, _, _, err := arch.Snapshot(ctx, []string{"."}, archiver.SnapshotOptions{})
			rtest.OK(t, err)

			tree, err := data.LoadTree(ctx, repo, *sn.Tree)
			rtest.OK(t, err)

			dst := &bytes.Buffer{}
			d := New(format, repo, dst)
			if err := d.DumpTree(ctx, tree, tt.target); err != nil {
				t.Fatalf("Dumper.Run error = %v", err)
			}
			if err := cd(t, tmpdir, dst); err != nil {
				t.Errorf("WriteDump() = does not match: %v", err)
			}

			// test that dump returns an error if the repository is broken
			tree, err = data.LoadTree(ctx, repo, *sn.Tree)
			rtest.OK(t, err)
			rtest.OK(t, be.Delete(ctx))
			// use new dumper as the old one has the blobs cached
			d = New(format, repo, dst)
			err = d.DumpTree(ctx, tree, tt.target)
			rtest.Assert(t, err != nil, "expected error, got nil")
		})
	}
}
