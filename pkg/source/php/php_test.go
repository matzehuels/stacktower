package php

import (
	"testing"
	"time"

	"github.com/matzehuels/stacktower/pkg/integrations/packagist"
)

func TestNewParser(t *testing.T) {
	p, err := NewParser(time.Minute)
	if err != nil {
		t.Fatalf("NewParser error: %v", err)
	}
	if p == nil || p.client == nil {
		t.Fatalf("parser or client is nil")
	}
}

func TestPackageInfo_Getters(t *testing.T) {
	pi := &packageInfo{&packagist.PackageInfo{
		Name:         "vendor/pkg",
		Version:      "1.0.0",
		Dependencies: []string{"vendor/dep"},
	}}

	if pi.GetName() != "vendor/pkg" {
		t.Errorf("GetName = %s", pi.GetName())
	}
	if pi.GetVersion() != "1.0.0" {
		t.Errorf("GetVersion = %s", pi.GetVersion())
	}
	deps := pi.GetDependencies()
	if len(deps) != 1 || deps[0] != "vendor/dep" {
		t.Errorf("GetDependencies = %#v", deps)
	}
}

func TestPackageInfo_ToMetadata(t *testing.T) {
	pi := &packageInfo{&packagist.PackageInfo{
		Version:     "2.3.4",
		Description: "desc",
		License:     "MIT",
		Author:      "Jane",
	}}

	m := pi.ToMetadata()
	if m["version"].(string) != "2.3.4" {
		t.Errorf("version missing or wrong: %#v", m)
	}
	if m["description"].(string) != "desc" {
		t.Errorf("description wrong: %#v", m)
	}
	if m["license"].(string) != "MIT" {
		t.Errorf("license wrong: %#v", m)
	}
	if m["author"].(string) != "Jane" {
		t.Errorf("author wrong: %#v", m)
	}
}

func TestPackageInfo_ToRepoInfo(t *testing.T) {
	pi := &packageInfo{&packagist.PackageInfo{
		Name:       "vendor/pkg",
		Version:    "0.1.0",
		Repository: "https://github.com/user/repo",
		HomePage:   "https://example.com",
	}}

	ri := pi.ToRepoInfo()
	if ri.Name != "vendor/pkg" || ri.Version != "0.1.0" {
		t.Errorf("unexpected name/version: %#v", ri)
	}
	if ri.ManifestFile != "composer.json" {
		t.Errorf("unexpected manifest: %s", ri.ManifestFile)
	}
	if ri.ProjectURLs["repository"] != "https://github.com/user/repo" {
		t.Errorf("repo url missing: %#v", ri.ProjectURLs)
	}
	if ri.ProjectURLs["homepage"] != "https://example.com" {
		t.Errorf("homepage url missing: %#v", ri.ProjectURLs)
	}
	if ri.HomePage != "https://example.com" {
		t.Errorf("homepage field wrong: %s", ri.HomePage)
	}
}
