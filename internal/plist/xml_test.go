package plist

import (
	"strings"
	"testing"
)

func TestEscapeXML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no special characters",
			input:    "HelloWorld123",
			expected: "HelloWorld123",
		},
		{
			name:     "ampersand",
			input:    "Tom & Jerry",
			expected: "Tom &amp; Jerry",
		},
		{
			name:     "less than",
			input:    "5 < 10",
			expected: "5 &lt; 10",
		},
		{
			name:     "greater than",
			input:    "10 > 5",
			expected: "10 &gt; 5",
		},
		{
			name:     "double quotes",
			input:    `Say "hello"`,
			expected: "Say &quot;hello&quot;",
		},
		{
			name:     "single quotes",
			input:    "It's working",
			expected: "It&#39;s working",
		},
		{
			name:     "all special characters",
			input:    `Tom & Jerry's "show" <aired> at 5:30`,
			expected: "Tom &amp; Jerry&#39;s &quot;show&quot; &lt;aired&gt; at 5:30",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EscapeXML(tt.input)
			if result != tt.expected {
				t.Errorf("EscapeXML(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestXmlHeader(t *testing.T) {
	header := xmlHeader()
	expected := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">`

	if header != expected {
		t.Errorf("xmlHeader() = %q, want %q", header, expected)
	}
}

func TestWrapPlist(t *testing.T) {
	content := "<dict>\n\t<key>test</key>\n\t<string>value</string>\n</dict>"
	result := wrapPlist(content)

	if !strings.Contains(result, `<?xml version="1.0" encoding="UTF-8"?>`) {
		t.Error("Result should contain XML header")
	}
	if !strings.Contains(result, `<!DOCTYPE plist`) {
		t.Error("Result should contain plist DOCTYPE")
	}
	if !strings.Contains(result, `<plist version="1.0">`) {
		t.Error("Result should contain plist root element")
	}
	if !strings.Contains(result, content) {
		t.Error("Result should contain the input content")
	}
	if !strings.Contains(result, `</plist>`) {
		t.Error("Result should contain closing plist tag")
	}
}

func TestWrapDict(t *testing.T) {
	content := "\t<key>test</key>\n\t<string>value</string>"
	result := wrapDict(content)
	expected := "<dict>\n" + content + "\n</dict>"

	if result != expected {
		t.Errorf("wrapDict() = %q, want %q", result, expected)
	}
}

func TestXmlKeyValue(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		expected string
	}{
		{
			name:     "simple key-value",
			key:      "CFBundleName",
			value:    "MyApp",
			expected: "\t<key>CFBundleName</key>\n\t<string>MyApp</string>",
		},
		{
			name:     "key with special characters",
			key:      "Test<Key>",
			value:    "Test & Value",
			expected: "\t<key>Test&lt;Key&gt;</key>\n\t<string>Test &amp; Value</string>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := xmlKeyValue(tt.key, tt.value)
			if result != tt.expected {
				t.Errorf("xmlKeyValue(%q, %q) = %q, want %q", tt.key, tt.value, result, tt.expected)
			}
		})
	}
}

func TestXmlKeyBool(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    bool
		expected string
	}{
		{
			name:     "true value",
			key:      "LSUIElement",
			value:    true,
			expected: "\t<key>LSUIElement</key>\n\t<true/>",
		},
		{
			name:     "false value",
			key:      "NSHighResolutionCapable",
			value:    false,
			expected: "\t<key>NSHighResolutionCapable</key>\n\t<false/>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := xmlKeyBool(tt.key, tt.value)
			if result != tt.expected {
				t.Errorf("xmlKeyBool(%q, %v) = %q, want %q", tt.key, tt.value, result, tt.expected)
			}
		})
	}
}

func TestXmlKeyArray(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		values   []string
		expected string
	}{
		{
			name:     "empty array",
			key:      "TestArray",
			values:   []string{},
			expected: "\t<key>TestArray</key>\n\t<array/>",
		},
		{
			name:     "single item",
			key:      "TestArray",
			values:   []string{"item1"},
			expected: "\t<key>TestArray</key>\n\t<array>\n\t\t<string>item1</string>\n\t</array>",
		},
		{
			name:     "multiple items",
			key:      "AppGroups",
			values:   []string{"group.com.example.app1", "group.com.example.app2"},
			expected: "\t<key>AppGroups</key>\n\t<array>\n\t\t<string>group.com.example.app1</string>\n\t\t<string>group.com.example.app2</string>\n\t</array>",
		},
		{
			name:     "items with special characters",
			key:      "TestArray",
			values:   []string{"item & value", "item < 2"},
			expected: "\t<key>TestArray</key>\n\t<array>\n\t\t<string>item &amp; value</string>\n\t\t<string>item &lt; 2</string>\n\t</array>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := xmlKeyArray(tt.key, tt.values)
			if result != tt.expected {
				t.Errorf("xmlKeyArray(%q, %v) = %q, want %q", tt.key, tt.values, result, tt.expected)
			}
		})
	}
}