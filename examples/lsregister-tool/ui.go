package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

func printJSON(records []Record) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(records); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
	}
}

func printTable(records []Record) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tBUNDLE ID\tPATH")
	for _, r := range records {
		name := limit(r.Name, 30)
		bid := limit(r.Identifier, 40)
		// Fallback if Identifier is empty but BundleID (short name) exists?
		if bid == "" {
			bid = limit(r.BundleID, 40)
		}
		path := limit(r.Path, 50)
		fmt.Fprintf(w, "%s\t%s\t%s\n", name, bid, path)
	}
	w.Flush()
	fmt.Printf("\nTotal: %d records\n", len(records))
}

func printRecord(r Record) {
	fmt.Println(r.Raw)
}

func limit(s string, n int) string {
	if len(s) > n {
		return s[:n-3] + "..."
	}
	return s
}

func filterRecords(records []Record, query string) []Record {
	if query == "" {
		return records
	}
	query = strings.ToLower(query)
	var matches []Record
	for _, r := range records {
		if strings.Contains(strings.ToLower(r.Name), query) ||
			strings.Contains(strings.ToLower(r.BundleID), query) ||
			strings.Contains(strings.ToLower(r.Identifier), query) ||
			strings.Contains(strings.ToLower(r.Path), query) {
			matches = append(matches, r)
		}
	}
	return matches
}
