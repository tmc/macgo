# sdef-to-cobra

Dynamic Cobra CLI generator from AppleScript dictionaries (sdef files).

This tool parses an application's AppleScript API and creates a type-safe Go wrapper using Cobra commands.

## How It Works

1. **Discover** - Runs `sdef` to get the application's AppleScript dictionary (XML)
2. **Parse** - Parses the XML into Go structs (Commands, Classes, Properties)
3. **Generate** - Dynamically creates Cobra commands for each AppleScript command
4. **Execute** - Translates flags into AppleScript and runs via `osascript`

## Usage

### Load and inspect an application's API

```bash
$ ./sdef-to-cobra --app /Applications/Safari.app load

Loaded safari API

Safari suite (Safari specific classes):

Commands:
  do JavaScript                  Applies a string of JavaScript code to a document.
  email contents                 Emails the contents of a tab.
  search the web                 Searches the web using Safari's current search provider.
  ...

Classes:
  tab                            A Safari window tab.
    URL                          text [r/w] (The current URL of the tab.)
    name                         text [r] (The name of the tab.)
    visible                      boolean [r] (Whether the tab is currently visible.)
```

### Example: Music.app

```bash
$ ./sdef-to-cobra --app /Applications/Music.app load
```

Would show commands like:
- play
- pause
- next track
- previous track
- set volume

And classes like:
- track (with properties: name, artist, album, duration)
- playlist
- source

## Architecture

### XML Parsing

The sdef XML format looks like:

```xml
<suite name="Safari suite" code="sfri">
  <command name="do JavaScript" code="sfridojs">
    <direct-parameter type="text" description="The JavaScript code"/>
    <parameter name="in" code="dcnm" optional="yes">
      <type type="document"/>
    </parameter>
  </command>
</suite>
```

This gets parsed into:

```go
type Command struct {
    Name        string
    Code        string
    Description string
    Parameters  []Parameter
}
```

### Dynamic Cobra Commands

For each AppleScript command, we generate a Cobra command:

```go
cmd := &cobra.Command{
    Use:   "do-javascript",
    Short: "Applies JavaScript code to a document",
    Run:   func(cmd *cobra.Command, args []string) {
        // Build AppleScript from flags
        // Execute via osascript
    },
}
```

### Example Execution

```bash
$ ./safari do-javascript --input "alert('Hello')"
```

Translates to:

```applescript
tell application "Safari"
    do JavaScript "alert('Hello')"
end tell
```

## Future Enhancements

1. **Standalone Generation**: Generate a complete standalone CLI tool
2. **Type Safety**: Convert AppleScript types to Go types
3. **Completions**: Shell completion for commands and parameters
4. **Interactive Mode**: REPL for exploring AppleScript APIs
5. **Code Generation**: Generate Go client libraries from sdef

## Comparison to osascript-wrapper

| Feature | osascript-wrapper | sdef-to-cobra |
|---------|------------------|----------------|
| Storage | File-based scripts | Dynamic commands |
| Discovery | Manual | Automatic from sdef |
| Type safety | None | Cobra flags |
| Completion | No | Possible |
| Extensibility | Text editing | Code generation |

## Related Tools

- **osascript** - Apple's AppleScript execution tool
- **sdef** - AppleScript dictionary extractor
- **osacompile** - AppleScript compiler
- **osascript-wrapper** - File-based AppleScript management

This tool bridges the gap between AppleScript's dynamic nature and Go's type safety.
