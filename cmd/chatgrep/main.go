package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/danielfollent/chatgrep/internal/claude"
	"github.com/danielfollent/chatgrep/internal/codex"
	"github.com/danielfollent/chatgrep/internal/color"
	"github.com/danielfollent/chatgrep/internal/copilot"
	"github.com/danielfollent/chatgrep/internal/fzf"
	"github.com/danielfollent/chatgrep/internal/preview"
	"github.com/danielfollent/chatgrep/internal/provider"
)

var version = "dev"

// multiStringFlag allows a flag to be specified multiple times.
type multiStringFlag []string

func (m *multiStringFlag) String() string     { return strings.Join(*m, ", ") }
func (m *multiStringFlag) Set(v string) error { *m = append(*m, v); return nil }

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "chatgrep: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		agent       string
		project     string
		plain       bool
		first       bool
		previewMode bool
		showVersion bool
		colorMode   string
		colorSpecs  multiStringFlag
	)

	flag.StringVar(&agent, "agent", "all", "")
	flag.StringVar(&agent, "a", "all", "")
	flag.StringVar(&project, "project", "", "")
	flag.StringVar(&project, "p", "", "")
	flag.BoolVar(&plain, "plain", false, "")
	flag.BoolVar(&first, "first", false, "")
	flag.BoolVar(&first, "f", false, "")
	flag.BoolVar(&previewMode, "preview", false, "")
	flag.BoolVar(&showVersion, "version", false, "")
	flag.StringVar(&colorMode, "color", "auto", "")
	flag.Var(&colorSpecs, "colors", "")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: chatgrep [flags] <query>\n\nFlags:\n")
		fmt.Fprintf(os.Stderr, "  -a, --agent <name>     agent to search: claude, copilot, codex, all (default \"all\")\n")
		fmt.Fprintf(os.Stderr, "  -p, --project <path>   filter to sessions in this directory (use . for cwd)\n")
		fmt.Fprintf(os.Stderr, "      --plain            plain text output (no fzf)\n")
		fmt.Fprintf(os.Stderr, "  -f, --first            print resume command for first match and exit\n")
		fmt.Fprintf(os.Stderr, "      --color <when>     color output: auto, always, never (default \"auto\")\n")
		fmt.Fprintf(os.Stderr, "      --colors <spec>    customize colors: 'element:attr:value' (repeatable)\n")
		fmt.Fprintf(os.Stderr, "      --version          print version and exit\n")
	}
	flag.Parse()

	if showVersion {
		fmt.Println("chatgrep " + version)
		return nil
	}

	// Build theme from flags and env
	theme, err := buildTheme(colorMode, colorSpecs)
	if err != nil {
		return err
	}

	if previewMode {
		return runPreview(flag.Args(), theme)
	}

	projectFilter, err := resolveProjectFlag(project)
	if err != nil {
		return err
	}

	if first {
		return runFirst(agent, flag.Arg(0), projectFilter)
	}

	pipePlain := plain || !isTTY(os.Stdout)
	return runSearch(agent, flag.Arg(0), projectFilter, pipePlain, theme, colorMode, colorSpecs)
}

// buildTheme constructs a Theme from the --color flag, CHATGREP_COLORS env, and --colors flags.
func buildTheme(colorMode string, colorSpecs []string) (*color.Theme, error) {
	isTerminal := isTTY(os.Stdout)
	enabled := color.ResolveEnabled(colorMode, isTerminal)

	theme := color.DefaultTheme()
	theme.Enabled = enabled

	// CHATGREP_COLORS env (lower precedence)
	if envColors := os.Getenv("CHATGREP_COLORS"); envColors != "" {
		specs := strings.Split(envColors, ";")
		if err := theme.ApplySpecs(specs); err != nil {
			return nil, fmt.Errorf("CHATGREP_COLORS: %w", err)
		}
	}

	// --colors flags (higher precedence, overrides env)
	if err := theme.ApplySpecs(colorSpecs); err != nil {
		return nil, fmt.Errorf("--colors: %w", err)
	}

	return theme, nil
}

