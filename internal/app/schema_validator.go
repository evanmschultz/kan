package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// SchemaValidationError describes a deterministic schema-validation failure.
type SchemaValidationError struct {
	Path    string
	Message string
}

// Error renders the schema-validation failure.
func (e SchemaValidationError) Error() string {
	path := strings.TrimSpace(e.Path)
	if path == "" {
		path = "$"
	}
	return fmt.Sprintf("%s: %s", path, e.Message)
}

// jsonSchemaValidator validates payloads against a compiled subset of JSON Schema.
type jsonSchemaValidator struct {
	root *jsonSchemaNode
}

// jsonSchemaNode represents one compiled schema node.
type jsonSchemaNode struct {
	typ                 string
	required            []string
	requiredSet         map[string]struct{}
	properties          map[string]*jsonSchemaNode
	allowAdditional     bool
	enum                []any
	items               *jsonSchemaNode
	minLength           *int
	maxLength           *int
}

// compileJSONSchema compiles a schema string into a reusable validator.
func compileJSONSchema(raw string) (*jsonSchemaValidator, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var decoded any
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil, err
	}
	root, err := compileJSONSchemaNode(decoded, "$")
	if err != nil {
		return nil, err
	}
	return &jsonSchemaValidator{root: root}, nil
}

// ValidatePayload validates raw JSON payload bytes against the compiled schema.
func (v *jsonSchemaValidator) ValidatePayload(payload json.RawMessage) error {
	if v == nil || v.root == nil {
		return nil
	}
	payload = bytes.TrimSpace(payload)
	if len(payload) == 0 {
		payload = []byte("{}")
	}
	var decoded any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return SchemaValidationError{Path: "$", Message: fmt.Sprintf("invalid JSON payload: %v", err)}
	}
	return validateJSONSchemaNode(v.root, decoded, "$")
}

// compileJSONSchemaNode compiles one schema node recursively.
func compileJSONSchemaNode(raw any, path string) (*jsonSchemaNode, error) {
	obj, ok := raw.(map[string]any)
	if !ok {
		return nil, SchemaValidationError{Path: path, Message: "schema must be an object"}
	}

	node := &jsonSchemaNode{
		typ:             "",
		required:        []string{},
		requiredSet:     map[string]struct{}{},
		properties:      map[string]*jsonSchemaNode{},
		allowAdditional: true,
		enum:            nil,
	}

	if rawType, ok := obj["type"]; ok {
		typeText, ok := rawType.(string)
		if !ok {
			return nil, SchemaValidationError{Path: path + ".type", Message: "must be a string"}
		}
		node.typ = strings.TrimSpace(strings.ToLower(typeText))
		switch node.typ {
		case "object", "array", "string", "number", "integer", "boolean", "null":
		default:
			return nil, SchemaValidationError{Path: path + ".type", Message: fmt.Sprintf("unsupported type %q", node.typ)}
		}
	}

	if rawRequired, ok := obj["required"]; ok {
		requiredList, ok := rawRequired.([]any)
		if !ok {
			return nil, SchemaValidationError{Path: path + ".required", Message: "must be an array"}
		}
		for idx, item := range requiredList {
			field, ok := item.(string)
			if !ok {
				return nil, SchemaValidationError{Path: fmt.Sprintf("%s.required[%d]", path, idx), Message: "must be a string"}
			}
			field = strings.TrimSpace(field)
			if field == "" {
				return nil, SchemaValidationError{Path: fmt.Sprintf("%s.required[%d]", path, idx), Message: "must not be empty"}
			}
			if _, exists := node.requiredSet[field]; exists {
				continue
			}
			node.required = append(node.required, field)
			node.requiredSet[field] = struct{}{}
		}
		sort.Strings(node.required)
	}

	if rawProps, ok := obj["properties"]; ok {
		props, ok := rawProps.(map[string]any)
		if !ok {
			return nil, SchemaValidationError{Path: path + ".properties", Message: "must be an object"}
		}
		keys := make([]string, 0, len(props))
		for key := range props {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			compiled, err := compileJSONSchemaNode(props[key], path+".properties."+key)
			if err != nil {
				return nil, err
			}
			node.properties[key] = compiled
		}
	}

	if rawAdditional, ok := obj["additionalProperties"]; ok {
		allow, ok := rawAdditional.(bool)
		if !ok {
			return nil, SchemaValidationError{Path: path + ".additionalProperties", Message: "must be a boolean"}
		}
		node.allowAdditional = allow
	}

	if rawEnum, ok := obj["enum"]; ok {
		enumList, ok := rawEnum.([]any)
		if !ok {
			return nil, SchemaValidationError{Path: path + ".enum", Message: "must be an array"}
		}
		node.enum = append([]any(nil), enumList...)
	}

	if rawItems, ok := obj["items"]; ok {
		compiled, err := compileJSONSchemaNode(rawItems, path+".items")
		if err != nil {
			return nil, err
		}
		node.items = compiled
	}

	if rawMin, ok := obj["minLength"]; ok {
		min, err := parseSchemaInt(rawMin)
		if err != nil {
			return nil, SchemaValidationError{Path: path + ".minLength", Message: err.Error()}
		}
		node.minLength = &min
	}
	if rawMax, ok := obj["maxLength"]; ok {
		max, err := parseSchemaInt(rawMax)
		if err != nil {
			return nil, SchemaValidationError{Path: path + ".maxLength", Message: err.Error()}
		}
		node.maxLength = &max
	}

	return node, nil
}

