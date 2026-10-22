// Command spechub finds, lints and resolves SpecHub specs (CLI), serves MCP (MCP),
// and runs the index and claim workflows.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/MajidBabaeifar/spechub/internal/crawl"
	"github.com/MajidBabaeifar/spechub/internal/ghapi"
	"github.com/MajidBabaeifar/spechub/internal/index"
	"github.com/MajidBabaeifar/spechub/internal/mcpserver"
	"github.com/MajidBabaeifar/spechub/internal/registry"
	"github.com/MajidBabaeifar/spechub/internal/resolve"
	"github.com/MajidBabaeifar/spechub/internal/site"
	"github.com/MajidBabaeifar/spechub/internal/source"
	"github.com/MajidBabaeifar/spechub/internal/spec"
)

// Exit codes (CLI-9).
const (
	exitOK      = 0
	exitFail    = 1
	exitUsage   = 2
	exitNetwork = 3
)

type usageError struct{ msg string }

func (e usageError) Error() string { return e.msg }

type netError struct{ err error }

func (e netError) Error() string { return e.err.Error() }
func (e netError) Unwrap() error { return e.err }

// failure is a non-zero exit whose message was already printed.
type failure struct{}

func (failure) Error() string { return "" }

const usage = `spechub: find, compose and build from app specs

  spechub search <query> [--json]          find specs
  spechub info <name>[@version] [--json]   versions, requires, implementations
  spechub resolve <name>[@range] [--out dir] [--update]
                                           write the bundle and spechub.lock
  spechub lint [dir]                       check a spec repo
  spechub mcp                              MCP server on stdio
  spechub version

Index: $SPECHUB_INDEX or ` + index.DefaultURL + `
`

// app holds swappable dependencies for tests.
type app struct {
	stdout, stderr io.Writer
	loadIndex      func() (*index.Index, error)
	fetcher        source.Fetcher
}

func main() {
	a := &app{stdout: os.Stdout, stderr: os.Stderr, fetcher: source.GitHub{}}
	a.loadIndex = func() (*index.Index, error) {
		ix, err := index.Load(index.Source())
		if err != nil {
			return nil, netError{err}
		}
		return ix, nil
	}
	os.Exit(a.run(os.Args[1:]))
}

func (a *app) run(args []string) int {
	if len(args) == 0 {
		fmt.Fprint(a.stderr, usage)
		return exitUsage
	}
	var err error
	switch args[0] {
	case "search":
		err = a.search(args[1:])
	case "info":
		err = a.info(args[1:])
	case "resolve":
		err = a.resolve(args[1:])
	case "lint":
		err = a.lint(args[1:])
	case "mcp":
		err = a.mcp()
	case "crawl":
		err = a.crawl(args[1:])
	case "claim-check":
		err = a.claimCheck(args[1:])
	case "version", "--version":
		fmt.Fprintln(a.stdout, version())
	case "help", "-h", "--help":
		fmt.Fprint(a.stdout, usage)
	default:
		err = usageError{"unknown command " + args[0]}
	}
	var ue usageError
	var ne netError
	switch {
	case err == nil:
		return exitOK
	case errors.As(err, &ue):
		fmt.Fprintf(a.stderr, "spechub: %s\n\n%s", err, usage)
		return exitUsage
	case errors.As(err, &ne):
		fmt.Fprintf(a.stderr, "spechub: %s\n", err)
		return exitNetwork
	case errors.Is(err, failure{}):
		return exitFail
	default:
		fmt.Fprintf(a.stderr, "spechub: %s\n", err)
		return exitFail
	}
}

func version() string {
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" {
		return "spechub " + bi.Main.Version
	}
	return "spechub (devel)"
}

// parse splits flags from positional args so flags may follow them.
func parse(fs *flag.FlagSet, args []string) ([]string, error) {
	fs.SetOutput(io.Discard)
	var pos []string
	for len(args) > 0 {
		if err := fs.Parse(args); err != nil {
			return nil, usageError{err.Error()}
		}
		args = fs.Args()
		if len(args) > 0 {
			pos = append(pos, args[0])
			args = args[1:]
		}
	}
	return pos, nil
}

