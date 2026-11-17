package collector

import (
	"encoding/binary"
	"io"
)

// demuxWriter removes the 8-byte frame headers Docker puts before each chunk
// of the log of a container without a TTY (stream type, 3 zero bytes, size).
// Splitting such a stream into lines left the headers inside the text, and a
// header byte equal to a space or a newline broke the "timestamp message"
// format, so the raw timestamp ended up in the message. The log of a
// container with a TTY has no headers and is written as it is.
type demuxWriter struct {
	w      io.Writer
	mode   int // 0 not known yet, 1 framed, 2 raw
	header []byte
	left   uint32 // bytes left in the current frame
}

const (
	demuxUnknown = iota
	demuxFramed
	demuxRaw
)

func newDemuxWriter(w io.Writer) io.Writer { return &demuxWriter{w: w} }

func (d *demuxWriter) Write(p []byte) (int, error) {
	n := len(p)
	if d.mode == demuxUnknown && len(p) > 0 {
		// frames start with the stream type: 0 stdin, 1 stdout, 2 stderr
		if p[0] <= 2 {
			d.mode = demuxFramed
		} else {
			d.mode = demuxRaw
		}
	}
	if d.mode == demuxRaw {
		_, err := d.w.Write(p)
		return n, err
	}
	for len(p) > 0 {
		if d.left == 0 {
			need := 8 - len(d.header)
			if need > len(p) {
				need = len(p)
			}
			d.header = append(d.header, p[:need]...)
			p = p[need:]
			if len(d.header) < 8 {
				break
			}
			d.left = binary.BigEndian.Uint32(d.header[4:])
			d.header = d.header[:0]
			continue
		}
		chunk := p
		if uint32(len(chunk)) > d.left {
			chunk = chunk[:d.left]
		}
		if _, err := d.w.Write(chunk); err != nil {
			return n, err
		}
		d.left -= uint32(len(chunk))
		p = p[len(chunk):]
	}
	return n, nil
}
