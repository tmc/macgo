// Package plist provides utilities for generating macOS property list files.
package plist

import (
	"strings"
)

// EscapeXML escapes special characters for XML content.
// This function handles the basic XML entities that need to be escaped
// when embedding content in XML documents.
func EscapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}

// xmlHeader returns the standard XML header for property list files.
func xmlHeader() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">`
}

// wrapPlist wraps content in a plist root element.
func wrapPlist(content string) string {
	return xmlHeader() + "\n" + `<plist version="1.0">` + "\n" + content + "\n" + `</plist>`
}

// wrapDict wraps content in a dict element.
func wrapDict(content string) string {
	return `<dict>` + "\n" + content + "\n" + `</dict>`
}

// xmlKeyValue creates a key-value pair for XML plists.
func xmlKeyValue(key, value string) string {
	return "\t<key>" + EscapeXML(key) + "</key>\n\t<string>" + EscapeXML(value) + "</string>"
}

// xmlKeyBool creates a key-boolean pair for XML plists.
func xmlKeyBool(key string, value bool) string {
	boolValue := "<true/>"
	if !value {
		boolValue = "<false/>"
	}
	return "\t<key>" + EscapeXML(key) + "</key>\n\t" + boolValue
}

// xmlKeyArray creates a key-array pair for XML plists.
func xmlKeyArray(key string, values []string) string {
	if len(values) == 0 {
		return "\t<key>" + EscapeXML(key) + "</key>\n\t<array/>"
	}

	result := "\t<key>" + EscapeXML(key) + "</key>\n\t<array>"
	for _, value := range values {
		result += "\n\t\t<string>" + EscapeXML(value) + "</string>"
	}
	result += "\n\t</array>"
	return result
}
