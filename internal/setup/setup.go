package setup

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// AllClients returns the MCP clients setup knows how to register with.
// Add more (Cursor, Continue, Zed, VS Code) by appending here.
func AllClients() []Client {
	return []Client{ClaudeCode{}, ClaudeDesktop{}, CodexCLI{}}
}

// DetectedClient is a Client whose config file is present (or whose parent
// directory exists, signalling the app is installed).
type DetectedClient struct {
	Client Client
	Path   string
	Exists bool // config file already exists
}

// Detect reports which clients look installed on the system. We consider a
// client "installed" if its config file exists OR its parent directory exists
// (the app created it but the user hasn't yet added any MCP server).
func Detect(clients []Client) []DetectedClient {
	out := make([]DetectedClient, 0, len(clients))
	for _, c := range clients {
		path, err := c.ConfigPath()
		if err != nil {
			continue
		}
		if _, err := os.Stat(path); err == nil {
			out = append(out, DetectedClient{Client: c, Path: path, Exists: true})
			continue
		}
		// Parent dir present → app installed but no MCP entries yet.
		if _, err := os.Stat(parentDir(path)); err == nil {
			out = append(out, DetectedClient{Client: c, Path: path, Exists: false})
		}
	}
	return out
}

func parentDir(p string) string {
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[:i]
	}
	if i := strings.LastIndex(p, "\\"); i >= 0 {
		return p[:i]
	}
	return "."
}

// Options controls Run() behavior.
type Options struct {
	// SelfBinary is the absolute path to remote-shell-mcp that we will write
	// into the MCP config's `command` field. Defaults to os.Executable.
	SelfBinary string
	// ServerName is the MCP server key (e.g. "remote-shell").
	ServerName string
	// Args / Env baked into the command.
	Args []string
	Env  map[string]string
	// Yes assumes "yes" to every prompt (non-interactive install).
	Yes bool
	// DryRun prints what would happen without writing anything.
	DryRun bool
	// OnlyClient, if set, restricts to a single client by name (case-insensitive
	// substring match on Client.Name()).
	OnlyClient string

	// IO for prompts and printing. Default: stdin/stderr.
	In  io.Reader
	Out io.Writer
}

// Run performs the setup interaction. Returns the per-client results.
func Run(opts Options) ([]ClientResult, error) {
	if opts.SelfBinary == "" {
		exe, err := os.Executable()
		if err != nil {
			return nil, err
		}
		opts.SelfBinary = exe
	}
	if opts.ServerName == "" {
		opts.ServerName = "remote-shell"
	}
	if opts.In == nil {
		opts.In = os.Stdin
	}
	if opts.Out == nil {
		opts.Out = os.Stderr
	}

	detected := Detect(AllClients())
	if opts.OnlyClient != "" {
		filtered := detected[:0]
		needle := strings.ToLower(opts.OnlyClient)
		for _, d := range detected {
			if strings.Contains(strings.ToLower(d.Client.Name()), needle) {
				filtered = append(filtered, d)
			}
		}
		detected = filtered
	}

	if len(detected) == 0 {
		fmt.Fprintln(opts.Out, "No supported MCP clients detected.")
		fmt.Fprintln(opts.Out, "  Looked for:")
		for _, c := range AllClients() {
			path, _ := c.ConfigPath()
			fmt.Fprintf(opts.Out, "    - %s (%s)\n", c.Name(), path)
		}
		return nil, nil
	}

	fmt.Fprintf(opts.Out, "Detected %d MCP client(s):\n", len(detected))
	for _, d := range detected {
		state := "config exists"
		if !d.Exists {
			state = "app installed, no MCP config yet"
		}
		fmt.Fprintf(opts.Out, "  - %s — %s (%s)\n", d.Client.Name(), d.Path, state)
	}
	fmt.Fprintln(opts.Out)
	fmt.Fprintf(opts.Out, "Will register: server name %q → command %s\n", opts.ServerName, opts.SelfBinary)
	if opts.DryRun {
		fmt.Fprintln(opts.Out, "(dry run — no files will be written)")
	}
	fmt.Fprintln(opts.Out)

	reader := bufio.NewReader(opts.In)
	var results []ClientResult
	for _, d := range detected {
		ok := opts.Yes
		if !ok {
			fmt.Fprintf(opts.Out, "Install into %s? [Y/n] ", d.Client.Name())
			line, _ := reader.ReadString('\n')
			line = strings.TrimSpace(strings.ToLower(line))
			ok = line == "" || line == "y" || line == "yes"
		}
		if !ok {
			results = append(results, ClientResult{Client: d.Client.Name(), Result: Result{Path: d.Path, Action: "skipped"}})
			continue
		}
		res, err := d.Client.Install(opts.ServerName, opts.SelfBinary, opts.Args, opts.Env, opts.DryRun)
		results = append(results, ClientResult{Client: d.Client.Name(), Result: res, Err: err})
		if err != nil {
			fmt.Fprintf(opts.Out, "  ERROR: %v\n", err)
			continue
		}
		fmt.Fprintf(opts.Out, "  %s: %s (%s)\n", res.Action, res.Path, d.Client.Name())
		if res.Backup != "" {
			fmt.Fprintf(opts.Out, "  backup: %s\n", res.Backup)
		}
		if opts.DryRun && res.Snippet != "" {
			fmt.Fprintln(opts.Out, "  diff:")
			for _, l := range strings.Split(strings.TrimRight(res.Snippet, "\n"), "\n") {
				fmt.Fprintln(opts.Out, "    "+l)
			}
		}
	}
	return results, nil
}

type ClientResult struct {
	Client string
	Result Result
	Err    error
}
