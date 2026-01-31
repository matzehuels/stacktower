package integrations_test

import (
	"fmt"

	"github.com/matzehuels/stacktower/pkg/integrations"
)

func ExampleNormalizePkgName() {
	// Package names are normalized to lowercase with hyphens
	fmt.Println(integrations.NormalizePkgName("FastAPI"))
	fmt.Println(integrations.NormalizePkgName("my_package"))
	fmt.Println(integrations.NormalizePkgName("  Spaces  "))
	// Output:
	// fastapi
	// my-package
	// spaces
}

func ExampleNormalizeRepoURL() {
	// Various repository URL formats are normalized to HTTPS
	fmt.Println(integrations.NormalizeRepoURL("git@github.com:user/repo.git"))
	fmt.Println(integrations.NormalizeRepoURL("git://github.com/user/repo"))
	fmt.Println(integrations.NormalizeRepoURL("git+https://github.com/user/repo.git"))
	fmt.Println(integrations.NormalizeRepoURL("https://github.com/user/repo"))
	// Output:
	// https://github.com/user/repo
	// https://github.com/user/repo
	// https://github.com/user/repo
	// https://github.com/user/repo
}

func ExampleURLEncode() {
	// URL-encode special characters for API queries
	fmt.Println(integrations.URLEncode("@scope/package"))
	fmt.Println(integrations.URLEncode("package name"))
	// Output:
	// %40scope%2Fpackage
	// package+name
}

func ExampleRepoMetrics() {
	// RepoMetrics holds repository data from GitHub/GitLab
	metrics := integrations.RepoMetrics{
		RepoURL:  "https://github.com/psf/requests",
		Owner:    "psf",
		Stars:    51000,
		Language: "Python",
		Archived: false,
	}

	fmt.Println("Repository:", metrics.RepoURL)
	fmt.Println("Stars:", metrics.Stars)
	fmt.Println("Archived:", metrics.Archived)
	// Output:
	// Repository: https://github.com/psf/requests
	// Stars: 51000
	// Archived: false
}

func ExampleContributor() {
	// Contributors track commit counts for bus factor analysis
	contributors := []integrations.Contributor{
		{Login: "maintainer1", Contributions: 500},
		{Login: "maintainer2", Contributions: 200},
		{Login: "contributor3", Contributions: 50},
	}

	fmt.Println("Top contributor:", contributors[0].Login)
	fmt.Println("Contributions:", contributors[0].Contributions)
	// Output:
	// Top contributor: maintainer1
	// Contributions: 500
}

func Example_errors() {
	// Standard errors for registry operations
	fmt.Println("ErrNotFound:", integrations.ErrNotFound)
	fmt.Println("ErrNetwork:", integrations.ErrNetwork)
	// Output:
	// ErrNotFound: not found
	// ErrNetwork: network error
}