func (a *app) search(args []string) error {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	asJSON := fs.Bool("json", false, "")
	pos, err := parse(fs, args)
	if err != nil {
		return err
	}
	if len(pos) == 0 {
		return usageError{"search needs a query"}
	}
	ix, err := a.loadIndex()
	if err != nil {
		return err
	}
	hits := ix.Search(strings.Join(pos, " "))
	if *asJSON {
		out := []mcpserver.Hit{}
		for _, s := range hits {
			out = append(out, mcpserver.Hit{Name: s.Name, Summary: s.Summary, Latest: s.Latest(), Stars: s.Stars, Implementations: len(s.Implementations), Resolvable: s.Resolvable()})
		}
		return json.NewEncoder(a.stdout).Encode(out)
	}
	if len(hits) == 0 {
		fmt.Fprintln(a.stderr, "no specs match")
		return failure{}
	}
	tw := tabwriter.NewWriter(a.stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tLATEST\tSTARS\tIMPLS\tSUMMARY")
	for _, s := range hits {
		name := s.Name
		if !s.Resolvable() {
			name += " (" + s.State + ")"
		}
		fmt.Fprintf(tw, "%s\t%s\t%d\t%d\t%s\n", name, s.Latest(), s.Stars, len(s.Implementations), s.Summary)
	}
	return tw.Flush()
}

func (a *app) info(args []string) error {
	fs := flag.NewFlagSet("info", flag.ContinueOnError)
	asJSON := fs.Bool("json", false, "")
	pos, err := parse(fs, args)
	if err != nil {
		return err
	}
	if len(pos) != 1 {
		return usageError{"info needs one spec name"}
	}
	ix, err := a.loadIndex()
	if err != nil {
		return err
	}
	name, ver := spec.SplitNameVersion(pos[0])
	out, err := mcpserver.Info(ix, name, ver)
	if err != nil {
		return err
	}
	if *asJSON {
		return json.NewEncoder(a.stdout).Encode(out)
	}
	fmt.Fprint(a.stdout, out.Text())
	return nil
}

func (a *app) resolve(args []string) error {
	fs := flag.NewFlagSet("resolve", flag.ContinueOnError)
	out := fs.String("out", "spechub-bundle", "")
	update := fs.Bool("update", false, "")
	pos, err := parse(fs, args)
	if err != nil {
		return err
	}
	if len(pos) != 1 {
		return usageError{"resolve needs one spec name"}
	}
	name, rng := spec.SplitNameVersion(pos[0])
	ix, err := a.loadIndex()
	if err != nil {
		return err
	}
	var lock *resolve.Lock
	if !*update {
		if lock, err = resolve.ReadLock("spechub.lock"); err != nil {
			return err
		}
		if lock != nil && lock.Root != name {
			lock = nil
		}
		if lock != nil && rng != "" {
			return usageError{"spechub.lock pins " + name + "; drop the range or pass --update"}
		}
	}
	sel, err := resolve.Select(ix, name, rng, lock)
	if err != nil {
		return err
	}
	b, err := resolve.Build(sel, a.fetcher)
	if err != nil {
		return netError{err}
	}
	if err := b.Write(*out); err != nil {
		return err
	}
	if err := resolve.WriteLock("spechub.lock", sel.Lock()); err != nil {
		return err
	}
	for _, n := range sel.Order {
		fmt.Fprintf(a.stdout, "%s@%s  %s\n", n.Name, n.Version, short(n.SHA))
	}
	fmt.Fprintf(a.stdout, "wrote %s/ (%d files) and spechub.lock; start at %s/BUNDLE.md\n", *out, len(b.Paths), *out)
	return nil
}

func short(s string) string {
	if len(s) > 12 {
		return s[:12]
	}
	return s
}

func (a *app) lint(args []string) error {
	dir := "."
	if len(args) > 1 {
		return usageError{"lint takes at most one directory"}
	}
	if len(args) == 1 {
		dir = args[0]
	}
	issues, err := spec.LintDir(dir)
	if err != nil {
		return err
	}
	errs := 0
	for _, i := range issues {
		fmt.Fprintln(a.stdout, i)
		if !i.Warning {
			errs++
		}
	}
	if errs > 0 {
		fmt.Fprintf(a.stderr, "%d error(s)\n", errs)
		return failure{}
	}
	fmt.Fprintln(a.stderr, "ok")
	return nil
}

func (a *app) mcp() error {
	srv := mcpserver.New(mcpserver.Deps{LoadIndex: a.loadIndex, Fetcher: a.fetcher, Version: version()})
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	return srv.Run(ctx, &mcp.StdioTransport{})
}

// crawl is run by the index workflow (IDX-1, IDX-9).
func (a *app) crawl(args []string) error {
	fs := flag.NewFlagSet("crawl", flag.ContinueOnError)
	names := fs.String("names", "registry/names.json", "")
	out := fs.String("out", "_site", "")
	prevURL := fs.String("prev", index.DefaultURL, "")
	if _, err := parse(fs, args); err != nil {
		return err
	}
	reg, err := registry.Load(*names)
	if err != nil {
		return err
	}
	if probs := reg.Validate(); len(probs) > 0 {
		return errors.New("names.json: " + strings.Join(probs, "; "))
	}
	var prev *index.Index
	if *prevURL != "" {
		if p, err := index.Load(*prevURL); err == nil {
			prev = p
		} else {
			fmt.Fprintf(a.stderr, "no previous index (%v); reading everything\n", err)
		}
	}
	gh := ghapi.New(os.Getenv("GITHUB_TOKEN"))
	ix, st := crawl.Run(gh, reg, prev, time.Now())
	for _, e := range st.Errors {
		fmt.Fprintln(a.stderr, "warning:", e)
	}
	fmt.Fprintf(a.stderr, "specs %d, unclaimed %d, read %d, reused %d, skipped %d, API calls %d, rate limited %v\n",
		len(ix.Specs), len(ix.Unclaimed), st.Read, st.Reused, st.Skipped, gh.Calls, st.RateLimited)
	return site.Generate(ix, *out)
}

// claimCheck is run by the claim workflow on a registry PR (REG-2).
func (a *app) claimCheck(args []string) error {
	fs := flag.NewFlagSet("claim-check", flag.ContinueOnError)
	repo := fs.String("repo", os.Getenv("GITHUB_REPOSITORY"), "")
	pr := fs.Int("pr", 0, "")
	base := fs.String("base", "registry/names.json", "")
	merge := fs.Bool("merge", false, "")
	if _, err := parse(fs, args); err != nil {
		return err
	}
	if *pr == 0 || *repo == "" {
		return usageError{"claim-check needs --pr and --repo"}
	}
	gh := ghapi.New(os.Getenv("GITHUB_TOKEN"))
	p, err := gh.PullRequest(*repo, *pr)
	if err != nil {
		return err
	}
	files, err := gh.PRFiles(*repo, *pr)
	if err != nil {
		return err
	}
	baseNames, err := registry.Load(*base)
	if err != nil {
		return err
	}
	var problems []string
	for _, f := range files {
		if f != "registry/names.json" {
			problems = append(problems, "a claim PR may only change registry/names.json, not "+f)
		}
	}
	verdict := registry.Verdict{}
	if len(problems) == 0 {
		raw, err := gh.Raw(p.Head.Repo.FullName, p.Head.SHA, "registry/names.json")
		if err != nil {
			return err
		}
		headNames, err := registry.Parse(raw)
		if err != nil {
			problems = append(problems, err.Error())
		} else {
			verdict = registry.CheckClaim(baseNames, headNames, p.User.Login, gh)
			problems = append(problems, verdict.Problems...)
		}
	}
	if len(problems) == 0 && verdict.OK() {
		fmt.Fprintf(a.stdout, "claim ok: %s\n", strings.Join(verdict.Added, ", "))
		if *merge {
			if err := gh.Merge(*repo, *pr, "chore(registry): claim "+strings.Join(verdict.Added, ", ")); err != nil {
				return err
			}
		}
		return nil
	}
	body := "SpecHub claim check didn't pass:\n\n- " + strings.Join(problems, "\n- ")
	if verdict.Manual {
		body += "\n\nA maintainer needs to review this PR."
	}
	body += "\n\nSee https://github.com/" + site.Repo + "#claim-a-name"
	fmt.Fprintln(a.stdout, body)
	if *merge {
		if err := gh.Comment(*repo, *pr, body); err != nil {
			return err
		}
	}
	return failure{}
}
