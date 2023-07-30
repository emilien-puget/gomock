package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"text/template"
)

const mockTemplate = `
package mocks

import (
	"github.com/stretchr/testify/mock"
)
{{ range .Mocks }}
type {{ .MockName }} struct {
	mock.Mock
}
{{- range .Methods }}

{{ .MethodCode }}
{{- end }}
{{ end -}}
`

type MethodData struct {
	MethodCode string
}

type TemplateData struct {
	Mocks []Mocks
}

type Mocks struct {
	MockName string
	Methods  []MethodData
}

var errMissingResult = errors.New("result is required")

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run main.go <interface>")
		return
	}

	path := flag.String("result", "", "the path of the generated file, not used if stdout is piped")
	flag.Parse()

	interfaceCode := os.Args[1]

	writer, closer, err := getWriter(path)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer closer()

	err = interfaceToMock(writer, interfaceCode)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func getWriter(path *string) (*bufio.Writer, func(), error) {
	o, _ := os.Stdout.Stat()
	if (o.Mode() & os.ModeCharDevice) == os.ModeCharDevice {
		if path == nil || *path == "" {
			return nil, nil, errMissingResult
		}
		file, err := os.Create(*path)
		if err != nil {
			return nil, nil, err
		}
		writer := bufio.NewWriter(file)
		return writer, func() {
			_ = writer.Flush()
			_ = file.Close()
		}, nil
	}
	writer := bufio.NewWriter(os.Stdout)
	return writer, func() {
		_ = writer.Flush()
	}, nil
}

func interfaceToMock(writer *bufio.Writer, interfaceCode string) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "", "package main\n"+interfaceCode, parser.AllErrors)
	if err != nil {
		return fmt.Errorf("parser.ParseFile: %w", err)
	}

	templateData := TemplateData{Mocks: []Mocks{}}

	for _, decl := range node.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok {
			for _, spec := range genDecl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					switch tp := typeSpec.Type.(type) {
					case *ast.InterfaceType:
						name := typeSpec.Name.String() + "Mock"
						templateData.Mocks = append(templateData.Mocks, Mocks{
							MockName: name,
							Methods:  getMethodsFromInterface(tp, name),
						})
					case *ast.StructType:
						for _, field := range tp.Fields.List {
							if typeSpec, ok := field.Type.(*ast.InterfaceType); ok {
								name := field.Names[0].String() + "Mock"
								templateData.Mocks = append(templateData.Mocks, Mocks{
									MockName: name,
									Methods:  getMethodsFromInterface(typeSpec, name),
								})
							}
						}
					}
				}
			}
		}
	}

	tmpl, err := template.New("mock").Parse(mockTemplate)
	if err != nil {
		return fmt.Errorf("template.New: %w", err)

	}

	err = tmpl.Execute(writer, templateData)
	if err != nil {
		return fmt.Errorf("tmpl.Execute: %w", err)
	}

	return nil
}

func getMethodsFromInterface(interfaceType *ast.InterfaceType, mockName string) []MethodData {
	var methods []MethodData
	for _, method := range interfaceType.Methods.List {
		methodName := method.Names[0].Name
		paramList := getParameters(method.Type.(*ast.FuncType).Params)
		returnList := getParameters(method.Type.(*ast.FuncType).Results)

		methodCode := fmt.Sprintf(
			"func (m *%s) %s(%s) (%s) {\n\targs := m.Called(%s)\n\treturn %s\n}",
			mockName,
			methodName,
			paramList,
			returnList,
			getArgumentNames(paramList),
			getReturnNames(returnList),
		)

		methods = append(methods, MethodData{
			MethodCode: methodCode,
		})
	}
	return methods
}

func getParameters(fieldList *ast.FieldList) string {
	var params []string
	if fieldList != nil {
		for _, field := range fieldList.List {
			if len(field.Names) == 0 {
				params = append(params, getTypeName(field.Type))

				continue
			}
			for _, name := range field.Names {
				params = append(params, fmt.Sprintf("%s %s", name.Name, getTypeName(field.Type)))
			}
		}
	}
	return strings.Join(params, ", ")
}

func getArgumentNames(paramList string) string {
	params := strings.Split(paramList, ", ")
	var argNames []string
	for _, param := range params {
		parts := strings.Fields(param)
		argNames = append(argNames, parts[0])
	}
	return strings.Join(argNames, ", ")
}

func getTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		x, _ := t.X.(*ast.Ident)
		sel := t.Sel.Name
		return fmt.Sprintf("%s.%s", x.Name, sel)
	case *ast.StarExpr:
		return "" + getTypeName(t.X)
	case *ast.ArrayType:
		return "[]" + getTypeName(t.Elt)
	case *ast.MapType:
		key := getTypeName(t.Key)
		value := getTypeName(t.Value)
		return fmt.Sprintf("map[%s]%s", key, value)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.FuncType:
		return "func" + getFunctionSignature(t)
	default:
		return ""
	}
}

func getReturnNames(returnList string) string {
	returnNames := strings.Split(returnList, ", ")
	var names []string
	for i := 0; i < len(returnNames); i++ {
		names = append(names, fmt.Sprintf("args.Get(%d).(%s)", i, getTypeNameFromString(returnNames[i])))
	}
	return strings.Join(names, ", ")
}

func getTypeNameFromString(typeString string) string {
	parts := strings.Fields(typeString)
	return parts[len(parts)-1]
}

func getFunctionSignature(funcType *ast.FuncType) string {
	var params []string
	if funcType.Params != nil {
		params = append(params, getParameters(funcType.Params))
	}
	if funcType.Results != nil {
		params = append(params, getParameters(funcType.Results))
	}
	return fmt.Sprintf("(%s)", strings.Join(params, ", "))
}
