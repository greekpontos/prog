package model

import "testing"

func TestPermissions(t *testing.T) {
	post := &Post{ID: 1, AuthorID: 1}
	if !post.CanBeEditedBy(1) || post.CanBeEditedBy(2) {
		t.Fatal("unexpected post edit permission result")
	}
	if !post.CanBeDeletedBy(1) || post.CanBeDeletedBy(2) {
		t.Fatal("unexpected post delete permission result")
	}

	comment := &Comment{ID: 1, AuthorID: 1}
	if !comment.CanBeEditedBy(1) || comment.CanBeEditedBy(2) {
		t.Fatal("unexpected comment edit permission result")
	}
	if !comment.CanBeDeletedBy(1) || comment.CanBeDeletedBy(2) {
		t.Fatal("unexpected comment delete permission result")
	}
}
