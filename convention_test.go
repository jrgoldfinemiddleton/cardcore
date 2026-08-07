package cardcore

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// funcGroup classifies where a function declaration belongs in the
// canonical ordering. Lower values must appear before higher values.
type funcGroup int

const (
	groupConstructor      funcGroup = iota // New* functions
	groupExportedMethod                    // Exported methods (A-Z receiver)
	groupExportedFunc                      // Exported package-level functions
	groupUnexportedMethod                  // Unexported methods (a-z receiver)
	groupUnexportedFunc                    // Unexported package-level functions
)

// testGroup classifies where a declaration belongs in test file ordering.
type testGroup int

const (
	testGroupInterfaceCheck  testGroup = iota // var _ T = (*Impl)(nil)
	testGroupUnitTest                         // func Test* (non-integration)
	testGroupIntegrationTest                  // func Test*Integration
	testGroupBenchmark                        // func Benchmark*
	testGroupFuzz                             // func Fuzz*
	testGroupExample                          // func Example*
	testGroupHelper                           // Non-Test/Benchmark/Fuzz/Example funcs
)

// funcInfo captures the ordering-relevant properties of a single
// function declaration.
type funcInfo struct {
	name     string
	group    funcGroup
	receiver string
	line     int
}

// testDeclInfo captures a declaration's position in a test file.
type testDeclInfo struct {
	name  string
	group testGroup
	line  int
}

// forbiddenCardcoreSelectors lists the cardcore identifiers that game-package
// test code must reference via prefixed aliases (rAce..rKing, sClubs..sSpades)
// rather than as qualified cardcore.X selectors.
var forbiddenCardcoreSelectors = []string{
	"Two", "Three", "Four", "Five", "Six",
	"Seven", "Eight", "Nine", "Ten",
	"Jack", "Queen", "King", "Ace",
	"Clubs", "Diamonds", "Hearts", "Spades",
}

// modulePath is this module's import path. Imports starting with this
// prefix are treated as internal and allowed by TestNoExternalDeps.
const modulePath = "github.com/jrgoldfinemiddleton/cardcore"

// walkOpts configures walkGoFiles. Zero values give sensible defaults:
// root defaults to cwd, suffix defaults to ".go", and skipDirs/skipFiles
// add to the always-skipped set (.git, vendor, testdata, .sisyphus).
type walkOpts struct {
	root      string
	suffix    string
	skipDirs  []string
	skipFiles []string
}

// docLinkRE matches bracketed Go doc links in comments: the Name, pkg,
// pkg.Symbol, pkg.Type.Member, and import/path.Symbol forms.
var docLinkRE = regexp.MustCompile(`\[([A-Za-z_][A-Za-z0-9_]*(?:[./][A-Za-z0-9_]+)*)\]`)

// adrRefRE matches ADR references in comments: ADR- followed by three
// digits.
var adrRefRE = regexp.MustCompile(`\bADR-(\d{3})\b`)

// docPathRE matches in-repo doc path references in comments, with an
// optional heading anchor (for example a doc/design.md section link).
var docPathRE = regexp.MustCompile(`\bdoc/[\w./#-]*\.md(?:#[\w-]+)?`)

// versionSuffixRE matches a major version suffix in an import path's
// last element, e.g. the v2 in math/rand/v2.
var versionSuffixRE = regexp.MustCompile(`^v\d+$`)

// agentsPathTokenRE matches backticked spans in AGENTS.md files.
var agentsPathTokenRE = regexp.MustCompile("`([^`]+)`")

// symbolSet captures the exported symbols of a package for doc link
// resolution: top-level names and per-type member names.
type symbolSet struct {
	top     map[string]bool
	members map[string]map[string]bool
}

// docLinkCache memoizes package symbol and name lookups and holds the
// set of existing ADR numbers and the module root for doc path
// validation.
type docLinkCache struct {
	symbols map[string]*symbolSet // importPath → symbols; nil means unresolvable.
	names   map[string]string     // importPath → declared package name.
	locals  map[string]*symbolSet // directory → declared top-level symbols.
	adrs    map[string]bool       // ADR number ("004") → exists.
	root    string                // Module root directory.
}

