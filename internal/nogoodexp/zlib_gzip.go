//go:build cgo

package nogoodexp

/*
#cgo LDFLAGS: -lz
#include <stdlib.h>
#include <zlib.h>

static int nous_raw_deflate(const unsigned char *src, size_t src_len,
                            unsigned char *dst, size_t dst_cap,
                            size_t *dst_len) {
    z_stream stream = {0};
    int rc = deflateInit2(&stream, 9, Z_DEFLATED, -15, 8, Z_DEFAULT_STRATEGY);
    if (rc != Z_OK) return rc;
    stream.next_in = (Bytef *)src;
    stream.avail_in = (uInt)src_len;
    stream.next_out = dst;
    stream.avail_out = (uInt)dst_cap;
    rc = deflate(&stream, Z_FINISH);
    if (rc == Z_STREAM_END) {
        *dst_len = stream.total_out;
        rc = Z_OK;
    }
    deflateEnd(&stream);
    return rc;
}
*/
import "C"

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"unsafe"
)

// canonicalGzip uses the zlib level-9 DEFLATE stream fixed by ngt/v1. Go's
// standard-library flate encoder is deterministic but byte-different from the
// preregistered golden zlib stream.
func canonicalGzip(raw []byte) ([]byte, error) {
	bound := int(C.compressBound(C.uLong(len(raw))))
	deflated := make([]byte, bound)
	var outputLength C.size_t
	var sourcePointer *C.uchar
	if len(raw) > 0 {
		sourcePointer = (*C.uchar)(unsafe.Pointer(&raw[0]))
	}
	rc := C.nous_raw_deflate(sourcePointer, C.size_t(len(raw)), (*C.uchar)(unsafe.Pointer(&deflated[0])), C.size_t(len(deflated)), &outputLength)
	if rc != C.Z_OK {
		return nil, fmt.Errorf("zlib raw deflate failed: %d", int(rc))
	}
	deflated = deflated[:int(outputLength)]
	output := make([]byte, 10+len(deflated)+8)
	copy(output[:10], []byte{0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff})
	copy(output[10:], deflated)
	binary.LittleEndian.PutUint32(output[len(output)-8:len(output)-4], crc32.ChecksumIEEE(raw))
	binary.LittleEndian.PutUint32(output[len(output)-4:], uint32(len(raw)))
	return output, nil
}
