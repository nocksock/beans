package launcher

import (
	"slices"
	"strings"
	"testing"
)

func TestLaunchWorkDir(t *testing.T) {
	tests := []struct {
		name   string
		launch Launch
		want   string
	}{
		{"defaults to project root", Launch{ProjectRoot: "/repo/pkg"}, "/repo/pkg"},
		{"uses config dir when set", Launch{ProjectRoot: "/repo/pkg", ConfigDir: "/repo"}, "/repo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.launch.WorkDir(); got != tt.want {
				t.Errorf("WorkDir() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLaunchEnv(t *testing.T) {
	l := Launch{
		ProjectRoot: "/repo/pkg",
		ConfigDir:   "/repo",
		BeansDir:    "/repo/.beans",
		BeanID:      "pkg-1234",
		BeanTitle:   "Do it",
	}

	env := l.Env()
	for _, want := range []string{
		"BEANS_ROOT=/repo/pkg",
		"BEANS_CONFIG_DIR=/repo",
		"BEANS_DIR=/repo/.beans",
		"BEANS_ID=pkg-1234",
		"BEANS_TASK=Do it",
	} {
		if !slices.Contains(env, want) {
			t.Errorf("Env() missing %q", want)
		}
	}
}

func TestLaunchEnvConfigDirDefaultsToProjectRoot(t *testing.T) {
	env := Launch{ProjectRoot: "/repo/pkg"}.Env()
	if !slices.Contains(env, "BEANS_CONFIG_DIR=/repo/pkg") {
		t.Errorf("Env() = %v, want BEANS_CONFIG_DIR=/repo/pkg", env)
	}
}

func TestCreateExecCommandRunsInWorkDir(t *testing.T) {
	projectRoot := t.TempDir()
	configDir := t.TempDir()

	tests := []struct {
		name   string
		launch Launch
		want   string
	}{
		{"single-line, inherited", Launch{Script: "pwd", ProjectRoot: projectRoot, ConfigDir: configDir}, configDir},
		{"multi-line, inherited", Launch{Script: "#!/bin/sh\npwd", ProjectRoot: projectRoot, ConfigDir: configDir}, configDir},
		{"single-line, local", Launch{Script: "pwd", ProjectRoot: projectRoot}, projectRoot},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, result, err := CreateExecCommand(tt.launch)
			if err != nil {
				t.Fatal(err)
			}
			defer result.Cleanup()

			out, err := cmd.Output()
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.TrimSpace(string(out)); got != tt.want {
				t.Errorf("ran in %q, want %q", got, tt.want)
			}
		})
	}
}