// TestRankSuitAliasesInGameTests walks every _test.go file under games/
// and verifies that no test code references cardcore rank or suit
// constants via qualified selectors (e.g. cardcore.Ace, cardcore.Hearts).
// Game-package tests must use the prefixed aliases (rAce..rKing,
// sClubs..sSpades) defined in their helpers_test.go.
func TestRankSuitAliasesInGameTests(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	walkGoFiles(t, walkOpts{root: filepath.Join(cwd, "games"), suffix: "_test.go"},
		func(path, rel string) {
			checkTestAliasUsage(t, path, rel)
		})
}

// TestNoNolint walks every .go file in the module and fails if any
// //nolint directive is present. Lint errors must be fixed in code
// rather than suppressed.
func TestNoNolint(t *testing.T) {
	walkGoFiles(t, walkOpts{}, func(path, rel string) {
		checkNoNolint(t, path, rel)
	})
}

// TestNoExternalDeps walks every .go file in the module and fails if any
// import is neither stdlib nor a sub-package of this module. The project
// uses the standard library only at runtime.
func TestNoExternalDeps(t *testing.T) {
	walkGoFiles(t, walkOpts{}, func(path, rel string) {
		checkNoExternalDeps(t, path, rel)
	})
}

// TestNoGameImportsInRootPkg walks the .go files in the repository root
// and fails if any of them import a package under games/. Game-specific
// logic must live in subpackages, not the root.
func TestNoGameImportsInRootPkg(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		path := filepath.Join(root, e.Name())
		checkNoGameImports(t, path, e.Name())
	}
}

// TestFunctionOrdering walks every .go file in the module and verifies
// that function declarations follow the ordering conventions described
// in CONTRIBUTING.md.
func TestFunctionOrdering(t *testing.T) {
	walkGoFiles(t, walkOpts{skipDirs: []string{"doc"}, skipFiles: []string{"doc.go"}},
		func(path, rel string) {
			checkDeclsBeforeFuncs(t, path, rel)
			if strings.HasSuffix(path, "_test.go") {
				checkTestFile(t, path, rel)
			} else {
				checkProdFile(t, path, rel)
			}
		})
}

// TestDocComments walks every .go file in the module and verifies that
// every function and method has a doc comment starting with its name.
// For doc.go files, it verifies the package doc comment exists and
// starts with "Package <name>".
func TestDocComments(t *testing.T) {
	walkGoFiles(t, walkOpts{skipDirs: []string{"doc"}}, func(path, rel string) {
		if strings.HasSuffix(path, "doc.go") {
			checkPackageDoc(t, path, rel)
			return
		}
		checkDocComments(t, path, rel)
	})
}

// TestDocLinks walks every .go file in the module and verifies that
// bracketed doc links resolve, that same-package identifiers are not
// linked, and that ADR and doc path references in comments point at
// existing files.
func TestDocLinks(t *testing.T) {
	cache := newDocLinkCache(t)
	walkGoFiles(t, walkOpts{}, func(path, rel string) {
		checkDocLinks(t, path, rel, cache)
	})
}

// TestAgentsMDPaths walks every nested AGENTS.md file and verifies that
// backticked path references in them resolve: relative to the file's
// own directory first, then relative to the module root. The root
// AGENTS.md is exempt; only nested files carry directory inventories.
func TestAgentsMDPaths(t *testing.T) {
	walkGoFiles(t, walkOpts{suffix: "AGENTS.md", skipDirs: []string{".omo"}},
		func(path, rel string) {
			if rel == "AGENTS.md" {
				return
			}
			checkAgentsMDPaths(t, path, rel)
		})
}

// walkGoFiles walks Go source files under opts.root (default: cwd) and
// invokes fn for each file matching opts.suffix (default: ".go").
// Directories named .git, vendor, testdata, or .sisyphus are always
// skipped, plus any in opts.skipDirs. Files whose basename appears in
// opts.skipFiles are skipped. rel is the path relative to the working
// directory.
func walkGoFiles(t *testing.T, opts walkOpts, fn func(path, rel string)) {
	t.Helper()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	root := opts.root
	if root == "" {
		root = cwd
	}
	suffix := opts.suffix
	if suffix == "" {
		suffix = ".go"
	}

	if _, err := os.Stat(root); os.IsNotExist(err) {
		return
	}

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			base := d.Name()
			switch base {
			case ".git", "vendor", "testdata", ".sisyphus":
				return filepath.SkipDir
			}
			if slices.Contains(opts.skipDirs, base) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, suffix) {
			return nil
		}
		base := filepath.Base(path)
		if slices.Contains(opts.skipFiles, base) {
			return nil
		}

		rel, _ := filepath.Rel(cwd, path)
		fn(path, rel)
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir: %v", err)
	}
}

