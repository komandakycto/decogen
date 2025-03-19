// internal/parser/parser.go
package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"

	"github.com/clobucks/decogen/internal/model"
)

// ParseInterface parses a Go source file and extracts the specified interface
func ParseInterface(sourcePath, interfaceName string) (*model.Interface, error) {
	// Set up the file set
	fset := token.NewFileSet()

	// Parse the source file
	file, err := parser.ParseFile(fset, sourcePath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse source file: %w", err)
	}

	// Find the package name
	packageName := file.Name.Name

	// Look for the interface declaration
	var interfaceType *ast.InterfaceType
	var comments *ast.CommentGroup

	// Inspect the file to find our interface
	ast.Inspect(file, func(n ast.Node) bool {
		// Look for general declarations (type, const, var, etc.)
		genDecl, ok := n.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			return true // Continue searching if not a type declaration
		}

		// Look through the specs in this declaration
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != interfaceName {
				continue // Skip if not our target interface
			}

			// Check if it's an interface
			if it, ok := typeSpec.Type.(*ast.InterfaceType); ok {
				interfaceType = it
				comments = genDecl.Doc // Get doc comments from the general declaration
				if comments == nil && typeSpec.Doc != nil {
					comments = typeSpec.Doc // Fallback to typeSpec comments if available
				}
				return false // Stop searching once found
			}
		}

		return true // Continue searching
	})

	// If we didn't find the interface, return an error
	if interfaceType == nil {
		return nil, fmt.Errorf("interface %s not found in %s", interfaceName, sourcePath)
	}

	// Extract imports
	imports := make(map[string]string)
	for _, imp := range file.Imports {
		var name string
		if imp.Name != nil {
			name = imp.Name.Name
		} else {
			path := strings.Trim(imp.Path.Value, "\"")
			name = filepath.Base(path)
		}
		imports[name] = strings.Trim(imp.Path.Value, "\"")
	}

	// Create the interface model
	result := &model.Interface{
		Name:        interfaceName,
		PackageName: packageName,
		Methods:     make([]*model.Method, 0),
		Imports:     imports,
	}

	// Add comments if available
	if comments != nil {
		result.Comments = comments.Text()
	}

	// Extract the methods
	for _, method := range interfaceType.Methods.List {
		// Check if it's a method with a function type
		funcType, ok := method.Type.(*ast.FuncType)
		if !ok {
			continue
		}

		methodModel := &model.Method{
			Name:       method.Names[0].Name,
			Parameters: make([]*model.Parameter, 0),
			Results:    make([]*model.Parameter, 0),
		}

		// Extract method comments if available
		if method.Doc != nil {
			methodModel.Comments = method.Doc.Text()
		} else if method.Comment != nil {
			methodModel.Comments = method.Comment.Text()
		}

		// Extract parameters
		if funcType.Params != nil {
			for i, param := range funcType.Params.List {
				paramType := extractType(param.Type)
				paramNames := make([]string, 0)

				// Extract parameter names
				if len(param.Names) > 0 {
					for _, name := range param.Names {
						paramNames = append(paramNames, name.Name)
					}
				} else {
					// For unnamed parameters, generate a name
					paramNames = append(paramNames, fmt.Sprintf("param%d", i))
				}

				for _, name := range paramNames {
					methodModel.Parameters = append(methodModel.Parameters, &model.Parameter{
						Name: name,
						Type: paramType,
					})
				}
			}
		}

		// Extract results
		if funcType.Results != nil {
			for i, result := range funcType.Results.List {
				resultType := extractType(result.Type)
				resultName := ""

				// Extract result name if available
				if len(result.Names) > 0 {
					resultName = result.Names[0].Name
				} else {
					// For unnamed results, generate a name
					resultName = fmt.Sprintf("result%d", i)
				}

				methodModel.Results = append(methodModel.Results, &model.Parameter{
					Name: resultName,
					Type: resultType,
				})
			}
		}

		result.Methods = append(result.Methods, methodModel)
	}

	return result, nil
}

// extractType extracts a type expression as a string
func extractType(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return fmt.Sprintf("%s.%s", extractType(t.X), t.Sel.Name)
	case *ast.StarExpr:
		return "*" + extractType(t.X)
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + extractType(t.Elt)
		}
		return fmt.Sprintf("[%s]%s", extractType(t.Len), extractType(t.Elt))
	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", extractType(t.Key), extractType(t.Value))
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.FuncType:
		return "func()" // Simplified for brevity
	case *ast.ChanType:
		return "chan" // Simplified for brevity
	case *ast.Ellipsis:
		return "..." + extractType(t.Elt)
	default:
		return fmt.Sprintf("unhandled(%T)", expr)
	}
}
