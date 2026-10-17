package cmd

import (
	"testing"

	"github.com/hmans/beans/internal/config"
)

func TestArchiveCmd(t *testing.T) {
	testCore, cleanup := setupQueryTestCore(t)
	defer cleanup()

	oldCfg := cfg
	cfg = config.Default()
	defer func() { cfg = oldCfg }()

	createQueryTestBean(t, testCore, "arc-todo", "Open", "todo")
	createQueryTestBean(t, testCore, "arc-done", "Done", "completed")
	createQueryTestBean(t, testCore, "arc-scrap", "Scrapped", "scrapped")
	createQueryTestBean(t, testCore, "arc-old", "Already archived", "archived")

	// A non-archived bean linking to an archived one keeps its references
	child := createQueryTestBean(t, testCore, "arc-child", "Child", "todo")
	child.Parent = "arc-done"
	child.Blocking = []string{"arc-scrap"}
	if err := testCore.Update(child); err != nil {
		t.Fatalf("failed to update child: %v", err)
	}

	oldForce := archiveForce
	archiveForce = true
	defer func() { archiveForce = oldForce }()

	if err := archiveCmd.RunE(archiveCmd, nil); err != nil {
		t.Fatalf("archive error = %v", err)
	}

	tests := []struct {
		id     string
		status string
	}{
		{"arc-todo", "todo"},
		{"arc-done", "archived"},
		{"arc-scrap", "archived"},
		{"arc-old", "archived"},
		{"arc-child", "todo"},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			b, err := testCore.Get(tt.id)
			if err != nil {
				t.Fatalf("bean %s missing after archive: %v", tt.id, err)
			}
			if b.Status != tt.status {
				t.Errorf("status = %q, want %q", b.Status, tt.status)
			}
		})
	}

	got, err := testCore.Get("arc-child")
	if err != nil {
		t.Fatalf("child missing: %v", err)
	}
	if got.Parent != "arc-done" {
		t.Errorf("child parent = %q, want %q", got.Parent, "arc-done")
	}
	if len(got.Blocking) != 1 || got.Blocking[0] != "arc-scrap" {
		t.Errorf("child blocking = %v, want [arc-scrap]", got.Blocking)
	}
}
