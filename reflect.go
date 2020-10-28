package main

import (
	"bytes"
	"errors"
	"fmt"
	"go/build"
	"golang.org/x/mod/modfile"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
)

func writeProgram(importPath string, symbols []string, comments map[string]string) ([]byte, error) {
	var program bytes.Buffer
	data := reflectData{
		ImportPath:   importPath,
		Symbols:      symbols,
		JsonFilename: strings.TrimSuffix(os.Getenv("GOFILE"), ".go") + suffix + ".json",
		Comments:     comments,
	}
	if err := reflectProgram.Execute(&program, &data); err != nil {
		return nil, err
	}
	return program.Bytes(), nil
}

// run the given program and parse the output as a model.Package.
func run(program string) error {
	f, err := ioutil.TempFile("", "")
	if err != nil {
		return err
	}

	filename := f.Name()
	defer os.Remove(filename)
	if err := f.Close(); err != nil {
		return err
	}

	// Run the program.
	cmd := exec.Command(program, "-output", filename)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	f, err = os.Open(filename)
	if err != nil {
		return err
	}

	if err := f.Close(); err != nil {
		return err
	}

	return nil
}

// runInDir writes the given program into the given dir, runs it there, and
// parses the output as a model.Package.
func runInDir(program []byte, dir string) error {
	// We use TempDir instead of TempFile so we can control the filename.
	tmpDir, err := ioutil.TempDir(dir, "codemsg_reflect_")
	if err != nil {
		return err
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			log.Printf("failed to remove temp directory: %s", err)
		}
	}()
	const progSource = "prog.go"
	var progBinary = "prog.bin"
	if runtime.GOOS == "windows" {
		// Windows won't execute a program unless it has a ".exe" suffix.
		progBinary += ".exe"
	}

	if err := ioutil.WriteFile(filepath.Join(tmpDir, progSource), program, 0600); err != nil {
		return err
	}

	cmdArgs := []string{}
	cmdArgs = append(cmdArgs, "build")
	cmdArgs = append(cmdArgs, "-o", progBinary, progSource)

	// Build the program.
	cmd := exec.Command("go", cmdArgs...)
	cmd.Dir = tmpDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	return run(filepath.Join(tmpDir, progBinary))
}

// packageNameOfDir get package import path via dir
func packageNameOfDir(srcDir string) (string, error) {
	fmt.Println("src dir: ", srcDir)
	wd, _ := os.Getwd()
	fmt.Println("working dir: ", wd)
	files, err := ioutil.ReadDir(srcDir)
	checkErr(err)

	var goFilePath string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".go") {
			goFilePath = file.Name()
			break
		}
	}
	if goFilePath == "" {
		return "", fmt.Errorf("go source file not found %s", srcDir)
	}

	packageImport, err := parsePackageImport(srcDir)
	if err != nil {
		return "", err
	}
	fmt.Println("package import: ", packageImport)
	return packageImport, nil
}

// parseImportPackage get package import path via source file
// an alternative implementation is to use:
// cfg := &packages.Config{Mode: packages.NeedName, Tests: true, Dir: srcDir}
// pkgs, err := packages.Load(cfg, "file="+source)
// However, it will call "go list" and slow down the performance
func parsePackageImport(srcDir string) (string, error) {
	moduleMode := os.Getenv("GO111MODULE")
	// trying to find the module
	if moduleMode != "off" {
		currentDir := srcDir
		for {
			dat, err := ioutil.ReadFile(filepath.Join(currentDir, "go.mod"))
			if os.IsNotExist(err) {
				if currentDir == filepath.Dir(currentDir) {
					// at the root
					break
				}
				currentDir = filepath.Dir(currentDir)
				continue
			} else if err != nil {
				return "", err
			}
			modulePath := modfile.ModulePath(dat)
			return filepath.ToSlash(filepath.Join(modulePath, strings.TrimPrefix(srcDir, currentDir))), nil
		}
	}
	// fall back to GOPATH mode
	goPaths := os.Getenv("GOPATH")
	if goPaths == "" {
		return "", fmt.Errorf("GOPATH is not set")
	}
	goPathList := strings.Split(goPaths, string(os.PathListSeparator))
	for _, goPath := range goPathList {
		sourceRoot := filepath.Join(goPath, "src") + string(os.PathSeparator)
		if strings.HasPrefix(srcDir, sourceRoot) {
			return filepath.ToSlash(strings.TrimPrefix(srcDir, sourceRoot)), nil
		}
	}
	return "", errors.New("Source directory is outside GOPATH")
}

// reflectMode generates mocks via reflection on an interface.
func reflectMode(importPath string, symbols []string, comments map[string]string) error {
	// TODO: sanity check arguments

	program, err := writeProgram(importPath, symbols, comments)
	if err != nil {
		return err
	}

	wd, _ := os.Getwd()

	// Try to run the reflection program  in the current working directory.
	if err := runInDir(program, wd); err == nil {
		return nil
	}

	// Try to run the program in the same directory as the input package.
	if p, err := build.Import(importPath, wd, build.FindOnly); err == nil {
		dir := p.Dir
		if err := runInDir(program, dir); err == nil {
			return nil
		}
	}

	// Try to run it in a standard temp directory.
	return runInDir(program, "")
}

type reflectData struct {
	ImportPath   string
	Symbols      []string
	JsonFilename string
	Comments     map[string]string
}

// This program reflects on an interface value, and prints the
// gob encoding of a model.Package to standard output.
// JSON doesn't work because of the model.Type interface.
var reflectProgram = template.Must(template.New("program").Parse(`
package main
import (
	"bytes"
	"fmt"
	"io/ioutil"
	"reflect"
	"sort"
	{{if .ImportPath}}
	pkg_ {{printf "%q" .ImportPath}}
	{{end}}
)
var output = "{{.JsonFilename}}"

// messages get msg from const comment
var messages = map[string]string{
	{{range $key, $value := .Comments}}
	"{{$key}}": "{{$value}}",{{end}}
}

func genJSON(m map[int]string) ([]byte, error) {
	var buf bytes.Buffer
	first := true
	fmt.Fprint(&buf, "{\n")
	// keep map keys in order
	var keys []int
	for k, _ := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		if first {
			first = false
		} else {
			fmt.Fprint(&buf, ",\n")
		}
		fmt.Fprintf(&buf, "    %d: \"%s\"", k, m[k])
	}
	fmt.Fprint(&buf, "\n}\n")
	return buf.Bytes(), nil
}

func checkErr(err error) {
	if err != nil {
		panic(fmt.Sprintf("err: %+v", err))
	}
}

func main() {
	var codemsg = make(map[int]string)
	its := []struct{
		sym string
		typ reflect.Value
	}{
	
		{{range .Symbols}}
			{{if $.ImportPath}}
			{ {{printf "%q" .}}, reflect.ValueOf((pkg_.{{.}}))},
			{{else}}
			{ {{printf "%q" .}}, reflect.ValueOf(({{.}}))},
			{{end}}
		{{end}}
	}
	
	for _, it := range its {
		codemsg[int(it.typ.Int())] = messages[it.sym]
	}

	code, err:=genJSON(codemsg)
	if err != nil {
		panic(err)
	}
	// 生成json
	checkErr(ioutil.WriteFile(output, code, 0644))

}
`))
