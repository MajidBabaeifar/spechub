package site

import (
	"bytes"
	"path"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"

	"github.com/specfork/specfork/internal/spec"
)

// linker resolves links inside one spec's rendered content (WEB-6).
type linker struct {
	// file is the repo-relative path being rendered, e.g. "specs/slots.md" or "README.md".
	file string
	// ids maps a local requirement ID to its file, for anchors.
	ids map[string]string
	// areaAnchor maps a repo-relative area file to its section anchor.
	areaAnchor map[string]string
	// pageOf returns the site path of an indexed spec, or "" when it isn't indexed.
	pageOf func(name string) string
	// blob is the base URL for other repo files at the rendered commit.
	blob string

	demote int
}

var (
	globalRefRe = regexp.MustCompile(`^(@[a-z0-9-]+/[a-z0-9-]+)(?::([A-Z][A-Z0-9]*-[0-9]+))?$`)
	localIDRe   = regexp.MustCompile(`^[A-Z][A-Z0-9]*-[0-9]+$`)
)

// render turns one Markdown file into safe HTML (WEB-8): raw HTML is escaped by
// goldmark's default renderer and dangerous URL schemes are dropped.
func (l *linker) render(src []byte, demote int) (string, error) {
	md := goldmark.New(
		goldmark.WithExtensions(extension.Table, extension.Strikethrough, extension.Linkify),
		goldmark.WithParserOptions(parser.WithASTTransformers(util.Prioritized(l, 100))),
	)
	l.demote = demote
	var buf bytes.Buffer
	if err := md.Convert(src, &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// Transform rewrites the AST before rendering.
func (l *linker) Transform(doc *ast.Document, reader text.Reader, pc parser.Context) {
	source := reader.Source()
	var codeSpans []*ast.CodeSpan
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch v := n.(type) {
		case *ast.Heading:
			v.Level += l.demote
			if v.Level > 6 {
				v.Level = 6
			}
		case *ast.Link:
			v.Destination = []byte(l.href(string(v.Destination)))
			if isExternal(string(v.Destination)) {
				v.SetAttributeString("rel", []byte("nofollow ugc noopener"))
			}
		case *ast.AutoLink:
			// left as is; goldmark escapes it and drops dangerous schemes
		case *ast.CodeSpan:
			if _, ok := v.Parent().(*ast.Link); !ok {
				codeSpans = append(codeSpans, v)
			}
		case *extast.TableRow:
			if first := v.FirstChild(); first != nil {
				id := strings.TrimSpace(string(cellText(first, source)))
				if localIDRe.MatchString(id) {
					v.SetAttributeString("id", []byte(id))
				}
			}
		}
		return ast.WalkContinue, nil
	})
	for _, cs := range codeSpans {
		m := globalRefRe.FindStringSubmatch(string(cellText(cs, source)))
		if m == nil {
			continue
		}
		page := l.pageOf(m[1])
		if page == "" {
			continue
		}
		dest := page
		if m[2] != "" {
			dest += "#" + m[2]
		}
		link := ast.NewLink()
		link.Destination = []byte(dest)
		parent := cs.Parent()
		parent.ReplaceChild(parent, cs, link)
		link.AppendChild(link, cs)
	}
}

// href maps a link destination found in the spec to its place on the site.
func (l *linker) href(dest string) string {
	if dest == "" || strings.HasPrefix(dest, "#") || strings.HasPrefix(dest, "/") || isExternal(dest) || strings.Contains(dest, ":") {
		return dest
	}
	target, frag, _ := strings.Cut(dest, "#")
	resolved := path.Clean(path.Join(path.Dir(l.file), target))
	if anchor, ok := l.areaAnchor[resolved]; ok {
		if frag != "" {
			return "#" + frag
		}
		return "#" + anchor
	}
	if strings.HasPrefix(resolved, "../") {
		return ""
	}
	return l.blob + resolved
}

func isExternal(u string) bool {
	return strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://")
}

// cellText collects the plain text under a node.
func cellText(n ast.Node, source []byte) []byte {
	var b bytes.Buffer
	ast.Walk(n, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if t, ok := c.(*ast.Text); ok {
			b.Write(t.Segment.Value(source))
		}
		if s, ok := c.(*ast.String); ok {
			b.Write(s.Value)
		}
		return ast.WalkContinue, nil
	})
	return b.Bytes()
}

// idLinks turns "[ID](file.md)" into in-page anchors before parsing, so a requirement
// link lands on its row rather than its area (WEB-6).
func (l *linker) idLinks(src []byte) []byte {
	re := regexp.MustCompile(`\[([A-Z][A-Z0-9]*-[0-9]+)\]\(([^)\s#]+\.md)\)`)
	return re.ReplaceAllFunc(src, func(m []byte) []byte {
		sm := re.FindSubmatch(m)
		id := string(sm[1])
		if _, ok := l.ids[id]; ok {
			return []byte("[" + id + "](#" + id + ")")
		}
		return m
	})
}

// stripFrontMatter returns the README body.
func stripFrontMatter(b []byte) []byte {
	if _, body, err := spec.SplitFrontMatter(b); err == nil {
		return body
	}
	return b
}
