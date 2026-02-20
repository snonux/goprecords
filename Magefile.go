//go:build mage

package main

// Magefile for goprecords. Targets follow the same style as other Go projects (e.g. hexai).

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

const binaryName = "goprecords"

// Default builds the binary.
func Default() { mg.Deps(Build) }

// Build builds the goprecords binary.
func Build() error {
	return sh.RunV("go", "build", "-o", binaryName, "./cmd/goprecords")
}

// Test runs all tests.
func Test() error {
	return sh.RunV("go", "test", "./...")
}

// Install builds and installs the binary to GOPATH/bin.
func Install() error {
	mg.Deps(Build)
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("GOPATH unset and home: %w", err)
		}
		gopath = filepath.Join(home, "go")
	}
	binDir := filepath.Join(gopath, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", binDir, err)
	}
	dest := filepath.Join(binDir, binaryName)
	return sh.RunV("cp", "-v", binaryName, dest)
}

// Uninstall removes the binary from GOPATH/bin.
func Uninstall() error {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("GOPATH unset and home: %w", err)
		}
		gopath = filepath.Join(home, "go")
	}
	dest := filepath.Join(gopath, "bin", binaryName)
	if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
