//go:build !cgo

package nogoodexp

import "fmt"

func canonicalGzip([]byte) ([]byte, error) {
	return nil, fmt.Errorf("ngt/v1 canonical zlib compression requires cgo")
}