// checkProdFile verifies production file ordering: constructors →
// exported methods → exported funcs → unexported methods → unexported
// funcs, with methods on the same receiver contiguous.
func checkProdFile(t *testing.T, path, rel string) {
	t.Helper()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Errorf("%s: parse error: %v", rel, err)
		return
	}

	funcs := make([]funcInfo, 0, len(f.Decls))
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		fi := classifyFunc(fn)
		fi.line = fset.Position(fn.Pos()).Line
		funcs = append(funcs, fi)
	}

	if len(funcs) == 0 {
		return
	}

	// Check group ordering: each function's group must be >= the previous.
	for i := 1; i < len(funcs); i++ {
		prev := funcs[i-1]
		curr := funcs[i]
		if curr.group < prev.group {
			t.Errorf("%s:%d: %s (group %s) appears after %s:%d: %s (group %s) — wrong order",
				rel, curr.line, curr.name, groupName(curr.group),
				rel, prev.line, prev.name, groupName(prev.group))
		}
	}

	// Check receiver contiguity: all methods on the same receiver must
	// be adjacent (no other receiver or package-level func between them).
	lastSeen := map[string]int{} // receiver → index of last occurrence
	for i, fi := range funcs {
		if fi.receiver == "" {
			continue
		}
		if prev, ok := lastSeen[fi.receiver]; ok {
			// Verify nothing with a different receiver or no receiver
			// appeared between prev and i.
			for j := prev + 1; j < i; j++ {
				between := funcs[j]
				if between.receiver != fi.receiver {
					t.Errorf(
						"%s:%d: %s (receiver %s) is separated from %s:%d: %s "+
							"by %s:%d: %s (receiver %q)",
						rel, fi.line, fi.name, fi.receiver,
						rel, funcs[prev].line, funcs[prev].name,
						rel, between.line, between.name, receiverLabel(between.receiver))
					break
				}
			}
		}
		lastSeen[fi.receiver] = i
	}
}

// checkTestFile verifies test file ordering: interface checks → unit
// tests → integration tests → helpers.
func checkTestFile(t *testing.T, path, rel string) {
	t.Helper()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Errorf("%s: parse error: %v", rel, err)
		return
	}

	var decls []testDeclInfo

	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			td := classifyTestFunc(d)
			td.line = fset.Position(d.Pos()).Line
			decls = append(decls, td)
		case *ast.GenDecl:
			if d.Tok == token.VAR {
				for _, spec := range d.Specs {
					vs, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					if isInterfaceCheck(vs) {
						decls = append(decls, testDeclInfo{
							name:  vs.Names[0].Name,
							group: testGroupInterfaceCheck,
							line:  fset.Position(d.Pos()).Line,
						})
					}
				}
			}
		}
	}

	if len(decls) == 0 {
		return
	}

	for i := 1; i < len(decls); i++ {
		prev := decls[i-1]
		curr := decls[i]
		if curr.group < prev.group {
			t.Errorf("%s:%d: %s (group %s) appears after %s:%d: %s (group %s) — wrong order",
				rel, curr.line, curr.name, testGroupName(curr.group),
				rel, prev.line, prev.name, testGroupName(prev.group))
		}
	}
}

// checkDeclsBeforeFuncs verifies that all type, const, and var
// declarations appear before any function or method declarations.
// Import declarations are exempt.
func checkDeclsBeforeFuncs(t *testing.T, path, rel string) {
	t.Helper()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Errorf("%s: parse error: %v", rel, err)
		return
	}

	firstFuncLine := 0
	firstFuncName := ""
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		firstFuncLine = fset.Position(fn.Pos()).Line
		firstFuncName = fn.Name.Name
		break
	}

	if firstFuncLine == 0 {
		return // No functions in file.
	}

	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		if gd.Tok == token.IMPORT {
			continue
		}
		line := fset.Position(gd.Pos()).Line
		if line > firstFuncLine {
			t.Errorf(
				"%s:%d: %s declaration appears after first function %s (line %d) — "+
					"declarations must precede all functions",
				rel, line, gd.Tok, firstFuncName, firstFuncLine)
		}
	}
}

