package wheel

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"unicode/utf8"
)

// writeEntry stores one file with its checksum and sizes known up front. zip.Writer.CreateHeader streams and
// appends a data descriptor after each entry, which PyPI rejects ("ZIP contains a data descriptor").
func writeEntry(w *zip.Writer, f file) error {
	var compressed bytes.Buffer
	deflate, e := flate.NewWriter(&compressed, flate.DefaultCompression)
	if e != nil {
		return e
	}
	if _, e = deflate.Write(f.data); e != nil {
		return e
	}
	if e = deflate.Close(); e != nil {
		return e
	}
	entry, e := w.CreateRaw(&zip.FileHeader{
		Name:   f.name,
		Method: zip.Deflate,
		// CreateRaw writes headers as given; only CreateHeader converts Modified, and it adds a data descriptor.
		//lint:ignore SA1019 the raw MS-DOS date is the only timestamp field CreateRaw honours.
		ModifiedDate:       dosDate,
		CRC32:              crc32.ChecksumIEEE(f.data),
		CompressedSize64:   uint64(compressed.Len()),
		UncompressedSize64: uint64(len(f.data)),
	})
	if e != nil {
		return e
	}
	_, e = entry.Write(compressed.Bytes())
	return e
}

const (
	localHeaderSig    = 0x04034b50
	centralHeaderSig  = 0x02014b50
	dataDescriptorSig = 0x08074b50
	endOfCentralSig   = 0x06054b50
	endOfCentralLen   = 22
)

// Verify checks a wheel against PyPI's upload rules (https://docs.pypi.org/archives): no data descriptors,
// a single central directory listing every entry, no duplicate names, printable UTF-8 names, no duplicate
// extra-field IDs, and no comments or data before or after the archive.
func Verify(path string) error {
	raw, e := os.ReadFile(path)
	if e != nil {
		return e
	}
	fail := func(format string, args ...any) error {
		return fmt.Errorf("%s is not accepted by PyPI: %s", path, fmt.Sprintf(format, args...))
	}
	if len(raw) < endOfCentralLen || binary.LittleEndian.Uint32(raw) != localHeaderSig {
		return fail("data before the first entry")
	}
	// With no archive comment, the end-of-central-directory record is exactly the last 22 bytes.
	end := raw[len(raw)-endOfCentralLen:]
	if binary.LittleEndian.Uint32(end) != endOfCentralSig {
		return fail("trailing data or an archive comment")
	}
	if disk, start := binary.LittleEndian.Uint16(end[4:]), binary.LittleEndian.Uint16(end[6:]); disk != 0 || start != 0 {
		return fail("multi-disk archive")
	}
	entries := int(binary.LittleEndian.Uint16(end[10:]))
	size := int(binary.LittleEndian.Uint32(end[12:]))
	offset := int(binary.LittleEndian.Uint32(end[16:]))
	if offset+size != len(raw)-endOfCentralLen {
		return fail("central directory is not directly before its end record")
	}
	r, e := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if e != nil {
		return fail("%v", e)
	}
	if r.Comment != "" || len(r.File) != entries {
		return fail("archive comment or central directory count mismatch")
	}
	names := map[string]bool{}
	next := 0 // Entries must be stored back to back from the start, with nothing between them.
	for _, f := range r.File {
		if names[f.Name] {
			return fail("duplicate entry %q", f.Name)
		}
		names[f.Name] = true
		if !utf8.ValidString(f.Name) {
			return fail("entry name %q is not UTF-8", f.Name)
		}
		for _, c := range f.Name {
			if c <= 0x20 || c == 0x7f {
				return fail("entry name %q contains an unprintable character", f.Name)
			}
		}
		if f.Comment != "" {
			return fail("entry %q has a comment", f.Name)
		}
		if f.Flags&0x8 != 0 {
			return fail("entry %q uses a data descriptor", f.Name)
		}
		if e = uniqueExtraIDs(f.Extra); e != nil {
			return fail("entry %q: %v", f.Name, e)
		}
		data, e := f.DataOffset()
		if e != nil {
			return fail("entry %q: %v", f.Name, e)
		}
		header, e := localHeaderStart(raw, next)
		if e != nil {
			return fail("entry %q: %v", f.Name, e)
		}
		if flags := binary.LittleEndian.Uint16(raw[header+6:]); flags&0x8 != 0 {
			return fail("entry %q uses a data descriptor", f.Name)
		}
		next = int(data) + int(f.CompressedSize64)
		if next+4 <= len(raw) && binary.LittleEndian.Uint32(raw[next:]) == dataDescriptorSig {
			return fail("entry %q is followed by a data descriptor", f.Name)
		}
		if rc, e := f.Open(); e != nil {
			return fail("entry %q: %v", f.Name, e)
		} else if _, e = io.Copy(io.Discard, rc); e != nil {
			_ = rc.Close()
			return fail("entry %q: %v", f.Name, e) // Also checks the stored CRC-32.
		} else if e = rc.Close(); e != nil {
			return fail("entry %q: %v", f.Name, e)
		}
	}
	if next != offset || binary.LittleEndian.Uint32(raw[offset:]) != centralHeaderSig {
		return fail("data between the last entry and the central directory")
	}
	return nil
}

func localHeaderStart(raw []byte, at int) (int, error) {
	if at+30 > len(raw) || binary.LittleEndian.Uint32(raw[at:]) != localHeaderSig {
		return 0, fmt.Errorf("unexpected data where the next entry should start")
	}
	return at, nil
}

func uniqueExtraIDs(extra []byte) error {
	seen := map[uint16]bool{}
	for len(extra) >= 4 {
		id, size := binary.LittleEndian.Uint16(extra), int(binary.LittleEndian.Uint16(extra[2:]))
		if seen[id] {
			return fmt.Errorf("duplicate extra field 0x%04x", id)
		}
		seen[id] = true
		if 4+size > len(extra) {
			return fmt.Errorf("truncated extra field 0x%04x", id)
		}
		extra = extra[4+size:]
	}
	return nil
}
