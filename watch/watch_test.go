package watch

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/qwenode/sohot/boot"
)

func TestIsExcluded(t *testing.T) {
	original := boot.V.Watch.Exclude
	t.Cleanup(func() { boot.V.Watch.Exclude = original })

	tests := []struct {
		path    string
		exclude []string
		want    bool
	}{
		{"tmp", []string{"tmp/"}, true},
		{`tmp\app.exe.tmp`, []string{"tmp/"}, true},
		{"main.go", []string{"tmp/"}, false},
		{"mytmp/a.go", []string{"tmp/"}, true},
		{"job.executor.go", []string{}, false},
		{".github/workflows/release.yml", []string{}, false},
		{".git/objects/ab", []string{}, true},
		{"sub/.idea/workspace.xml", []string{}, true},
		{"bin/app.exe", []string{}, true},
		{"watch/watch_test.go", []string{"_test.go"}, true},
		{"main.go", []string{""}, false},
	}
	for _, tt := range tests {
		boot.V.Watch.Exclude = tt.exclude
		if got := isExcluded(tt.path); got != tt.want {
			t.Errorf("isExcluded(%q) with exclude %q = %v, want %v", tt.path, tt.exclude, got, tt.want)
		}
	}
}

func TestIsBuildOutputAndTarget(t *testing.T) {
	original := boot.V.Build.Name
	t.Cleanup(func() { boot.V.Build.Name = original })

	app, err := filepath.Abs("app")
	if err != nil {
		t.Fatal(err)
	}
	binApp, err := filepath.Abs("bin/app.exe")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		path       string
		wantOutput bool
		wantTarget bool
	}{
		{app, "app", true, true},
		{app, "app.tmp", true, false},
		{app, "app.delete_me_20260917120000", true, false},
		{app, "app.go", false, false},
		{app, "app.tmpl", false, false},
		{binApp, "bin/APP.EXE", true, true},
		{binApp, "bin/app.log", false, false},
		{"", "app", false, false},
	}
	for _, tt := range tests {
		boot.V.Build.Name = tt.name
		if got := isBuildOutput(tt.path); got != tt.wantOutput {
			t.Errorf("isBuildOutput(%q) with build.name %q = %v, want %v", tt.path, tt.name, got, tt.wantOutput)
		}
		if got := isBuildTarget(tt.path); got != tt.wantTarget {
			t.Errorf("isBuildTarget(%q) with build.name %q = %v, want %v", tt.path, tt.name, got, tt.wantTarget)
		}
	}
}

func TestReplaceExecutable(t *testing.T) {
	tests := []struct {
		desc    string
		old     string
		new     string
		wantErr bool
		want    string
	}{
		{"replaces existing executable", "old", "new", false, "new"},
		{"creates missing executable", "", "new", false, "new"},
		{"restores old executable when new one is missing", "old", "", true, "old"},
	}
	for _, tt := range tests {
		dir := t.TempDir()
		name := filepath.Join(dir, "app")
		tempFile := name + ".tmp"
		if tt.old != "" {
			if err := os.WriteFile(name, []byte(tt.old), 0o755); err != nil {
				t.Fatal(err)
			}
		}
		if tt.new != "" {
			if err := os.WriteFile(tempFile, []byte(tt.new), 0o755); err != nil {
				t.Fatal(err)
			}
		}

		err := replaceExecutable(tempFile, name)
		if (err != nil) != tt.wantErr {
			t.Errorf("%s: replaceExecutable() error = %v, wantErr %v", tt.desc, err, tt.wantErr)
		}
		got, err := os.ReadFile(name)
		if err != nil {
			t.Errorf("%s: read %s: %v", tt.desc, name, err)
		} else if string(got) != tt.want {
			t.Errorf("%s: %s contains %q, want %q", tt.desc, name, got, tt.want)
		}
		if leftovers, _ := filepath.Glob(name + ".delete_me_*"); len(leftovers) != 0 {
			t.Errorf("%s: leftover files %v", tt.desc, leftovers)
		}
	}
}
