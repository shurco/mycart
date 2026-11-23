package dbtransfer

import (
	"bufio"
	"bytes"
	"errors"
	"io"
)

// copySection streams one table's rows out of a dump, stopping before the \.
// terminator, and counts them.
//
// A dump holds one row per line — COPY text escapes the newlines inside a value
// — so the terminator is a line of its own that no row can produce, and the row
// count is simply the number of lines handed on.
type copySection struct {
	br      *bufio.Reader
	pending []byte
	rows    int64
	done    bool
}

func (s *copySection) Read(p []byte) (int, error) {
	for len(s.pending) == 0 {
		if s.done {
			return 0, io.EOF
		}

		line, err := s.br.ReadBytes('\n')
		if len(line) == 0 {
			if errors.Is(err, io.EOF) {
				return 0, errors.New("the dump ends inside a table: the terminator is missing")
			}
			if err != nil {
				return 0, err
			}
			continue
		}

		if isTerminator(line) {
			s.done = true
			return 0, io.EOF
		}

		s.rows++
		s.pending = line
	}

	n := copy(p, s.pending)
	s.pending = s.pending[n:]
	return n, nil
}

// isTerminator reports whether a line is the \. that ends a COPY section.
func isTerminator(line []byte) bool {
	return bytes.Equal(bytes.TrimRight(line, "\r\n"), []byte(copyTerminal))
}
