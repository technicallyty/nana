package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
)

func main() {
	patcher := flag.NewFlagSet("delete", flag.ExitOnError)
	funcName := patcher.String("func", "", "function to patch -- example: -func nameOfFunc")
	fileName := patcher.String("file", "", "the path to the file to delete -- example: -file path/to/file")
	if len(os.Args) < 3 {
		fmt.Println("not enough arguments passed")
		os.Exit(51)
	}
	if os.Args[1] != "delete" {
		fmt.Sprintf("command '%s' not recognized", os.Args[1])
		os.Exit(50)
	}
	patcher.Parse(os.Args[2:])
	err := RemoveFunction(*fileName, *funcName)
	if err != nil {
		panic(err)
	}
}

func RemoveFunction(path, function string) error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	for i,v := range f.Decls {
		switch t := v.(type) {
		case *ast.FuncDecl:
			if t.Name.String() == function {
				// delete all protoreflect methods so we can make our own
				if i != len(f.Decls)-1 {
					f.Decls[i] = f.Decls[len(f.Decls)-1]
				}
				// drop the last element
				f.Decls =  f.Decls[:len(f.Decls)-1]
				//if len(t.Recv.List) < 1 {
				//	return errors.New("expected receiver on ProtoReflect Method")
				//}
				//if len(t.Recv.List[0].Names) < 1 {
				//	return errors.New("expected receiver on ProtoReflect Method")
				//}
				//
				//rewrite := t.Recv.List[0].Names[0].Name
				//exp, err := parser.ParseExpr(rewrite)
				//if err != nil {
				//	return err
				//}
				//
				//returnThis := ast.ReturnStmt{
				//	Return:  t.Body.List[len(t.Body.List)-1].Pos(),
				//	Results: []ast.Expr{exp},
				//}
				//block := ast.BlockStmt{
				//	Lbrace: t.Body.Lbrace,
				//	List:   []ast.Stmt{&returnThis},
				//	Rbrace: t.Body.Rbrace,
				//}
				//t.Body = &block
			}
		}
	}

	buf := getBytesFromFile(fset, f)
	err = os.Remove(path)
	if err != nil {
		return err
	}
	err = os.WriteFile(path, buf.Bytes(), 0776)
	if err != nil {
		return err
	}

	return nil
}

func Patch(src string, paths... string) (*ast.File, *token.FileSet, error) {
	srcFset := token.NewFileSet()
	srcF, err := parser.ParseFile(srcFset, src, nil, parser.ParseComments)
	if err != nil {
		return nil,nil, err
	}

	importMap := make(map[string]*ast.ImportSpec)
	for _,imp := range srcF.Imports {
		importMap[imp.Path.Value] = imp
	}

	for _,path := range paths {
		f, err := parser.ParseFile(srcFset, path, nil, parser.ParseComments)
		if err != nil {
			return nil,nil, err
		}

		importMap = patchImports(f.Imports, importMap)

		err = patchScope(srcF.Scope, f.Scope)
		if err != nil {
			return nil,nil, err
		}

		mergedDecls, err := patchDecls(srcF.Decls, f.Decls)
		if err != nil {
			return nil, nil, err
		}

		srcF.Decls = mergedDecls
		os.Remove(path)
	}

	os.Remove(src)
	bz := getBytesFromFile(srcFset, srcF)
	os.WriteFile(src, bz.Bytes(), 0766)

	return  srcF,srcFset, nil
}

func patchImports(imports []*ast.ImportSpec, importMap map[string]*ast.ImportSpec) map[string]*ast.ImportSpec {
	for _, imp := range imports {
		iport, ok := importMap[imp.Path.Value]
		if !ok {
			importMap[imp.Path.Value] = iport
		}
	}

	return importMap
}


func patchScope(src,dst *ast.Scope) error {
	for k,v := range dst.Objects {
		if _, ok := src.Objects[k]; ok {
			return fmt.Errorf("duplicate scope found %s", k)
		}
		src.Objects[k] = v
	}
	return nil
}

func patchDecls(src, dst []ast.Decl) ([]ast.Decl, error) {
	patched := make([]ast.Decl, 0, len(src) + len(dst))
	patched = append(patched, src...)

	dstImports, ok := dst[0].(*ast.GenDecl)
	if !ok {
		return nil, errors.New("destination did not have imports")
	}
	srcImports, ok  := src[0].(*ast.GenDecl)
	if !ok {
		return nil, errors.New("src did not have imports")
	}
	srcImports.Specs = append(srcImports.Specs, dstImports.Specs...)

	src[0] = srcImports
	dst = dst[1:]
	src = append(src, dst...)

	return src, nil
}

func getBytesFromFile(set *token.FileSet, file *ast.File) *bytes.Buffer {
	var buf bytes.Buffer
	err := printer.Fprint(&buf, set, file)
	if err != nil {
		panic(err)
	}
	return &buf
}
