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
			if ident.Obj.Kind == ast.Con {
				// 获取注释信息
				comments[ident.Name] = getComment(ident.Name, spec.Doc)
			}
		}
	}

	code, err := genCode(comments)
	checkErr(err)

	// 生成代码文件
	out := strings.TrimSuffix(file, ".go") + suffix + ".go"
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

func checkErr(err error) {
	if err != nil {
		panic(fmt.Sprintf("err: %+v", err))
	}
}