// runPreview handles the --preview callback from fzf.
// Args: [provider:sessionID, msgUUID]
func runPreview(args []string, theme *color.Theme) error {
	if len(args) < 2 {
		return fmt.Errorf("--preview requires 2 args: provider:sessionID msgUUID")
	}

	provName, sessionID, err := parsePreviewTarget(args[0])
	if err != nil {
		return err
	}

	prov, err := makeProvider(provName)
	if err != nil {
		return err
	}

	msgs, err := prov.PreviewContext(sessionID, args[1], 5)
	if err != nil {
		return err
	}

	fmt.Print(preview.Render(msgs, args[1], theme))
	return nil
}

// runSearch is the main interactive flow: search, fzf, output resume command.
func runSearch(agent, query, projectFilter string, plain bool, theme *color.Theme, colorMode string, colorSpecs []string) error {
	names, err := resolveProviderNames(agent)
	if err != nil {
		return err
	}

	providers := make(map[string]provider.Provider)
	for _, name := range names {
		p, err := makeProvider(name)
		if err != nil {
			// When searching all agents, skip providers whose config dir is missing
			if agent == "all" {
				continue
			}
			return err
		}
		providers[name] = p
	}

	// Search all providers and collect matches
	var allMatches []provider.Match
	for _, p := range providers {
		matches, err := p.Search(query, 8)
		if err != nil {
			return fmt.Errorf("%s: %w", p.Name(), err)
		}
		allMatches = append(allMatches, matches...)
	}

	allMatches = filterByProject(allMatches, projectFilter)

	if len(allMatches) == 0 {
		if query != "" {
			fmt.Fprintf(os.Stderr, "no matches for %q\n", query)
		} else {
			fmt.Fprintln(os.Stderr, "no sessions found")
		}
		return nil
	}

	// Sort newest-first; fzf uses input order as tiebreaker via --tiebreak=index
	sort.Slice(allMatches, func(i, j int) bool {
		return allMatches[i].Timestamp > allMatches[j].Timestamp
	})

	// Filter empty snippets
	var filtered []provider.Match
	for _, m := range allMatches {
		if strings.TrimSpace(m.Snippet) != "" {
			filtered = append(filtered, m)
		}
	}

	if len(filtered) == 0 {
		fmt.Fprintf(os.Stderr, "no matches for %q\n", query)
		return nil
	}

	// Pipe/plain mode: print tab-delimited lines, no fzf
	if plain {
		for _, m := range filtered {
			fmt.Println(fzf.FormatPlainLine(m))
		}
		return nil
	}

	// Interactive mode: launch fzf
	var lines []string
	for _, m := range filtered {
		lines = append(lines, fzf.FormatLine(m, theme))
	}

	binary, err := os.Executable()
	if err != nil {
		binary = "chatgrep"
	}
	previewCmd := buildPreviewCmd(binary, colorMode, colorSpecs)

	selected, err := fzf.Run(lines, previewCmd, binary, query, theme.Enabled)
	if err != nil {
		return err
	}
	if selected == "" {
		return nil // user cancelled
	}

	sel, err := fzf.ParseSelection(selected)
	if err != nil {
		return err
	}

	p, ok := providers[sel.ProviderName]
	if !ok {
		return fmt.Errorf("unknown provider in selection: %q", sel.ProviderName)
	}

	// Find CWD from the matched result
	var cwd string
	for _, m := range allMatches {
		if m.SessionID == sel.SessionID && m.ProviderName == sel.ProviderName {
			cwd = m.CWD
			break
		}
	}

	fmt.Println(p.ResumeCommand(sel.SessionID, cwd))
	return nil
}

