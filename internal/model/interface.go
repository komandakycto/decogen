package model

import (
	"fmt"
	"strings"
)

// Interface represents a parsed Go interface
type Interface struct {
	Name        string
	PackageName string
	Methods     []*Method
	Comments    string
	Imports     map[string]string
}

// Method represents a method in an interface
type Method struct {
	Name       string
	Parameters []*Parameter
	Results    []*Parameter
	Comments   string
}

// Parameter represents a parameter or result in a method
type Parameter struct {
	Name string
	Type string
}

// FormatMethodSignature formats a method signature for code generation
func (m *Method) FormatMethodSignature() string {
	var params []string
	for _, p := range m.Parameters {
		params = append(params, fmt.Sprintf("%s %s", p.Name, p.Type))
	}

	var results []string
	for _, r := range m.Results {
		results = append(results, r.Type)
	}

	resultStr := ""
	if len(results) == 1 {
		resultStr = results[0]
	} else if len(results) > 1 {
		resultStr = fmt.Sprintf("(%s)", strings.Join(results, ", "))
	}

	return fmt.Sprintf("%s(%s) %s", m.Name, strings.Join(params, ", "), resultStr)
}

// FormatMethodCall formats a method call for the underlying implementation
func (m *Method) FormatMethodCall() string {
	var params []string
	for _, p := range m.Parameters {
		params = append(params, p.Name)
	}

	return fmt.Sprintf("%s(%s)", m.Name, strings.Join(params, ", "))
}

// HasReturnValue checks if the method has at least one return value
func (m *Method) HasReturnValue() bool {
	return len(m.Results) > 0
}

// HasErrorReturn checks if the method returns an error (common in Go)
func (m *Method) HasErrorReturn() bool {
	if len(m.Results) == 0 {
		return false
	}

	lastResult := m.Results[len(m.Results)-1]
	return lastResult.Type == "error"
}

// FormatResultDeclarations generates variable declarations for results
func (m *Method) FormatResultDeclarations() string {
	if !m.HasReturnValue() {
		return ""
	}

	var decls []string
	for _, r := range m.Results {
		if r.Type == "error" {
			continue // We'll handle errors separately
		}
		decls = append(decls, fmt.Sprintf("var %s %s", r.Name, r.Type))
	}

	if len(decls) == 0 {
		return ""
	}

	return strings.Join(decls, "\n\t")
}

// FormatResultReturn formats the return statement
func (m *Method) FormatResultReturn(errorVar string) string {
	if !m.HasReturnValue() {
		return "return"
	}

	var returns []string
	for _, r := range m.Results {
		if r.Type == "error" {
			returns = append(returns, errorVar)
		} else {
			returns = append(returns, r.Name)
		}
	}

	return fmt.Sprintf("return %s", strings.Join(returns, ", "))
}

// FormatContextParam returns the context parameter name if one exists
func (m *Method) FormatContextParam() string {
	for _, p := range m.Parameters {
		if p.Type == "context.Context" {
			return p.Name
		}
	}
	return ""
}
