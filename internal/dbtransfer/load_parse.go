package dbtransfer

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

// readHeader reads the leading comments of a dump and returns the myCart header
// among them. Anything else at the top of the file is not a dump.
func readHeader(in *bufio.Reader) (Header, error) {
	for range 20 {
		line, err := in.ReadString('\n')
		text := strings.TrimSpace(line)
		if text == "" {
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return Header{}, fmt.Errorf("read the dump: %w", err)
			}
			continue
		}
		if !strings.HasPrefix(text, "--") {
			return Header{}, fmt.Errorf("this is not a myCart dump: it starts with %s", quoteLine(text))
		}

		var header Header
		if parseCommentJSON(text, &header) == nil && header.Magic == Magic {
			return header, nil
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return Header{}, fmt.Errorf("read the dump: %w", err)
		}
	}
	return Header{}, errors.New("this is not a myCart dump: it has no header")
}

// parseTrailer reads a dump's closing summary, if the comment is one.
func parseTrailer(line string) (*Trailer, bool) {
	var trailer Trailer
	if err := parseCommentJSON(line, &trailer); err != nil || trailer.Magic != MagicTrailer {
		return nil, false
	}
	return &trailer, true
}

// parseCopyHeader reads the `COPY "table" ("c1", "c2") FROM stdin;` line that
// starts a section.
func parseCopyHeader(line string) (table, error) {
	rest, ok := cutPrefixFold(strings.TrimSpace(line), "COPY")
	if !ok {
		return table{}, fmt.Errorf("this is not a COPY section: %s", quoteLine(line))
	}

	name, rest, err := readQuotedIdent(strings.TrimSpace(rest))
	if err != nil {
		return table{}, err
	}

	rest, ok = strings.CutPrefix(strings.TrimSpace(rest), "(")
	if !ok {
		return table{}, fmt.Errorf("the COPY section of %s has no column list", name)
	}

	columns, rest, err := readColumnList(rest, name)
	if err != nil {
		return table{}, err
	}

	if err := checkCopyTail(rest, name); err != nil {
		return table{}, err
	}

	return table{Name: name, Columns: columns}, nil
}

// readQuotedIdent reads a double-quoted identifier and returns it with the rest
// of the line. The dump quotes every identifier, so anything else is a file
// this package did not write.
func readQuotedIdent(s string) (string, string, error) {
	if !strings.HasPrefix(s, `"`) {
		return "", s, fmt.Errorf("expected a quoted name, got %s", quoteLine(s))
	}

	var name strings.Builder
	for i := 1; i < len(s); i++ {
		if s[i] != '"' {
			name.WriteByte(s[i])
			continue
		}
		if i+1 < len(s) && s[i+1] == '"' {
			name.WriteByte('"')
			i++
			continue
		}
		return name.String(), s[i+1:], nil
	}
	return "", s, fmt.Errorf("unterminated name in %s", quoteLine(s))
}

// cutPrefixFold is strings.CutPrefix without the case sensitivity: SQL keywords
// are written in any case by anything that is not this package.
func cutPrefixFold(s, prefix string) (string, bool) {
	if hasPrefixFold(s, prefix) {
		return s[len(prefix):], true
	}
	return s, false
}

func hasPrefixFold(s, prefix string) bool {
	return len(s) >= len(prefix) && strings.EqualFold(s[:len(prefix)], prefix)
}

// quoteLine renders a line for an error message without dumping a whole file
// into the terminal.
func quoteLine(line string) string {
	const limit = 60
	if len(line) > limit {
		return fmt.Sprintf("%q...", line[:limit])
	}
	return fmt.Sprintf("%q", line)
}

// readColumnList reads the parenthesised column list of a COPY header, after its
// opening bracket, and returns the columns with the rest of the line.
func readColumnList(rest, name string) ([]string, string, error) {
	var columns []string
	closed := false
	for !closed {
		column, next, err := readQuotedIdent(strings.TrimSpace(rest))
		if err != nil {
			return nil, rest, err
		}
		columns = append(columns, column)

		rest = strings.TrimSpace(next)
		switch {
		case strings.HasPrefix(rest, ","):
			rest = rest[1:]
		case strings.HasPrefix(rest, ")"):
			rest = rest[1:]
			closed = true
		default:
			return nil, rest, fmt.Errorf("the COPY section of %s has a malformed column list", name)
		}
	}
	return columns, rest, nil
}

// checkCopyTail checks that a COPY header ends with `FROM stdin;` once its
// column list is read.
func checkCopyTail(rest, name string) error {
	rest, ok := cutPrefixFold(strings.TrimSpace(rest), "FROM")
	if !ok {
		return fmt.Errorf("the COPY section of %s does not say FROM stdin", name)
	}
	rest, ok = cutPrefixFold(strings.TrimSpace(rest), "STDIN")
	if !ok {
		return fmt.Errorf("the COPY section of %s does not say FROM stdin", name)
	}
	if rest = strings.TrimSpace(rest); rest != ";" {
		return fmt.Errorf("the COPY section of %s ends with %s", name, quoteLine(rest))
	}
	return nil
}
