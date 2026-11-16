package main

import "testing"

func TestCheckIdempotencyKey(t *testing.T) {
	if checkIdempotencyKey(1, "conv-a", "k1") {
		t.Fatal("first use of a key must not be a duplicate")
	}
	if !checkIdempotencyKey(1, "conv-a", "k1") {
		t.Fatal("second use of the same key must be a duplicate")
	}
	if checkIdempotencyKey(1, "conv-a", "k2") || checkIdempotencyKey(2, "conv-a", "k1") || checkIdempotencyKey(1, "conv-b", "k1") {
		t.Fatal("different key, user or conversation must not collide")
	}
}
