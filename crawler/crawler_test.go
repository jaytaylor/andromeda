package crawler

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"jaytaylor.com/universe/twilightzone/go/cmd/go/external/cfg"
)

func TestImportsStd(t *testing.T) {
	imports, testImports, err := imports(cfg.BuildContext.GOROOT, "testing")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("imports=%v", imports)
	t.Logf("testImports=%v", testImports)
}

func TestImportsNonStd(t *testing.T) {
	var (
		pkgPath   = "universedynimporttest"
		parentDir = os.TempDir()
		localPath = filepath.Join(parentDir, "src", pkgPath)
	)

	if err := os.RemoveAll(localPath); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(localPath, os.FileMode(int(0700))); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(localPath); err != nil {
			t.Fatal(err)
		}
	}()

	const goMain = `package main

import (
	"fmt"
)

func main() {
	fmt.Println("hello world!")
}`

	if err := ioutil.WriteFile(filepath.Join(localPath, "main.go"), []byte(goMain), os.FileMode(int(0600))); err != nil {
		t.Fatal(err)
	}

	imports, testImports, err := imports(parentDir, pkgPath)
	if err != nil {
		t.Fatal(err)
	}

	if expected, actual := 1, len(imports); actual != expected {
		t.Errorf("Expected number of imports=%v but actual=%v", expected, actual)
	}
	if expected, actual := "fmt", imports[0]; actual != expected {
		t.Errorf("Expected imports=%v but actual=%v", expected, actual)
	}

	t.Logf("imports=%v", imports)
	t.Logf("testImports=%v", testImports)
}