// parseSchemaInt converts JSON number values into ints for schema bounds.
func parseSchemaInt(raw any) (int, error) {
	switch value := raw.(type) {
	case float64:
		if value < 0 {
			return 0, fmt.Errorf("must be >= 0")
		}
		if value != float64(int(value)) {
			return 0, fmt.Errorf("must be an integer")
		}
		return int(value), nil
	case int:
		if value < 0 {
			return 0, fmt.Errorf("must be >= 0")
		}
		return value, nil
	default:
		return 0, fmt.Errorf("must be a number")
	}
}

// validateJSONSchemaNode validates one value against one compiled schema node.
func validateJSONSchemaNode(node *jsonSchemaNode, value any, path string) error {
	if node == nil {
		return nil
	}

	if len(node.enum) > 0 {
		matched := false
		for _, candidate := range node.enum {
			if reflect.DeepEqual(candidate, value) {
				matched = true
				break
			}
		}
		if !matched {
			return SchemaValidationError{Path: path, Message: "value is not in enum set"}
		}
	}

	switch node.typ {
	case "", "object":
		obj, ok := value.(map[string]any)
		if !ok {
			if node.typ == "" {
				return nil
			}
			return SchemaValidationError{Path: path, Message: "expected object"}
		}
		for _, key := range node.required {
			if _, exists := obj[key]; !exists {
				return SchemaValidationError{Path: path, Message: fmt.Sprintf("missing required field %q", key)}
			}
		}
		for key, childValue := range obj {
			childSchema, ok := node.properties[key]
			if !ok {
				if !node.allowAdditional {
					return SchemaValidationError{Path: path, Message: fmt.Sprintf("additional property %q is not allowed", key)}
				}
				continue
			}
			if err := validateJSONSchemaNode(childSchema, childValue, path+"."+key); err != nil {
				return err
			}
		}
		return nil
	case "array":
		items, ok := value.([]any)
		if !ok {
			return SchemaValidationError{Path: path, Message: "expected array"}
		}
		for idx, item := range items {
			if err := validateJSONSchemaNode(node.items, item, fmt.Sprintf("%s[%d]", path, idx)); err != nil {
				return err
			}
		}
		return nil
	case "string":
		text, ok := value.(string)
		if !ok {
			return SchemaValidationError{Path: path, Message: "expected string"}
		}
		if node.minLength != nil && len(text) < *node.minLength {
			return SchemaValidationError{Path: path, Message: fmt.Sprintf("string length must be >= %d", *node.minLength)}
		}
		if node.maxLength != nil && len(text) > *node.maxLength {
			return SchemaValidationError{Path: path, Message: fmt.Sprintf("string length must be <= %d", *node.maxLength)}
		}
		return nil
	case "number":
		if _, ok := value.(float64); !ok {
			return SchemaValidationError{Path: path, Message: "expected number"}
		}
		return nil
	case "integer":
		number, ok := value.(float64)
		if !ok {
			return SchemaValidationError{Path: path, Message: "expected integer"}
		}
		if number != float64(int(number)) {
			return SchemaValidationError{Path: path, Message: "expected integer"}
		}
		return nil
	case "boolean":
		if _, ok := value.(bool); !ok {
			return SchemaValidationError{Path: path, Message: "expected boolean"}
		}
		return nil
	case "null":
		if value != nil {
			return SchemaValidationError{Path: path, Message: "expected null"}
		}
		return nil
	default:
		return SchemaValidationError{Path: path, Message: fmt.Sprintf("unsupported type %q", node.typ)}
	}
}