// checkDocComments verifies that every function and method in the file
// has a doc comment whose first word is the function or method name.
func checkDocComments(t *testing.T, path, rel string) {
	t.Helper()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		t.Errorf("%s: parse error: %v", rel, err)
		return
	}

	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		name := fn.Name.Name
		line := fset.Position(fn.Pos()).Line

		if fn.Doc == nil || len(fn.Doc.List) == 0 {
			t.Errorf("%s:%d: %s has no doc comment", rel, line, name)
			continue
		}

		first := fn.Doc.List[0].Text
		// Expected: "// Name ..." where Name is the function name.
		prefix := "// " + name + " "
		if !strings.HasPrefix(first, prefix) {
			t.Errorf("%s:%d: doc comment for %s must start with %q, got %q",
				rel, line, name, "// "+name+" ...", first)
		}
	}
}

// checkPackageDoc verifies that a doc.go file has a package doc comment
// starting with "Package <name>".
func checkPackageDoc(t *testing.T, path, rel string) {
	t.Helper()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		t.Errorf("%s: parse error: %v", rel, err)
		return
	}

	if f.Doc == nil || len(f.Doc.List) == 0 {
		t.Errorf("%s: doc.go has no package doc comment", rel)
		return
	}

	first := f.Doc.List[0].Text
	prefix := "// Package " + f.Name.Name + " "
	if !strings.HasPrefix(first, prefix) {
		t.Errorf("%s: package doc comment must start with %q, got %q",
			rel, "// Package "+f.Name.Name+" ...", first)
	}
}

// checkTestAliasUsage reports any cardcore.X selector in the file that
// references a rank or suit constant requiring an alias. Selectors inside
// const declarations are skipped so the alias definitions themselves
// (e.g. const rAce = cardcore.Ace) do not trigger the rule.
func checkTestAliasUsage(t *testing.T, path, rel string) {
	t.Helper()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Errorf("%s: parse error: %v", rel, err)
		return
	}

	report := func(n ast.Node) {
		ast.Inspect(n, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			ident, ok := sel.X.(*ast.Ident)
			if !ok || ident.Name != "cardcore" {
				return true
			}
			if !slices.Contains(forbiddenCardcoreSelectors, sel.Sel.Name) {
				return true
			}
			line := fset.Position(sel.Pos()).Line
			t.Errorf(
				"%s:%d: cardcore.%s must use the prefixed alias (rAce..rKing, sClubs..sSpades)",
				rel, line, sel.Sel.Name)
			return true
		})
	}

	for _, decl := range f.Decls {
		if gd, ok := decl.(*ast.GenDecl); ok && gd.Tok == token.CONST {
			continue
		}
		report(decl)
	}
}

// checkNoNolint reports any //nolint directive in the file. The check
// walks AST comments rather than scanning raw lines so that a literal
// "//nolint" appearing inside a string (such as the error message in
// this very test) is not treated as a directive.
func checkNoNolint(t *testing.T, path, rel string) {
	t.Helper()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		t.Errorf("%s: parse error: %v", rel, err)
		return
	}

	for _, cg := range f.Comments {
		for _, c := range cg.List {
			text := strings.TrimPrefix(c.Text, "//")
			text = strings.TrimPrefix(text, "/*")
			text = strings.TrimSpace(text)
			if strings.HasPrefix(text, "nolint") {
				line := fset.Position(c.Pos()).Line
				t.Errorf("%s:%d: nolint directive forbidden — fix the code instead",
					rel, line)
			}
		}
	}
}

