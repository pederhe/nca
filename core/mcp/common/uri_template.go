package common

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// Maximum allowed sizes for security
const (
	MaxTemplateLength      = 1000000 // 1MB
	MaxVariableLength      = 1000000 // 1MB
	MaxTemplateExpressions = 10000
	MaxRegexLength         = 1000000 // 1MB
)

// Variables is a map of variable names to their values
type Variables map[string]interface{}

// Part represents a part of a URI template
type Part struct {
	Name     string
	Operator string
	Names    []string
	Exploded bool
	Raw      string // Raw text if it's not a template part
	IsText   bool   // Whether this is raw text or a template part
}

// UriTemplate implements RFC 6570 URI Templates
type UriTemplate struct {
	template string
	parts    []Part
}

// NewUriTemplate creates a new URI template from a string
func NewUriTemplate(template string) (*UriTemplate, error) {
	if len(template) > MaxTemplateLength {
		return nil, fmt.Errorf("template exceeds maximum length of %d characters (got %d)", MaxTemplateLength, len(template))
	}

	parts, err := parseTemplate(template)
	if err != nil {
		return nil, err
	}

	return &UriTemplate{
		template: template,
		parts:    parts,
	}, nil
}

// IsTemplate returns true if the given string contains any URI template expressions
// A template expression is a sequence of characters enclosed in curly braces,
// like {foo} or {?bar}
func IsTemplate(str string) bool {
	re := regexp.MustCompile(`\{[^}\s]+\}`)
	return re.MatchString(str)
}

// String returns the original template string
func (ut *UriTemplate) String() string {
	return ut.template
}

// parseTemplate breaks a template string into its component parts
func parseTemplate(template string) ([]Part, error) {
	var parts []Part
	currentText := ""
	i := 0
	expressionCount := 0

	for i < len(template) {
		if template[i] == '{' {
			if currentText != "" {
				parts = append(parts, Part{
					Raw:    currentText,
					IsText: true,
				})
				currentText = ""
			}

			end := strings.Index(template[i:], "}")
			if end == -1 {
				return nil, errors.New("unclosed template expression")
			}
			end += i // Adjust end position to be relative to the start of the full string

			expressionCount++
			if expressionCount > MaxTemplateExpressions {
				return nil, fmt.Errorf("template contains too many expressions (max %d)", MaxTemplateExpressions)
			}

			expr := template[i+1 : end]
			operator := getOperator(expr)
			exploded := strings.Contains(expr, "*")
			names := getNames(expr, operator)

			if len(names) == 0 {
				return nil, errors.New("empty template expression")
			}
			name := names[0]

			// Validate variable name length
			for _, n := range names {
				if len(n) > MaxVariableLength {
					return nil, fmt.Errorf("variable name exceeds maximum length of %d characters (got %d)", MaxVariableLength, len(n))
				}
			}

			parts = append(parts, Part{
				Name:     name,
				Operator: operator,
				Names:    names,
				Exploded: exploded,
				IsText:   false,
			})
			i = end + 1
		} else {
			currentText += string(template[i])
			i++
		}
	}

	if currentText != "" {
		parts = append(parts, Part{
			Raw:    currentText,
			IsText: true,
		})
	}

	return parts, nil
}

// getOperator extracts the operator from a template expression
func getOperator(expr string) string {
	operators := []string{"+", "#", ".", "/", "?", "&"}
	for _, op := range operators {
		if strings.HasPrefix(expr, op) {
			return op
		}
	}
	return ""
}

// getNames extracts variable names from a template expression
func getNames(expr, operator string) []string {
	expr = expr[len(operator):]
	names := strings.Split(expr, ",")
	var result []string

	for _, name := range names {
		name = strings.Replace(name, "*", "", -1)
		name = strings.TrimSpace(name)
		if name != "" {
			result = append(result, name)
		}
	}

	return result
}

// encodeValue encodes a value according to the operator's rules
func encodeValue(value string, operator string) string {
	if len(value) > MaxVariableLength {
		// Truncate the value if it's too long
		value = value[:MaxVariableLength]
	}

	if operator == "+" || operator == "#" {
		return url.PathEscape(value)
	}
	return url.QueryEscape(value)
}

// Expand replaces variables in the template with their values
func (ut *UriTemplate) Expand(variables Variables) (string, error) {
	var result strings.Builder
	hasQueryParam := false

	for _, part := range ut.parts {
		if part.IsText {
			result.WriteString(part.Raw)
			continue
		}

		expanded, err := expandPart(part, variables)
		if err != nil {
			return "", err
		}

		if expanded == "" {
			continue
		}

		// Convert ? to & if we already have a query parameter
		if part.Operator == "?" && hasQueryParam {
			result.WriteString("&")
			result.WriteString(expanded[1:]) // Skip the leading ?
		} else {
			result.WriteString(expanded)
			if part.Operator == "?" {
				hasQueryParam = true
			}
		}
	}

	return result.String(), nil
}

