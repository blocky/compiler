package container_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/blocky/compiler/internal/container"
)

func writeBytes(dst *bytes.Buffer, kind byte, b []byte) error {
	header := make([]byte, 8)
	header[0] = kind
	binary.BigEndian.PutUint32(header[4:], uint32(len(b)))
	if _, err := dst.Write(header[:]); err != nil {
		return err
	}
	if _, err := dst.Write(b); err != nil {
		return err
	}
	return nil
}

func TestSplitOutput(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		// given
		wantOutput := &bytes.Buffer{}

		wantStdOut := []byte("output from stdout")
		err := writeBytes(wantOutput, container.StdOut, wantStdOut)
		require.NoError(t, err)

		wantStdErr := []byte("output from stderr")
		err = writeBytes(wantOutput, container.StdErr, wantStdErr)
		require.NoError(t, err)

		// when
		gotStdOut, gotStdErr, err := container.SplitOutput(wantOutput)

		// then
		require.NoError(t, err)
		assert.Equal(t, wantStdOut, gotStdOut)
		assert.Equal(t, wantStdErr, gotStdErr)
	})

	t.Run("happy path - unknown stream types are ignored", func(t *testing.T) {
		// given
		wantOutput := &bytes.Buffer{}

		wantStdOut := []byte("output from stdout")
		err := writeBytes(wantOutput, container.StdOut, wantStdOut)
		require.NoError(t, err)

		wantUnknown := []byte("output from unknown source")
		err = writeBytes(wantOutput, byte(3), wantUnknown)
		require.NoError(t, err)

		// when
		gotStdOut, gotStdErr, err := container.SplitOutput(wantOutput)

		// then
		require.NoError(t, err)
		assert.Equal(t, wantStdOut, gotStdOut)
		assert.Empty(t, gotStdErr)
	})

	t.Run("error reading header", func(t *testing.T) {
		// given
		wantOutput := &bytes.Buffer{}
		wantOutput.Write([]byte{1, 2, 3})

		// when
		_, _, err := container.SplitOutput(wantOutput)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "reading header")
	})

	t.Run("error reading payload", func(t *testing.T) {
		// given
		wantOutput := &bytes.Buffer{}
		payload := []byte{4, 5, 6}
		header := make([]byte, 8)
		header[0] = byte(1)
		binary.BigEndian.PutUint32(header[4:], uint32(len(payload)+1))
		wantOutput.Write(header)
		// when
		_, _, err := container.SplitOutput(wantOutput)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "reading payload")
	})
}