// checkNoExternalDeps reports any import that is neither stdlib nor a
// sub-package of this module. Stdlib packages are recognized by the
// absence of a "." in their first path segment (e.g. "fmt", "go/ast"),
// since every public Go module path includes a domain.
func checkNoExternalDeps(t *testing.T, path, rel string) {
	t.Helper()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		t.Errorf("%s: parse error: %v", rel, err)
		return
	}

	for _, imp := range f.Imports {
		p := strings.Trim(imp.Path.Value, `"`)
		if p == modulePath || strings.HasPrefix(p, modulePath+"/") {
			continue
		}
		first, _, _ := strings.Cut(p, "/")
		if !strings.Contains(first, ".") {
			continue
		}
		line := fset.Position(imp.Pos()).Line
		t.Errorf("%s:%d: external dependency %q forbidden — stdlib only",
			rel, line, p)
	}
}

// checkNoGameImports reports any import that points into a game
// subpackage of this module. Used by TestNoGameImportsInRootPkg.
func checkNoGameImports(t *testing.T, path, rel string) {
	t.Helper()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		t.Errorf("%s: parse error: %v", rel, err)
		return
	}

	for _, imp := range f.Imports {
		p := strings.Trim(imp.Path.Value, `"`)
		if strings.HasPrefix(p, modulePath+"/games/") {
			line := fset.Position(imp.Pos()).Line
			t.Errorf("%s:%d: root cardcore package must not import game subpackage %q",
				rel, line, p)
		}
	}
}

// classifyFunc determines the group and receiver for a production
// function declaration.
func classifyFunc(fn *ast.FuncDecl) funcInfo {
	name := fn.Name.Name
	exported := ast.IsExported(name)
	recv := receiverType(fn)

	var g funcGroup
	switch {
	case recv == "" && exported && strings.HasPrefix(name, "New"):
		g = groupConstructor
	case recv != "" && exported:
		g = groupExportedMethod
	case recv == "" && exported:
		g = groupExportedFunc
	case recv != "" && !exported:
		g = groupUnexportedMethod
	default:
		g = groupUnexportedFunc
	}

	return funcInfo{name: name, group: g, receiver: recv}
}

// classifyTestFunc determines the test group for a function in a test file.
func classifyTestFunc(fn *ast.FuncDecl) testDeclInfo {
	name := fn.Name.Name

	var g testGroup
	switch {
	case strings.HasPrefix(name, "Test"):
		if isIntegrationTestName(name) {
			g = testGroupIntegrationTest
		} else {
			g = testGroupUnitTest
		}
	case strings.HasPrefix(name, "Benchmark"):
		g = testGroupBenchmark
	case strings.HasPrefix(name, "Fuzz"):
		g = testGroupFuzz
	case strings.HasPrefix(name, "Example"):
		g = testGroupExample
	default:
		g = testGroupHelper
	}

	return testDeclInfo{name: name, group: g}
}

// isIntegrationTestName reports whether a test function name indicates
// an integration test.
func isIntegrationTestName(name string) bool {
	return strings.HasSuffix(name, "Integration")
}

// receiverType returns the base type name of a method's receiver, or
// "" for package-level functions.
func receiverType(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return ""
	}
	t := fn.Recv.List[0].Type
	// Strip pointer.
	if star, ok := t.(*ast.StarExpr); ok {
		t = star.X
	}
	if ident, ok := t.(*ast.Ident); ok {
		return ident.Name
	}
	return ""
}

// isInterfaceCheck reports whether a var spec looks like
// var _ SomeType = (*Impl)(nil).
func isInterfaceCheck(vs *ast.ValueSpec) bool {
	if len(vs.Names) != 1 || vs.Names[0].Name != "_" {
		return false
	}
	return vs.Type != nil
}

// groupName returns a human-readable label for a production function group.
func groupName(g funcGroup) string {
	switch g {
	case groupConstructor:
		return "constructor"
	case groupExportedMethod:
		return "exported method"
	case groupExportedFunc:
		return "exported func"
	case groupUnexportedMethod:
		return "unexported method"
	case groupUnexportedFunc:
		return "unexported func"
	default:
		return "unknown"
	}
}

// testGroupName returns a human-readable label for a test declaration group.
func testGroupName(g testGroup) string {
	switch g {
	case testGroupInterfaceCheck:
		return "interface check"
	case testGroupUnitTest:
		return "unit test"
	case testGroupIntegrationTest:
		return "integration test"
	case testGroupBenchmark:
		return "benchmark"
	case testGroupFuzz:
		return "fuzz"
	case testGroupExample:
		return "example"
	case testGroupHelper:
		return "helper"
	default:
		return "unknown"
	}
}

