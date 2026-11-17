package collector

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func frame(stream byte, s string) []byte {
	h := make([]byte, 8)
	h[0] = stream
	binary.BigEndian.PutUint32(h[4:], uint32(len(s)))
	return append(h, s...)
}

func TestDemuxFramed(t *testing.T) {
	// a 32-byte line puts a space (0x20) in the header
	line := "2026-11-17T08:47:40.688992924Z x\n"
	in := append(frame(1, line), frame(2, "2026-11-17T08:47:41Z err\n")...)
	var out bytes.Buffer
	w := newDemuxWriter(&out)
	// write in small pieces to cross header and frame boundaries
	for i := 0; i < len(in); i += 5 {
		end := i + 5
		if end > len(in) {
			end = len(in)
		}
		if _, err := w.Write(in[i:end]); err != nil {
			t.Fatal(err)
		}
	}
	want := line + "2026-11-17T08:47:41Z err\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestDemuxRaw(t *testing.T) {
	var out bytes.Buffer
	w := newDemuxWriter(&out)
	in := "2026-11-17T08:47:41Z tty line\n"
	if _, err := w.Write([]byte(in)); err != nil {
		t.Fatal(err)
	}
	if out.String() != in {
		t.Fatalf("got %q", out.String())
	}
}
