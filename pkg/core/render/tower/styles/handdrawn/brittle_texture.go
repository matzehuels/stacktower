package handdrawn

import (
	_ "embed"
	"sync"
)

//go:embed brittle-bg-base64.txt
var brittleTextureBase64 string

var (
	brittleDataURIOnce sync.Once
	brittleDataURI     string
)

func getBrittleTextureDataURI() string {
	brittleDataURIOnce.Do(func() {
		brittleDataURI = "data:image/png;base64," + brittleTextureBase64
	})
	return brittleDataURI
}