// receiverLabel returns a display string for a receiver, or "package-level"
// if empty.
func receiverLabel(recv string) string {
	if recv == "" {
		return "package-level"
	}
	return recv
}

// newDocLinkCache builds the shared state for doc link validation: the
// module root and the set of existing ADR numbers.
func newDocLinkCache(t *testing.T) *docLinkCache {
	t.Helper()

	root, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}

	adrFileRE := regexp.MustCompile(`^(\d{3})-.+\.md$`)
	adrs := map[string]bool{}
	entries, err := os.ReadDir(filepath.Join(root, "doc", "decisions"))
	if err != nil {
		t.Fatalf("read doc/decisions: %v", err)
	}
	for _, e := range entries {
		if m := adrFileRE.FindStringSubmatch(e.Name()); m != nil {
			adrs[m[1]] = true
		}
	}

	return &docLinkCache{
		symbols: map[string]*symbolSet{},
		names:   map[string]string{},
		locals:  map[string]*symbolSet{},
		adrs:    adrs,
		root:    root,
	}
}

// checkDocLinks validates the links and references in every comment of
// a single file.
func checkDocLinks(t *testing.T, path, rel string, cache *docLinkCache) {
	t.Helper()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		t.Errorf("%s: parse error: %v", rel, err)
		return
	}

	imports := map[string]string{}
	for _, imp := range f.Imports {
		p, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			continue
		}
		name := cache.packageName(p)
		if imp.Name != nil {
			name = imp.Name.Name
		}
		imports[name] = p
	}
	if filepath.Base(path) == "doc.go" {
		mergeDirImports(t, path, imports, cache)
	}
	localSyms := cache.localSymbols(filepath.Dir(path))

	for _, cg := range f.Comments {
		for _, c := range cg.List {
			line := fset.Position(c.Pos()).Line
			checkBracketLinks(t, rel, line, c.Text, imports, cache, localSyms)
			checkADRRefs(t, rel, line, c.Text, cache)
			checkDocPaths(t, rel, line, c.Text, cache)
		}
	}
}

// checkBracketLinks validates every bracketed doc link in a comment.
// Single-word brackets that name an identifier declared in the same
// package are forbidden: name the identifier plainly instead. Other
// single-word brackets are package links or index/prose shorthand such
// as deal[seat], which Go renders as plain text. Links with a package
// and symbol part must resolve.
func checkBracketLinks(
	t *testing.T, rel string, line int, text string,
	imports map[string]string, cache *docLinkCache, localSyms *symbolSet,
) {
	t.Helper()

	for _, m := range docLinkRE.FindAllStringSubmatch(text, -1) {
		link := m[1]
		pkg, sym := splitDocLink(link)
		if sym == "" {
			if localSyms.top[pkg] {
				t.Errorf("%s:%d: same-package doc link [%s] forbidden: "+
					"name the identifier without brackets", rel, line, link)
			}
			continue
		}
		checkSymbolLink(t, rel, line, link, pkg, sym, imports, cache)
	}
}

// localSymbols collects the top-level symbols declared by every Go
// file in dir, distinguishing same-package doc links from prose
// bracket shorthand. Results are memoized per test run.
func (c *docLinkCache) localSymbols(dir string) *symbolSet {
	if syms, ok := c.locals[dir]; ok {
		return syms
	}
	syms := &symbolSet{top: map[string]bool{}, members: map[string]map[string]bool{}}
	entries, err := os.ReadDir(dir)
	if err == nil {
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".go") {
				collectFileSymbols(filepath.Join(dir, e.Name()), syms)
			}
		}
	}
	c.locals[dir] = syms
	return syms
}

// splitDocLink splits a bracketed doc link target into its package and
// symbol parts. For full import paths the symbol follows the first dot
// after the last slash (import paths may themselves contain dots); for
// short package names it follows the first dot.
func splitDocLink(link string) (pkg, sym string) {
	lastSlash := strings.LastIndex(link, "/")
	dot := strings.Index(link[lastSlash+1:], ".")
	if dot < 0 {
		return link, ""
	}
	return link[:lastSlash+1+dot], link[lastSlash+1+dot+1:]
}