// runFirst prints the resume command for the top match (newest) and exits.
func runFirst(agent, query, projectFilter string) error {
	names, err := resolveProviderNames(agent)
	if err != nil {
		return err
	}

	providers := make(map[string]provider.Provider)
	for _, name := range names {
		p, err := makeProvider(name)
		if err != nil {
			if agent == "all" {
				continue
			}
			return err
		}
		providers[name] = p
	}

	var allMatches []provider.Match
	for _, p := range providers {
		matches, err := p.Search(query, 8)
		if err != nil {
			return fmt.Errorf("%s: %w", p.Name(), err)
		}
		allMatches = append(allMatches, matches...)
	}

	allMatches = filterByProject(allMatches, projectFilter)

	if len(allMatches) == 0 {
		if query != "" {
			fmt.Fprintf(os.Stderr, "no matches for %q\n", query)
		} else {
			fmt.Fprintln(os.Stderr, "no sessions found")
		}
		return nil
	}

	sort.Slice(allMatches, func(i, j int) bool {
		return allMatches[i].Timestamp > allMatches[j].Timestamp
	})

	cmd := resumeCommandForMatch(allMatches[0], providers)
	if cmd == "" {
		return fmt.Errorf("unknown provider: %s", allMatches[0].ProviderName)
	}
	fmt.Println(cmd)
	return nil
}

// resumeCommandForMatch builds a resume command from a match and its provider.
func resumeCommandForMatch(m provider.Match, providers map[string]provider.Provider) string {
	p, ok := providers[m.ProviderName]
	if !ok {
		return ""
	}
	return p.ResumeCommand(m.SessionID, m.CWD)
}

// makeProvider creates a provider by name, validating that its config dir exists.
func makeProvider(name string) (provider.Provider, error) {
	switch name {
	case "claude":
		dir := claude.DefaultBaseDir()
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			return nil, fmt.Errorf("claude sessions dir not found: %s", dir)
		}
		return claude.NewProvider(dir), nil
	case "copilot":
		dir := copilot.DefaultBaseDir()
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			return nil, fmt.Errorf("copilot sessions dir not found: %s", dir)
		}
		return copilot.NewProvider(dir), nil
	case "codex":
		dir := codex.DefaultBaseDir()
		sessionsDir := codex.SessionsDir(dir)
		if _, err := os.Stat(sessionsDir); os.IsNotExist(err) {
			return nil, fmt.Errorf("codex sessions dir not found: %s", sessionsDir)
		}
		return codex.NewProvider(dir), nil
	default:
		return nil, fmt.Errorf("unknown agent: %q", name)
	}
}

func resolveProviderNames(agent string) ([]string, error) {
	switch agent {
	case "claude":
		return []string{"claude"}, nil
	case "copilot":
		return []string{"copilot"}, nil
	case "codex":
		return []string{"codex"}, nil
	case "all":
		return []string{"claude", "copilot", "codex"}, nil
	default:
		return nil, fmt.Errorf("unknown agent: %q (use claude, copilot, codex, or all)", agent)
	}
}

func parsePreviewTarget(arg string) (providerName, sessionID string, err error) {
	parts := strings.SplitN(arg, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid preview target: expected provider:sessionID, got %q", arg)
	}
	return parts[0], parts[1], nil
}

// buildPreviewCmd constructs the fzf preview command, embedding color flags
// so the preview subprocess inherits the parent's color configuration.
func buildPreviewCmd(binaryPath string, colorMode string, colorSpecs []string) string {
	cmd := binaryPath + " --color=" + colorMode
	for _, spec := range colorSpecs {
		cmd += " --colors " + shellQuote(spec)
	}
	cmd += " --preview {1} {2}"
	return cmd
}

// shellQuote wraps a string in single quotes for safe shell embedding.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// resolveProjectFlag expands "." to the current working directory.
func resolveProjectFlag(val string) (string, error) {
	if val == "" {
		return "", nil
	}
	if val == "." {
		return os.Getwd()
	}
	return val, nil
}

// isTTY reports whether f is a terminal.
func isTTY(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// filterByProject keeps only matches whose CWD starts with prefix.
func filterByProject(matches []provider.Match, prefix string) []provider.Match {
	if prefix == "" {
		return matches
	}
	var out []provider.Match
	for _, m := range matches {
		if strings.HasPrefix(m.CWD, prefix) {
			out = append(out, m)
		}
	}
	return out
}
