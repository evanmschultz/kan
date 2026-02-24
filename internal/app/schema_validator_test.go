package app

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestCompileJSONSchemaValidatePayload exercises core schema-type validation branches.
func TestCompileJSONSchemaValidatePayload(t *testing.T) {
	schema := `{
		"type":"object",
		"required":["name","count","active","tags","mode","ratio","nothing"],
		"additionalProperties":false,
		"properties":{
			"name":{"type":"string","minLength":2,"maxLength":5},
			"count":{"type":"integer"},
			"active":{"type":"boolean"},
			"tags":{"type":"array","items":{"type":"string"}},
			"mode":{"enum":["fast","safe"]},
			"ratio":{"type":"number"},
			"nothing":{"type":"null"}
		}
	}`
	validator, err := compileJSONSchema(schema)
	if err != nil {
		t.Fatalf("compileJSONSchema() error = %v", err)
	}

	tests := []struct {
		name    string
		payload string
		wantErr string
	}{
		{
			name:    "valid payload",
			payload: `{"name":"alex","count":2,"active":true,"tags":["x","y"],"mode":"fast","ratio":1.5,"nothing":null}`,
		},
		{
			name:    "missing required",
			payload: `{"count":2}`,
			wantErr: `missing required field "active"`,
		},
		{
			name:    "additional property blocked",
			payload: `{"name":"alex","count":2,"active":true,"tags":["x"],"mode":"fast","ratio":1.5,"nothing":null,"extra":1}`,
			wantErr: `additional property "extra" is not allowed`,
		},
		{
			name:    "string too short",
			payload: `{"name":"a","count":2,"active":true,"tags":["x"],"mode":"fast","ratio":1.5,"nothing":null}`,
			wantErr: "string length must be >= 2",
		},
		{
			name:    "string too long",
			payload: `{"name":"toolong","count":2,"active":true,"tags":["x"],"mode":"fast","ratio":1.5,"nothing":null}`,
			wantErr: "string length must be <= 5",
		},
		{
			name:    "integer type mismatch",
			payload: `{"name":"alex","count":2.2,"active":true,"tags":["x"],"mode":"fast","ratio":1.5,"nothing":null}`,
			wantErr: "expected integer",
		},
		{
			name:    "boolean type mismatch",
			payload: `{"name":"alex","count":2,"active":"true","tags":["x"],"mode":"fast","ratio":1.5,"nothing":null}`,
			wantErr: "expected boolean",
		},
		{
			name:    "array item type mismatch",
			payload: `{"name":"alex","count":2,"active":true,"tags":[1],"mode":"fast","ratio":1.5,"nothing":null}`,
			wantErr: "$.tags[0]: expected string",
		},
		{
			name:    "enum mismatch",
			payload: `{"name":"alex","count":2,"active":true,"tags":["x"],"mode":"slow","ratio":1.5,"nothing":null}`,
			wantErr: "value is not in enum set",
		},
		{
			name:    "invalid json payload",
			payload: `{"name":`,
			wantErr: "invalid JSON payload",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := validator.ValidatePayload(json.RawMessage(tc.payload))
			if tc.wantErr == "" && err != nil {
				t.Fatalf("ValidatePayload() unexpected error = %v", err)
			}
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("ValidatePayload() expected error containing %q", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("ValidatePayload() error = %v, want contains %q", err, tc.wantErr)
				}
			}
		})
	}
}

// TestCompileJSONSchemaSupportsTypelessObject verifies permissive behavior when schema omits explicit type.
func TestCompileJSONSchemaSupportsTypelessObject(t *testing.T) {
	validator, err := compileJSONSchema(`{"properties":{"name":{"type":"string"}}}`)
	if err != nil {
		t.Fatalf("compileJSONSchema() error = %v", err)
	}
	if err := validator.ValidatePayload(json.RawMessage(`5`)); err != nil {
		t.Fatalf("ValidatePayload() unexpected error for type-less root = %v", err)
	}
}

// TestCompileJSONSchemaRejectsInvalidDefinitions verifies deterministic schema compile failures.
func TestCompileJSONSchemaRejectsInvalidDefinitions(t *testing.T) {
	tests := []struct {
		name    string
		schema  string
		wantErr string
	}{
		{name: "root must be object", schema: `[]`, wantErr: "schema must be an object"},
		{name: "type must be string", schema: `{"type":1}`, wantErr: "$.type"},
		{name: "unsupported type", schema: `{"type":"mystery"}`, wantErr: "unsupported type"},
		{name: "required must be array", schema: `{"required":"name"}`, wantErr: "$.required"},
		{name: "required item must be string", schema: `{"required":[1]}`, wantErr: "$.required[0]"},
		{name: "required item cannot be empty", schema: `{"required":[""]}`, wantErr: "must not be empty"},
		{name: "properties must be object", schema: `{"properties":[]}`, wantErr: "$.properties"},
		{name: "additionalProperties must be bool", schema: `{"additionalProperties":"no"}`, wantErr: "$.additionalProperties"},
		{name: "enum must be array", schema: `{"enum":"x"}`, wantErr: "$.enum"},
		{name: "items must be object schema", schema: `{"items":[]}`, wantErr: "$.items"},
		{name: "minLength must be >= 0", schema: `{"minLength":-1}`, wantErr: "$.minLength"},
		{name: "maxLength must be integer", schema: `{"maxLength":1.5}`, wantErr: "$.maxLength"},
		{name: "maxLength must be number", schema: `{"maxLength":"x"}`, wantErr: "$.maxLength"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := compileJSONSchema(tc.schema)
			if err == nil {
				t.Fatalf("compileJSONSchema() expected error containing %q", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("compileJSONSchema() error = %v, want contains %q", err, tc.wantErr)
			}
		})
	}
}

// TestParseSchemaInt verifies numeric normalization behavior for length bounds.
func TestParseSchemaInt(t *testing.T) {
	value, err := parseSchemaInt(3)
	if err != nil {
		t.Fatalf("parseSchemaInt(int) error = %v", err)
	}
	if value != 3 {
		t.Fatalf("parseSchemaInt(int) = %d, want 3", value)
	}
	if _, err := parseSchemaInt(-1); err == nil {
		t.Fatal("parseSchemaInt(-1) expected error")
	}
}