// checkSymbolLink validates a pkg.Symbol or pkg.Type.Member doc link:
// the short package form must be imported by the file, and the symbol
// must exist in the target package.
func checkSymbolLink(
	t *testing.T, rel string, line int, link, pkg, sym string,
	imports map[string]string, cache *docLinkCache,
) {
	t.Helper()

	importPath := pkg
	if !strings.Contains(pkg, "/") {
		p, ok := imports[pkg]
		if !ok {
			t.Errorf("%s:%d: doc link [%s] uses the short form but package "+
				"%q is not imported by this file", rel, line, link, pkg)
			return
		}
		importPath = p
	}

	syms := cache.lookup(importPath)
	if syms == nil {
		t.Errorf("%s:%d: doc link [%s]: cannot resolve package %q",
			rel, line, link, importPath)
		return
	}

	typ, member, hasMember := strings.Cut(sym, ".")
	if !syms.top[typ] {
		t.Errorf("%s:%d: doc link [%s]: symbol %q not found in package %q",
			rel, line, link, typ, importPath)
		return
	}
	if hasMember && !syms.members[typ][member] {
		t.Errorf("%s:%d: doc link [%s]: member %q not found on %q in "+
			"package %q", rel, line, link, member, typ, importPath)
	}
}

// checkADRRefs verifies that every ADR reference in a comment names an
// ADR file that exists in doc/decisions.
func checkADRRefs(t *testing.T, rel string, line int, text string, cache *docLinkCache) {
	t.Helper()

	for _, m := range adrRefRE.FindAllStringSubmatch(text, -1) {
		if !cache.adrs[m[1]] {
			t.Errorf("%s:%d: ADR-%s does not match any file in doc/decisions",
				rel, line, m[1])
		}
	}
}

// checkDocPaths verifies that every in-repo doc path reference in a
// comment points at a file that exists relative to the module root.
// Heading anchors are ignored.
func checkDocPaths(t *testing.T, rel string, line int, text string, cache *docLinkCache) {
	t.Helper()

	for _, m := range docPathRE.FindAllString(text, -1) {
		p, _, _ := strings.Cut(m, "#")
		if _, err := os.Stat(filepath.Join(cache.root, p)); err != nil {
			t.Errorf("%s:%d: referenced doc %q does not exist", rel, line, p)
		}
	}
}

// mergeDirImports merges the import sets of all non-test Go files in
// the same directory as path into imports. Go resolves doc links in
// package documentation against the imports of the whole package.
func mergeDirImports(
	t *testing.T, path string, imports map[string]string, cache *docLinkCache,
) {
	t.Helper()

	dir := filepath.Dir(path)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") ||
			name == "doc.go" {
			continue
		}
		f, err := parser.ParseFile(
			token.NewFileSet(), filepath.Join(dir, name), nil, parser.ImportsOnly)
		if err != nil {
			continue
		}
		for _, imp := range f.Imports {
			p, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				continue
			}
			iname := cache.packageName(p)
			if imp.Name != nil {
				iname = imp.Name.Name
			}
			imports[iname] = p
		}
	}
}

// packageName returns the reference name for an import path without an
// explicit alias: the last path element, or the declared package name
// resolved via go list when the last element is a major version suffix
// (for example math/rand/v2 declares package rand).
func (c *docLinkCache) packageName(importPath string) string {
	base := importPath[strings.LastIndex(importPath, "/")+1:]
	if !versionSuffixRE.MatchString(base) {
		return base
	}
	if name, ok := c.names[importPath]; ok {
		return name
	}
	name := base
	out, err := exec.Command("go", "list", "-f", "{{.Name}}", importPath).Output()
	if err == nil {
		name = strings.TrimSpace(string(out))
	}
	c.names[importPath] = name
	return name
}

// lookup resolves the exported symbols of importPath, memoizing results
// per test run. It returns nil when the package cannot be resolved.
func (c *docLinkCache) lookup(importPath string) *symbolSet {
	if syms, ok := c.symbols[importPath]; ok {
		return syms
	}
	syms := loadPackageSymbols(importPath)
	c.symbols[importPath] = syms
	return syms
}

