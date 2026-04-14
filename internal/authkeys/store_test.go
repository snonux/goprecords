package authkeys

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenStore_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	path := filepath.Join(t.TempDir(), "auth.db")
	_, err := OpenStore(ctx, path)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestCreateVerifyReplace(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "auth.db")
	s, err := OpenStore(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.EnsureSchema(ctx); err != nil {
		t.Fatal(err)
	}
	n, err := s.KeyCount(ctx)
	if err != nil || n != 0 {
		t.Fatalf("KeyCount got %d err %v", n, err)
	}
	tok1, err := s.CreateKey(ctx, "host-a")
	if err != nil {
		t.Fatal(err)
	}
	if tok1 == "" {
		t.Fatal("empty token")
	}
	n, err = s.KeyCount(ctx)
	if err != nil || n != 1 {
		t.Fatalf("KeyCount after create got %d err %v", n, err)
	}
	ok, err := s.Verify(ctx, "host-a", tok1)
	if err != nil || !ok {
		t.Fatalf("Verify tok1 got %v ok=%v", err, ok)
	}
	ok, err = s.Verify(ctx, "host-a", "wrong")
	if err != nil || ok {
		t.Fatalf("Verify wrong got %v ok=%v", err, ok)
	}
	tok2, err := s.CreateKey(ctx, "host-a")
	if err != nil {
		t.Fatal(err)
	}
	if tok2 == tok1 {
		t.Fatal("expected new token after replace")
	}
	ok, err = s.Verify(ctx, "host-a", tok1)
	if err != nil || ok {
		t.Fatalf("old token should fail got %v ok=%v", err, ok)
	}
	ok, err = s.Verify(ctx, "host-a", tok2)
	if err != nil || !ok {
		t.Fatalf("new token should work got %v ok=%v", err, ok)
	}
}

func TestDefaultPath(t *testing.T) {
	p := DefaultPath("/var/stats")
	if filepath.Base(p) != "goprecords-auth.db" {
		t.Fatalf("got %q", p)
	}
}

func TestCloseNilStore(t *testing.T) {
	var s *Store
	if err := s.Close(); err != nil {
		t.Fatalf("Close nil: %v", err)
	}
}

func TestCloseNilDB(t *testing.T) {
	s := &Store{}
	if err := s.Close(); err != nil {
		t.Fatalf("Close nil db: %v", err)
	}
}

func TestCreateKeyEmptyHostname(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "auth.db")
	s, err := OpenStore(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.EnsureSchema(ctx); err != nil {
		t.Fatal(err)
	}
	_, err = s.CreateKey(ctx, "")
	if err == nil || !strings.Contains(err.Error(), "hostname") {
		t.Fatalf("expected empty hostname error, got %v", err)
	}
}

func TestVerifyUnknownHost(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "auth.db")
	s, err := OpenStore(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.EnsureSchema(ctx); err != nil {
		t.Fatal(err)
	}
	ok, err := s.Verify(ctx, "nohost", "any")
	if err != nil || ok {
		t.Fatalf("Verify unknown host: ok=%v err=%v", ok, err)
	}
}

func TestOpsAfterClose(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "auth.db")
	s, err := OpenStore(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.EnsureSchema(ctx); err != nil {
		s.Close()
		t.Fatal(err)
	}
	if _, err := s.CreateKey(ctx, "h"); err != nil {
		s.Close()
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	_, err = s.KeyCount(ctx)
	if err == nil {
		t.Fatal("KeyCount after close expected error")
	}
	_, err = s.Verify(ctx, "h", "x")
	if err == nil {
		t.Fatal("Verify after close expected error")
	}
}
