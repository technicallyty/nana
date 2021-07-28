package nana

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
)

func main(){}

func PatchProtoReflect(path string) error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	for _,v := range f.Decls {
		switch t := v.(type) {
		case *ast.FuncDecl:
			if t.Name.String() == "ProtoReflect" {

				if len(t.Recv.List) < 1 {
					return errors.New("expected receiver on ProtoReflect Method")
				}
				if len(t.Recv.List[0].Names) < 1 {
					return errors.New("expected receiver on ProtoReflect Method")
				}

				rewrite := t.Recv.List[0].Names[0].Name
				exp, err := parser.ParseExpr(rewrite)
				if err != nil {
					return err
				}

				returnThis := ast.ReturnStmt{
					Return:  t.Body.List[len(t.Body.List)-1].Pos(),
					Results: []ast.Expr{exp},
				}
				block := ast.BlockStmt{
					Lbrace: t.Body.Lbrace,
					List:   []ast.Stmt{&returnThis},
					Rbrace: t.Body.Rbrace,
				}
				t.Body = &block
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
