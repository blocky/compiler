package container

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

const (
	StdOut byte = 1
	StdErr byte = 2
)

func readHeader(r io.Reader) (*header, error) {
	h := make([]byte, 8)
	if _, err := io.ReadFull(r, h); err != nil {
		return nil, err
	}
	var out header
	copy(out[:], h)
	return &out, nil
}

type header [8]byte

func (h *header) StreamType() byte {
	return h[0]
}

func (h *header) PayloadLen() uint32 {
	return binary.BigEndian.Uint32(h[4:8])
}

func SplitOutput(src io.Reader) ([]byte, []byte, error) {
	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}

	for {
		h, err := readHeader(src)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("reading header: %w", err)
		}

		if h.PayloadLen() == 0 {
			continue
		}

		payload := make([]byte, h.PayloadLen())
		if _, err = io.ReadFull(src, payload); err != nil {
			return nil, nil, fmt.Errorf("reading payload: %w", err)
		}

		switch h.StreamType() {
		case StdOut:
			outBuf.Write(payload)
		case StdErr:
			errBuf.Write(payload)
		}
	}

	return outBuf.Bytes(), errBuf.Bytes(), nil
}
