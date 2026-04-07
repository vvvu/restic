package dump

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/restic/restic/internal/archiver"
	"github.com/restic/restic/internal/backend"
	"github.com/restic/restic/internal/data"
	"github.com/restic/restic/internal/fs"
	"github.com/restic/restic/internal/repository"
	"github.com/restic/restic/internal/restic"
	rtest "github.com/restic/restic/internal/test"
)

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

// TestWriteNodeBlobLoadError verifies that writeNode properly handles blob load errors
// without deadlocking. This is a regression test for a bug where the writer goroutine
// would block forever waiting for data from a blob loader that failed.
func TestWriteNodeBlobLoadError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a mock repository that fails on LoadBlob
	mockRepo := &mockBlobLoader{
		loadBlobFunc: func(ctx context.Context, blobType restic.BlobType, id restic.ID, buf []byte) ([]byte, error) {
			return nil, errors.New("simulated blob load failure")
		},
		connectionsFunc: func() uint {
			return 2
		},
	}

	// Create a node with content (blobs that will fail to load)
	node := &data.Node{
		Type:    data.NodeTypeFile,
		Size:    100,
		Content: []restic.ID{restic.NewRandomID(), restic.NewRandomID()},
	}

	dst := &bytes.Buffer{}
	d := New("tar", mockRepo, dst)

	// This should return an error, not deadlock
	// Use a timeout to detect deadlocks
	done := make(chan error, 1)
	go func() {
		done <- d.writeNode(ctx, dst, node)
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected error from writeNode, got nil")
		}
		// Expected - we got an error instead of deadlocking
	case <-time.After(5 * time.Second):
		t.Fatal("writeNode appears to have deadlocked - no response after 5 seconds")
	}
}

// mockBlobLoader is a mock implementation of restic.BlobLoader for testing
type mockBlobLoader struct {
	loadBlobFunc    func(ctx context.Context, blobType restic.BlobType, id restic.ID, buf []byte) ([]byte, error)
	connectionsFunc func() uint
}

func (m *mockBlobLoader) LoadBlob(ctx context.Context, blobType restic.BlobType, id restic.ID, buf []byte) ([]byte, error) {
	return m.loadBlobFunc(ctx, blobType, id, buf)
}

func (m *mockBlobLoader) Connections() uint {
	if m.connectionsFunc != nil {
		return m.connectionsFunc()
	}
	return 2
}

// Embed other required methods with stub implementations
func (m *mockBlobLoader) LookupBlobSize(blobType restic.BlobType, id restic.ID) (size uint, exists bool) {
	return 0, false
}
