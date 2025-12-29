package main

import (
	"bufio"
	"io"
	"strings"
)

// Record represents a single entry in the lsregister dump
type Record struct {
	Raw        string `json:"raw,omitempty"`
	BundleID   string `json:"bundle_id"`
	Path       string `json:"path"`
	Name       string `json:"name"`
	Container  string `json:"container,omitempty"`
	Class      string `json:"class,omitempty"`
	Type       string `json:"type,omitempty"`
	Identifier string `json:"identifier,omitempty"`
}

// ParseDump parses the output of lsregister -dump
func ParseDump(r io.Reader) ([]Record, error) {
	scanner := bufio.NewScanner(r)
	var records []Record
	var current lines

	// separator is the long line of dashes
	const separator = "--------------------------------------------------------------------------------"

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == separator {
			if len(current) > 0 {
				if rec := parseRecord(current); rec != nil {
					records = append(records, *rec)
				}
				current = nil
			}
			continue
		}
		current = append(current, line)
	}

	// Flush last record
	if len(current) > 0 {
		if rec := parseRecord(current); rec != nil {
			records = append(records, *rec)
		}
	}

	return records, scanner.Err()
}

type lines []string

func parseRecord(ls lines) *Record {
	if len(ls) == 0 {
		return nil
	}

	r := &Record{
		Raw: strings.Join(ls, "\n"),
	}

	// Simple extraction of key fields
	// Fields are typically "key: value" with variable whitespace
	// Note: values can be quoted or formatted.

	for _, line := range ls {
		line = strings.TrimSpace(line)
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "bundle id":
			// value might have suffix like "(0x1234)"
			if idx := strings.LastIndex(val, " (0x"); idx != -1 {
				r.BundleID = strings.TrimSpace(val[:idx])
			} else {
				r.BundleID = val
			}
		case "path":
			// path usually has suffix like "(0x1234)"
			if idx := strings.LastIndex(val, " (0x"); idx != -1 {
				r.Path = strings.TrimSpace(val[:idx])
			} else {
				r.Path = val
			}
		case "name":
			r.Name = val
		case "container":
			if idx := strings.LastIndex(val, " (0x"); idx != -1 {
				r.Container = strings.TrimSpace(val[:idx])
			} else {
				r.Container = val
			}
		case "class":
			r.Class = val
		case "type code":
			r.Type = val
		case "identifier":
			r.Identifier = val
		}
	}

	// Filter out header blocks (which don't have bundle id typically, or distinct format)
	// The dump header starts with "Checking data integrity..." etc.
	// Best check: must have at least one key field?
	if r.BundleID == "" && r.Path == "" {
		return nil
	}

	return r
}
