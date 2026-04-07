package main

import (
	"testing"

	"github.com/restic/restic/internal/data"
	"github.com/restic/restic/internal/repository"
	"github.com/restic/restic/internal/restic"
	rtest "github.com/restic/restic/internal/test"
)

// TestAddBlobsNilSubtree tests that addBlobs handles directory nodes
// with nil Subtree gracefully (regression test for nil pointer dereference)
func TestAddBlobsNilSubtree(t *testing.T) {
	// Test directory node with nil Subtree - should not panic
	dirNode := &data.Node{
		Name:    "testdir",
		Type:    data.NodeTypeDir,
		Subtree: nil, // explicitly nil to test the fix
	}

	// Create a minimal mock repository for testing
	repo, _, _ := repository.TestRepositoryWithVersion(t, 0)

	// This should not panic
	addBlobs(repo.NewAssociatedBlobSet(), dirNode)

	// Test directory node with valid Subtree - should work normally
	validID, err := restic.ParseID("c3ab8ff13720e8ad9047dd39466b3c8974e592c2fa383d4a3960714caef0c4f2")
	rtest.OK(t, err)

	dirNodeWithSubtree := &data.Node{
		Name:    "testdir2",
		Type:    data.NodeTypeDir,
		Subtree: &validID,
	}

	bs := repo.NewAssociatedBlobSet()
	addBlobs(bs, dirNodeWithSubtree)

	// Verify the blob was added
	if bs.Len() != 1 {
		t.Errorf("expected 1 blob, got %d", bs.Len())
	}
}
