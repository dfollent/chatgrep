package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/danielfollent/chatgrep/internal/claude"
	"github.com/danielfollent/chatgrep/internal/copilot"
	"github.com/danielfollent/chatgrep/internal/fzf"
	"github.com/danielfollent/chatgrep/internal/preview"
	"github.com/danielfollent/chatgrep/internal/provider"
)

var version = "dev"

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
		previewMode bool
		showVersion bool
	)

	flag.StringVar(&agent, "agent", "all", "agent to search: claude, copilot, all")
	flag.StringVar(&agent, "A", "all", "agent to search (shorthand)")
	flag.StringVar(&project, "project", "", "filter to sessions in this directory (use . for cwd)")
	flag.StringVar(&project, "p", "", "filter to sessions in this directory (shorthand)")
	flag.BoolVar(&plain, "plain", false, "plain text output (no fzf)")
	flag.BoolVar(&previewMode, "preview", false, "internal: render preview for fzf")
	flag.BoolVar(&showVersion, "version", false, "print version and exit")
	flag.Parse()

	if showVersion {
		fmt.Println("chatgrep " + version)
		return nil
	}

	if previewMode {
		return runPreview(flag.Args())
	}

	projectFilter, err := resolveProjectFlag(project)
	if err != nil {
		return err
	}

	pipePlain := plain || !isTTY(os.Stdout)
	return runSearch(agent, flag.Arg(0), projectFilter, pipePlain)
}

// runPreview handles the --preview callback from fzf.
// Args: [provider:sessionID, msgUUID]
func runPreview(args []string) error {
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

	fmt.Print(preview.Render(msgs, args[1]))
	return nil
}

// runSearch is the main interactive flow: search, fzf, output resume command.
func runSearch(agent, query, projectFilter string, plain bool) error {
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
		lines = append(lines, fzf.FormatLine(m))
	}

	binary, err := os.Executable()
	if err != nil {
		binary = "chatgrep"
	}
	previewCmd := buildPreviewCmd(binary)

	selected, err := fzf.Run(lines, previewCmd, binary, query)
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
	case "all":
		return []string{"claude", "copilot"}, nil
	default:
		return nil, fmt.Errorf("unknown agent: %q (use claude, copilot, or all)", agent)
	}
}

func parsePreviewTarget(arg string) (providerName, sessionID string, err error) {
	parts := strings.SplitN(arg, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid preview target: expected provider:sessionID, got %q", arg)
	}
	return parts[0], parts[1], nil
}

func buildPreviewCmd(binaryPath string) string {
	return binaryPath + " --preview {1} {2}"
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
