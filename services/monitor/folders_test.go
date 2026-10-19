package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetHomeSubFoldersWithDev(t *testing.T) {
	// Create temporary directory simulating /home/user
	tmpDir := t.TempDir()
	userDir := filepath.Join(tmpDir, "testuser")
	devDir := filepath.Join(userDir, "dev")
	proj1 := filepath.Join(devDir, "project1")
	proj2 := filepath.Join(devDir, "project2")

	if err := os.MkdirAll(proj1, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(proj2, 0755); err != nil {
		t.Fatal(err)
	}

	// Create dummy files (> 10MB so they pass the size threshold)
	// We'll write small sparse or 11MB files
	f1, err := os.Create(filepath.Join(proj1, "data.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if err := f1.Truncate(12 * 1024 * 1024); err != nil {
		f1.Close()
		t.Fatal(err)
	}
	f1.Close()

	subs := getHomeSubFolders(tmpDir)
	var foundDev bool
	for _, s := range subs {
		if s.Path == "dev" {
			foundDev = true
			if len(s.Subs) == 0 {
				t.Fatalf("expected dev subfolder to have populated Subs, got 0")
			}
			var foundProj1 bool
			for _, sub := range s.Subs {
				if sub.Path == "project1" {
					foundProj1 = true
				}
			}
			if !foundProj1 {
				t.Errorf("expected to find project1 in dev subs")
			}
		}
	}
	if !foundDev {
		t.Errorf("expected to find dev folder in subs")
	}
}
