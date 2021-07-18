package ff

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

func EnvParser(r io.Reader, set func(name, value string) error) error {
	s := bufio.NewScanner(r)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" {
			continue // skip empties
		}

		if line[0] == '#' {
			continue // skip comments
		}

		index := strings.IndexRune(line, '=')
		if index < 0 {
			return fmt.Errorf("invalid line: %s", line)
		}

		var (
			name  = strings.TrimSpace(line[:index])
			value = strings.TrimSpace(line[index+1:])
		)

		if len(name) <= 0 {
			return fmt.Errorf("invalid line: %s", line)
		}

		if len(value) <= 0 {
			return fmt.Errorf("invalid line: %s", line)
		}

		if i := strings.Index(value, " #"); i >= 0 {
			value = strings.TrimSpace(value[:i])
		}

		if unquoted, err := strconv.Unquote(value); err == nil {
			value = unquoted
		}

		if err := set(name, value); err != nil {
			return err
		}
	}
	return nil
}
