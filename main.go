package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"text/template"

	"github.com/pkg/errors"
)

var (
	constType = "int"
)

func main() {
	file := os.Getenv("GOFILE")

	// 保存注释信息
	var comments = make(map[string]string)
	var codemsg = make(map[int]string)

	// 解析代码源文件，获取常量和注释之间的关系
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
	checkErr(err)

	// Create an ast.CommentMap from the ast.File's comments.
	// This helps keeping the association between comments
	// and AST nodes.
	commentMap := ast.NewCommentMap(fset, f, f.Comments)
	for node := range commentMap {
		// 仅支持一条声明语句，一个常量的情况, 即不支持批量赋值
		if spec, ok := node.(*ast.ValueSpec); ok && len(spec.Names) == 1 {
			// 仅提取常量的注释
			ident := spec.Names[0]
			value := spec.Values[0].(*ast.BasicLit).Value
			valueInt, err := strconv.Atoi(value)
			checkErr(err)
			if ident.Obj.Kind == ast.Con {
				// 获取注释信息
				cmt := getComment(ident.Name, spec.Doc)
				comments[ident.Name] = cmt
				codemsg[valueInt] = cmt
			}
		}
	}

	code, err := genCode(comments)
	checkErr(err)

	// 生成代码文件
	out := strings.TrimSuffix(file, ".go") + suffix + ".go"
	checkErr(ioutil.WriteFile(out, code, 0644))

	code, _ = genJSON(codemsg)
	// 生成json
	out = strings.TrimSuffix(file, ".go") + suffix + ".json"
	checkErr(ioutil.WriteFile(out, code, 0644))

}

// getComment 获取注释信息
func getComment(name string, group *ast.CommentGroup) string {
	var buf bytes.Buffer

	// collect comments text
	// Note: CommentGroup.Text() does too much work for what we
	//       need and would only replace this innermost loop.
	//       Just do it explicitly.
	for _, comment := range group.List {
		text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
		buf.WriteString(text)
	}

	// replace any invisibles with blanks
	//bs := buf.Bytes()
	//for i, b := range bs {
	//	switch b {
	//	case '\t', '\n', '\r':
	//		bs[i] = ' '
	//	}
	//}
	//return string(bs)
	return string(buf.Bytes())
}

func genCode(comments map[string]string) ([]byte, error) {
	buf := bytes.NewBufferString("")

	file := os.Getenv("GOFILE")
	data := map[string]interface{}{
		"pkg":       os.Getenv("GOPACKAGE"),
		"source":    file,
		"file":      strings.TrimSuffix(file, ".go") + suffix + ".go",
		"comments":  comments,
		"constType": constType,
	}

	t, err := template.New("").Parse(tpl)
	if err != nil {
		return nil, errors.Wrapf(err, "template init err")
	}

	err = t.Execute(buf, data)
	if err != nil {
		return nil, errors.Wrapf(err, "template data err")
	}

	return format.Source(buf.Bytes())
}
func genJSON(m map[int]string) ([]byte, error) {
	var buf bytes.Buffer
	first := true
	fmt.Fprint(&buf, "{\n")
	for k, v := range m {
		if first {
			first = false
		} else {
			fmt.Fprint(&buf, ",\n")
		}
		fmt.Fprintf(&buf, `%d:"%s"`, k, v)
	}
	fmt.Fprint(&buf, "\n}\n")
	return buf.Bytes(), nil
}
func checkErr(err error) {
	if err != nil {
		panic(fmt.Sprintf("err: %+v", err))
	}
}