// loadPackageSymbols resolves importPath via go list and collects the
// exported top-level symbols and per-type members from its source. It
// returns nil when the package cannot be resolved.
func loadPackageSymbols(importPath string) *symbolSet {
	out, err := exec.Command("go", "list", "-f", "{{.Dir}}", importPath).Output()
	if err != nil {
		return nil
	}
	dir := strings.TrimSpace(string(out))

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	syms := &symbolSet{top: map[string]bool{}, members: map[string]map[string]bool{}}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		collectFileSymbols(filepath.Join(dir, name), syms)
	}
	return syms
}

// collectFileSymbols adds the exported declarations of a single Go
// source file to syms.
func collectFileSymbols(path string, syms *symbolSet) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return
	}
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			collectFuncSymbol(d, syms)
		case *ast.GenDecl:
			collectGenSymbols(d, syms)
		}
	}
}

// collectFuncSymbol adds an exported function or method to syms.
func collectFuncSymbol(fn *ast.FuncDecl, syms *symbolSet) {
	if !ast.IsExported(fn.Name.Name) {
		return
	}
	recv := receiverType(fn)
	if recv == "" {
		syms.top[fn.Name.Name] = true
		return
	}
	if syms.members[recv] == nil {
		syms.members[recv] = map[string]bool{}
	}
	syms.members[recv][fn.Name.Name] = true
}

// collectGenSymbols adds exported type, var, and const declarations to
// syms, with interface methods and struct fields recorded as members.
func collectGenSymbols(gd *ast.GenDecl, syms *symbolSet) {
	for _, spec := range gd.Specs {
		switch s := spec.(type) {
		case *ast.ValueSpec:
			for _, n := range s.Names {
				if ast.IsExported(n.Name) {
					syms.top[n.Name] = true
				}
			}
		case *ast.TypeSpec:
			if !ast.IsExported(s.Name.Name) {
				continue
			}
			syms.top[s.Name.Name] = true
			collectTypeMembers(s, syms)
		}
	}
}

// collectTypeMembers records the interface method names and struct
// field names of an exported type as members in syms.
func collectTypeMembers(ts *ast.TypeSpec, syms *symbolSet) {
	add := func(name string) {
		if syms.members[ts.Name.Name] == nil {
			syms.members[ts.Name.Name] = map[string]bool{}
		}
		syms.members[ts.Name.Name][name] = true
	}
	switch typ := ts.Type.(type) {
	case *ast.InterfaceType:
		for _, m := range typ.Methods.List {
			for _, n := range m.Names {
				add(n.Name)
			}
		}
	case *ast.StructType:
		for _, fld := range typ.Fields.List {
			for _, n := range fld.Names {
				add(n.Name)
			}
		}
	}
}

// checkAgentsMDPaths validates the path-like backticked tokens of one
// nested AGENTS.md file.
func checkAgentsMDPaths(t *testing.T, path, rel string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("%s: read: %v", rel, err)
		return
	}
	dir := filepath.Dir(path)
	selfLabel := filepath.Base(dir) + "/"

	for _, m := range agentsPathTokenRE.FindAllStringSubmatch(string(data), -1) {
		token := m[1]
		if !isAgentsPathCandidate(token, selfLabel) {
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, token)); err == nil {
			continue
		}
		if _, err := os.Stat(token); err == nil {
			continue
		}
		t.Errorf("%s: referenced path `%s` does not exist relative to "+
			"the file's directory or the module root", rel, token)
	}
}

// isAgentsPathCandidate reports whether a backticked token looks like a
// file or directory path reference: path-safe characters only, no ".."
// or placeholders, no dots outside the final file extension, and either
// a .go/.md extension, a trailing slash, or an interior slash.
func isAgentsPathCandidate(token, selfLabel string) bool {
	if token == selfLabel {
		return false
	}
	if strings.ContainsAny(token, " <>:*?\"'()[]{}&,;=#") {
		return false
	}
	if strings.Contains(token, "..") {
		return false
	}
	isPathish := strings.HasSuffix(token, ".go") ||
		strings.HasSuffix(token, ".md") ||
		strings.HasSuffix(token, "/") ||
		strings.Contains(token, "/")
	if !isPathish {
		return false
	}
	stem := strings.TrimSuffix(strings.TrimSuffix(token, ".go"), ".md")
	return !strings.Contains(stem, ".")
}
