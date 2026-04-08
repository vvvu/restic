package main

import (
	"testing"

	"github.com/restic/restic/internal/data"
	"github.com/restic/restic/internal/repository"
	"github.com/restic/restic/internal/restic"
)

// TestAddBlobsNilSubtree tests that addBlobs handles directory nodes with nil Subtree gracefully.
// This is a regression test for a bug where addBlobs dereferenced node.Subtree without checking
// if it was nil first, which could cause a panic on corrupted repository data.
func TestAddBlobsNilSubtree(t *testing.T) {
	repo := repository.TestRepository(t)

	// Test directory node with nil Subtree - should not panic
	dirNode := &data.Node{
		Name:    "testdir",
		Type:    data.NodeTypeDir,
		Subtree: nil, // explicitly nil to test the fix
	}

	// This should not panic
	bs := repo.NewAssociatedBlobSet()
	addBlobs(bs, dirNode)

	// Verify no blobs were added for the nil subtree by counting keys
	count := 0
	for range bs.Keys() {
		count++
	}
	if count != 0 {
		t.Errorf("addBlobs should not add blobs for nil subtree, got %d blobs", count)
	}

	// Test with valid subtree
	validSubtree, err := restic.ParseID("d8cc1659d8cc1659d8cc1659d8cc1659d8cc1659d8cc1659d8cc1659d8cc1659")
	if err != nil {
		t.Fatalf("ParseID failed: %v", err)
	}
	dirNodeWithSubtree := &data.Node{
		Name:    "testdir2",
		Type:    data.NodeTypeDir,
		Subtree: &validSubtree,
	}

	addBlobs(bs, dirNodeWithSubtree)

	// Verify the blob was added
	count = 0
	for range bs.Keys() {
		count++
	}
	if count != 1 {
		t.Errorf("addBlobs should add one blob for valid subtree, got %d blobs", count)
	}
}

// TestAddBlobsNilNode tests that addBlobs handles nil nodes gracefully.
func TestAddBlobsNilNode(t *testing.T) {
	repo := repository.TestRepository(t)

	// This should not panic
	bs := repo.NewAssociatedBlobSet()
	addBlobs(bs, nil)

	count := 0
	for range bs.Keys() {
		count++
	}
	if count != 0 {
		t.Errorf("addBlobs should not add blobs for nil node, got %d blobs", count)
	}
}

// TestAddBlobsFileNode tests that addBlobs correctly handles file nodes.
func TestAddBlobsFileNode(t *testing.T) {
	repo := repository.TestRepository(t)

	blobID, err := restic.ParseID("6e5f7f696e5f7f696e5f7f696e5f7f696e5f7f696e5f7f696e5f7f696e5f7f69")
	if err != nil {
		t.Fatalf("ParseID failed: %v", err)
	}
	fileNode := &data.Node{
		Name:    "testfile",
		Type:    data.NodeTypeFile,
		Content: restic.IDs{blobID},
	}

	bs := repo.NewAssociatedBlobSet()
	addBlobs(bs, fileNode)

	// Verify the blob was added
	count := 0
	for range bs.Keys() {
		count++
	}
	if count != 1 {
		t.Errorf("addBlobs should add one blob for file node, got %d blobs", count)
	}

	h := restic.BlobHandle{ID: blobID, Type: restic.DataBlob}
	if !bs.Has(h) {
		t.Errorf("addBlobs should add the file's blob")
	}
}