// expandPart expands a template part
func expandPart(part Part, variables Variables) (string, error) {
	if part.Operator == "?" || part.Operator == "&" {
		var pairs []string
		for _, name := range part.Names {
			value, ok := variables[name]
			if !ok {
				continue
			}

			var encoded string
			switch v := value.(type) {
			case []string:
				var encodedValues []string
				for _, val := range v {
					encodedValues = append(encodedValues, encodeValue(val, part.Operator))
				}
				encoded = strings.Join(encodedValues, ",")
			case string:
				encoded = encodeValue(v, part.Operator)
			default:
				encoded = encodeValue(fmt.Sprintf("%v", v), part.Operator)
			}

			pairs = append(pairs, name+"="+encoded)
		}

		if len(pairs) == 0 {
			return "", nil
		}
		separator := part.Operator
		return separator + strings.Join(pairs, "&"), nil
	}

	if len(part.Names) > 1 {
		var values []string
		for _, name := range part.Names {
			value, ok := variables[name]
			if !ok {
				continue
			}

			switch v := value.(type) {
			case []string:
				if len(v) > 0 {
					values = append(values, v[0])
				}
			case string:
				values = append(values, v)
			default:
				values = append(values, fmt.Sprintf("%v", v))
			}
		}

		if len(values) == 0 {
			return "", nil
		}
		return strings.Join(values, ","), nil
	}

	// Single variable case
	value, ok := variables[part.Name]
	if !ok {
		return "", nil
	}

	var values []string
	switch v := value.(type) {
	case []string:
		values = v
	case string:
		values = []string{v}
	default:
		values = []string{fmt.Sprintf("%v", v)}
	}

	var encoded []string
	for _, v := range values {
		encoded = append(encoded, encodeValue(v, part.Operator))
	}

	switch part.Operator {
	case "":
		return strings.Join(encoded, ","), nil
	case "+":
		return strings.Join(encoded, ","), nil
	case "#":
		return "#" + strings.Join(encoded, ","), nil
	case ".":
		return "." + strings.Join(encoded, "."), nil
	case "/":
		return "/" + strings.Join(encoded, "/"), nil
	default:
		return strings.Join(encoded, ","), nil
	}
}

// Match extracts variables from a URI according to the template
func (ut *UriTemplate) Match(uri string) (Variables, error) {
	// This implementation is a simplified version that handles only basic templates
	variables := make(Variables)

	// Build a regex from the template
	regexStr, names, err := ut.buildRegex()
	if err != nil {
		return nil, err
	}

	// Compile the regex
	regex, err := regexp.Compile(regexStr)
	if err != nil {
		return nil, fmt.Errorf("failed to compile template regex: %w", err)
	}

	// Match the URI against the regex
	matches := regex.FindStringSubmatch(uri)
	if matches == nil {
		return nil, nil
	}

	// Extract the variables
	for i, name := range names {
		if i+1 < len(matches) {
			variables[name] = matches[i+1]
		}
	}

	return variables, nil
}

// buildRegex converts a URI template to a regex pattern that can extract variables
func (ut *UriTemplate) buildRegex() (string, []string, error) {
	var pattern strings.Builder
	var names []string

	pattern.WriteString("^")

	for _, part := range ut.parts {
		if part.IsText {
			pattern.WriteString(regexp.QuoteMeta(part.Raw))
			continue
		}

		// Handle different operators
		switch part.Operator {
		case "":
			for _, name := range part.Names {
				pattern.WriteString("([^/,]+)")
				names = append(names, name)
			}
		case "+":
			for _, name := range part.Names {
				pattern.WriteString("(.+)")
				names = append(names, name)
			}
		case "#":
			pattern.WriteString("#")
			for _, name := range part.Names {
				pattern.WriteString("([^,]+)")
				names = append(names, name)
			}
		case ".":
			pattern.WriteString("\\.")
			for _, name := range part.Names {
				pattern.WriteString("([^./]+)")
				names = append(names, name)
			}
		case "/":
			pattern.WriteString("/")
			for _, name := range part.Names {
				pattern.WriteString("([^/]+)")
				names = append(names, name)
			}
		case "?":
			pattern.WriteString("\\?")
			for i, name := range part.Names {
				if i > 0 {
					pattern.WriteString("&")
				}
				pattern.WriteString(regexp.QuoteMeta(name))
				pattern.WriteString("=([^&]*)")
				names = append(names, name)
			}
		case "&":
			pattern.WriteString("&")
			for _, name := range part.Names {
				pattern.WriteString(regexp.QuoteMeta(name))
				pattern.WriteString("=([^&]*)")
				names = append(names, name)
			}
		}
	}

	pattern.WriteString("$")

	if pattern.Len() > MaxRegexLength {
		return "", nil, fmt.Errorf("regex pattern exceeds maximum length of %d", MaxRegexLength)
	}

	return pattern.String(), names, nil
}
