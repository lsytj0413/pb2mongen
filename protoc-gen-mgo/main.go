package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/pluginpb"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", filepath.Base(os.Args[0]), err)
		os.Exit(1)
	}
}

func run() error {
	in, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return err
	}

	cmd := exec.Command("protoc-gen-go")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	_, err = io.WriteString(stdin, string(in))
	if err != nil {
		return err
	}
	stdin.Close()

	stdout, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}

	resp := &pluginpb.CodeGeneratorResponse{}
	err = proto.Unmarshal(stdout, resp)
	if err != nil {
		return err
	}

	if err = updateBsonTag(resp); err != nil {
		return err
	}

	stdout, err = proto.Marshal(resp)
	if err != nil {
		return err
	}

	if _, err = os.Stdout.Write([]byte(stdout)); err != nil {
		return err
	}

	return nil
}

func updateBsonTag(resp *pluginpb.CodeGeneratorResponse) error {
	if resp == nil || resp.Error != nil {
		return nil
	}

	updateStructType := func(n *ast.StructType) {
		if n.Fields == nil {
			return
		}

		for _, field := range n.Fields.List {
			if field == nil || field.Tag == nil {
				continue
			}

			tag, err := strconv.Unquote(field.Tag.Value)
			if err != nil {
				continue
			}

			if _, ok := reflect.StructTag(tag).Lookup("protobuf"); !ok {
				continue
			}

			v, ok := reflect.StructTag(tag).Lookup("json")
			if !ok {
				continue
			}

			tags := strings.Split(v, ",")
			if len(tags) > 0 {
				field.Tag.Value = fmt.Sprintf("`%s bson:%q`", tag, tags[0])
			}
		}
	}

	updateGenDeclTableStruct := func(n *ast.GenDecl) {
		for _, spec := range n.Specs {
			if n, ok := spec.(*ast.TypeSpec); ok {
				structType, ok := n.Type.(*ast.StructType)
				if !ok {
					continue
				}

				updateStructType(structType)
			}
		}
	}

	for _, file := range resp.GetFile() {
		if file == nil || file.Content == nil {
			continue
		}

		content := file.GetContent()
		fs := token.NewFileSet()
		f, err := parser.ParseFile(fs, file.GetName(), content, parser.ParseComments)
		if err != nil {
			return err
		}

		for _, decl := range f.Decls {
			var n ast.Node = decl
			if n, ok := n.(*ast.GenDecl); ok {
				updateGenDeclTableStruct(n)
			}
		}

		var buf bytes.Buffer
		err = format.Node(&buf, fs, f)
		if err != nil {
			return err
		}

		content = buf.String()
		file.Content = &content
	}

	return nil
}
