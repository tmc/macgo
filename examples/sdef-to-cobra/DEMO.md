# sdef-to-cobra Live Demo

## What We Built

A dynamic CLI generator that:
1. Parses AppleScript dictionaries (sdef) into Go structs
2. Generates type-safe Cobra commands
3. Executes AppleScript via osascript

## Live Demo Results

### Step 1: Discover Safari's AppleScript API

```bash
$ ./sdef-to-cobra --app /Applications/Safari.app load
```

**Output:**
```
Loaded safari API

Safari suite (Safari specific classes):

Commands:
  add reading list item          Add a new Reading List item with the given URL.
  do JavaScript                  Applies a string of JavaScript code to a document.
  email contents                 Emails the contents of a tab.
  search the web                 Searches the web using Safari's current search provider.
  show bookmarks                 Shows Safari's bookmarks.

Classes:
  tab                            A Safari window tab.
    URL                          text [r/w] (The current URL of the tab.)
    name                         text [r] (The name of the tab.)
    visible                      boolean [r] (Whether the tab is currently visible.)
```

### Step 2: Run Live Tests

```bash
$ ./sdef-to-cobra test
```

**Output:**
```
Testing AppleScript execution via sdef commands...

=== Testing Music.app player state ===
Script: tell application "Music" to get player state
Player state: stopped

=== Testing Safari.app open URL ===
Script: tell application "Safari"
	make new document
	set URL of front document to "https://golang.org"
end tell
Success: Opened golang.org
```

**Result:** Safari automatically opened a new tab with golang.org! ✅

## What This Demonstrates

### 1. Automatic API Discovery
- No manual documentation needed
- Direct extraction from application binaries
- Complete command and class information

### 2. Type-Safe Execution
- Go structs model the AppleScript API
- Cobra provides CLI structure
- osascript executes the commands

### 3. Real Automation
The tests successfully:
- ✅ Queried Music.app player state
- ✅ Opened Safari
- ✅ Created a new document (tab)
- ✅ Navigated to golang.org

## The Power of sdef

Every scriptable macOS application exposes its API via sdef:

- **Safari**: tabs, windows, bookmarks, JavaScript execution
- **Music**: play, pause, playlists, tracks, volume
- **Finder**: files, folders, windows, selections
- **Mail**: accounts, mailboxes, messages, compose
- **Calendar**: events, calendars, reminders

All accessible programmatically without writing a single line of AppleScript!

## Complete Workflow

```
Application Binary
    ↓
sdef (extract dictionary)
    ↓
XML (commands, classes, properties)
    ↓
Go Parser (xml.Unmarshal)
    ↓
Cobra Commands (dynamic generation)
    ↓
osascript (execution)
    ↓
Automation! 🎉
```

## Next Steps

1. **Generate Standalone Tools**: Create `safari-cli`, `music-cli` executables
2. **Add Shell Completion**: Tab-complete commands and parameters
3. **Type Conversion**: Map AppleScript types to Go types
4. **Interactive REPL**: Explore APIs interactively
5. **Code Generation**: Generate full Go client libraries

This bridges the gap between macOS's 40-year AppleScript legacy and modern Go tooling.
