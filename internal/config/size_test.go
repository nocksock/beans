package config

import "testing"

func TestIsValidSize(t *testing.T) {
	cfg := Default()
	tests := []struct {
		size string
		want bool
	}{
		{"", true},
		{"xs", true},
		{"sm", true},
		{"md", true},
		{"lg", true},
		{"xl", true},
		{"xxl", false},
		{"MD", false},
	}
	for _, tt := range tests {
		t.Run(tt.size, func(t *testing.T) {
			if got := cfg.IsValidSize(tt.size); got != tt.want {
				t.Errorf("IsValidSize(%q) = %v, want %v", tt.size, got, tt.want)
			}
		})
	}
}

func TestSizeNames(t *testing.T) {
	cfg := Default()
	want := []string{"xs", "sm", "md", "lg", "xl"}
	got := cfg.SizeNames()
	if len(got) != len(want) {
		t.Fatalf("len(SizeNames()) = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("SizeNames()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
	if list := cfg.SizeList(); list != "xs, sm, md, lg, xl" {
		t.Errorf("SizeList() = %q", list)
	}
}

func TestGetSize(t *testing.T) {
	cfg := Default()
	tests := []struct {
		name string
		want string // expected SizeConfig.Name, "" for nil
	}{
		{"", "md"},
		{"xs", "xs"},
		{"xl", "xl"},
		{"huge", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cfg.GetSize(tt.name)
			if tt.want == "" {
				if got != nil {
					t.Errorf("GetSize(%q) = %v, want nil", tt.name, got)
				}
				return
			}
			if got == nil || got.Name != tt.want {
				t.Errorf("GetSize(%q) = %v, want %q", tt.name, got, tt.want)
			}
		})
	}
}
