package cardcore

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
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
	testGroupIntegrationTest                  // func Test*Integration or Test*FullGame*
	testGroupHelper                           // Non-Test funcs
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

// TestFunctionOrdering walks every .go file in the module and verifies
// that function declarations follow the ordering conventions described
// in CONTRIBUTING.md.
func TestFunctionOrdering(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			base := d.Name()
			if base == ".git" || base == "vendor" || base == "testdata" || base == "doc" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		// Skip doc.go files — they have no functions.
		if strings.HasSuffix(path, "doc.go") {
			return nil
		}

		rel, _ := filepath.Rel(root, path)

		checkDeclsBeforeFuncs(t, path, rel)
		if strings.HasSuffix(path, "_test.go") {
			checkTestFile(t, path, rel)
		} else {
			checkProdFile(t, path, rel)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir: %v", err)
	}
}

// TestDocComments walks every .go file in the module and verifies that
// every function and method has a doc comment starting with its name.
// For doc.go files, it verifies the package doc comment exists and
// starts with "Package <name>".
func TestDocComments(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			base := d.Name()
			if base == ".git" || base == "vendor" || base == "testdata" || base == "doc" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		rel, _ := filepath.Rel(root, path)

		if strings.HasSuffix(path, "doc.go") {
			checkPackageDoc(t, path, rel)
			return nil
		}

		checkDocComments(t, path, rel)

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

	var funcs []funcInfo
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
					t.Errorf("%s:%d: %s (receiver %s) is separated from %s:%d: %s by %s:%d: %s (receiver %q)",
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
			t.Errorf("%s:%d: %s declaration appears after first function %s (line %d) — declarations must precede all functions",
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
	case !strings.HasPrefix(name, "Test"):
		g = testGroupHelper
	case isIntegrationTestName(name):
		g = testGroupIntegrationTest
	default:
		g = testGroupUnitTest
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
