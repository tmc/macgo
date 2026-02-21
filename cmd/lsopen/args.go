package main

import (
	"fmt"
	"strings"
)

type options struct {
	appName       string   // -a
	bundleID      string   // -b
	useTextEdit   bool     // -e
	useTextEditor bool     // -t
	readStdin     bool     // -f
	fresh         bool     // -F
	waitForExit   bool     // -W
	reveal        bool     // -R
	newInstance   bool     // -n
	background    bool     // -g
	hidden        bool     // -j
	searchHeaders bool     // -h
	sdk           string   // -s
	forceURL      bool     // -u
	stdinPath     string   // --stdin / -i
	stdoutPath    string   // --stdout / -o
	stderrPath    string   // --stderr
	env           []string // --env
	arch          string   // --arch
	args          []string // --args (rest)
	files         []string // positional
}

const usage = `Usage: lsopen [-e] [-t] [-f] [-W] [-R] [-n] [-g] [-j] [-h] [-s <partial SDK name>] [-b <bundle identifier>] [-a <application>] [-u URL] [filenames] [--args arguments]

Help: Open opens files from a shell.
      By default, opens each file using the default application for that file.
      If the file is in the form of a URL, the file will be opened as a URL.

Options:
      -a                Opens with the specified application.
      -b                Opens with the specified application bundle identifier.
      -e                Opens with TextEdit.
      -t                Opens with default text editor.
      -f                Reads input from standard input and opens with TextEdit.
      -F  --fresh       Launches the app fresh, without restoring windows.
      -W  --wait-apps   Blocks until the used applications are closed.
      -R  --reveal      Selects in the Finder instead of opening.
      -n  --new         Open a new instance of the application.
      -g  --background  Does not bring the application to the foreground.
      -j  --hide        Launches the app hidden.
      -h  --header      Searches header file locations for matching headers.
      -s                For -h, the SDK to use.
      -u  --url         Open this URL, even if it matches a filepath.
      -i  --stdin PATH  Launches with stdin connected to PATH.
      -o  --stdout PATH Launches with stdout connected to PATH.
          --stderr PATH Launches with stderr connected to PATH.
          --env VAR     Add an environment variable (AAA=foo or AAA).
          --arch ARCH   Open with given CPU architecture.
          --args        All remaining args passed to the application.`

func parseArgs(args []string) (*options, error) {
	opts := &options{}
	i := 0

	for i < len(args) {
		arg := args[i]

		if arg == "--args" {
			opts.args = args[i+1:]
			break
		}

		if arg == "--" {
			i++
			opts.files = append(opts.files, args[i:]...)
			break
		}

		switch arg {
		case "--fresh":
			opts.fresh = true
		case "--wait-apps":
			opts.waitForExit = true
		case "--reveal":
			opts.reveal = true
		case "--new":
			opts.newInstance = true
		case "--background":
			opts.background = true
		case "--hide":
			opts.hidden = true
		case "--header":
			opts.searchHeaders = true
		case "--url":
			opts.forceURL = true
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("--url requires an argument")
			}
			opts.files = append(opts.files, args[i])
		case "--stdin":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("--stdin requires a path argument")
			}
			opts.stdinPath = args[i]
		case "--stdout":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("--stdout requires a path argument")
			}
			opts.stdoutPath = args[i]
		case "--stderr":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("--stderr requires a path argument")
			}
			opts.stderrPath = args[i]
		case "--env":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("--env requires a VAR argument")
			}
			opts.env = append(opts.env, args[i])
		case "--arch":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("--arch requires an argument")
			}
			opts.arch = args[i]
		default:
			if strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--") && len(arg) > 1 {
				// Short flags — may be combined like -gn
				flags := arg[1:]
				j := 0
				for j < len(flags) {
					switch flags[j] {
					case 'a':
						i++
						if i >= len(args) {
							return nil, fmt.Errorf("-a requires an application argument")
						}
						opts.appName = args[i]
					case 'b':
						i++
						if i >= len(args) {
							return nil, fmt.Errorf("-b requires a bundle identifier argument")
						}
						opts.bundleID = args[i]
					case 'e':
						opts.useTextEdit = true
					case 't':
						opts.useTextEditor = true
					case 'f':
						opts.readStdin = true
					case 'F':
						opts.fresh = true
					case 'W':
						opts.waitForExit = true
					case 'R':
						opts.reveal = true
					case 'n':
						opts.newInstance = true
					case 'g':
						opts.background = true
					case 'j':
						opts.hidden = true
					case 'h':
						opts.searchHeaders = true
					case 'H':
						opts.searchHeaders = true
					case 's':
						i++
						if i >= len(args) {
							return nil, fmt.Errorf("-s requires an SDK name argument")
						}
						opts.sdk = args[i]
					case 'u':
						opts.forceURL = true
						// -u may take inline URL or next positional
						if j+1 < len(flags) {
							// rest of flags is the URL? No, -u is always followed by a separate arg
						}
					case 'i':
						i++
						if i >= len(args) {
							return nil, fmt.Errorf("-i requires a path argument")
						}
						opts.stdinPath = args[i]
					case 'o':
						i++
						if i >= len(args) {
							return nil, fmt.Errorf("-o requires a path argument")
						}
						opts.stdoutPath = args[i]
					case 'v':
						// version — ignore
					default:
						return nil, fmt.Errorf("unknown option: -%c", flags[j])
					}
					j++
				}
			} else if strings.HasPrefix(arg, "--") {
				return nil, fmt.Errorf("unknown option: %s", arg)
			} else {
				opts.files = append(opts.files, arg)
			}
		}
		i++
	}

	// Validate
	if opts.readStdin && len(opts.files) > 0 {
		// -f with files: files take precedence, stdin is ignored
	}
	if opts.useTextEdit && opts.useTextEditor {
		return nil, fmt.Errorf("-e and -t are mutually exclusive")
	}

	return opts, nil
}
