package fuego

import (
	"encoding/json"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/require"
)

type nullableFieldsResponse struct {
	Name     string   `json:"name"`
	Nickname *string  `json:"nickname,omitempty"`
	Age      *int     `json:"age,omitempty"`
	Mode     *string  `json:"mode,omitempty" validate:"omitempty,oneof=fast slow"`
	Tags     []string `json:"tags"`
}

func TestNormalizeNullableForOpenAPI31(t *testing.T) {
	s := NewServer(WithEngineOptions(WithOpenAPIConfig(OpenAPIConfig{DisableLocalSave: true, DisableMessages: true})))
	Get(s, "/nullable", func(c ContextNoBody) (nullableFieldsResponse, error) {
		return nullableFieldsResponse{}, nil
	})

	doc := s.OutputOpenAPISpec()
	raw, err := json.Marshal(doc)
	require.NoError(t, err)
	require.NotContains(t, string(raw), `"nullable"`)

	schema := doc.Components.Schemas["nullableFieldsResponse"].Value
	require.NotNil(t, schema)
	require.True(t, schema.Properties["nickname"].Value.Type.Includes(openapi3.TypeNull))
	require.True(t, schema.Properties["nickname"].Value.Type.Includes(openapi3.TypeString))
	require.True(t, schema.Properties["age"].Value.Type.Includes(openapi3.TypeNull))
	require.False(t, schema.Properties["name"].Value.Type.Includes(openapi3.TypeNull))
}

func TestNormalizeNullableForOpenAPI31Composition(t *testing.T) {
	doc := &openapi3.T{
		OpenAPI: "3.1.0",
		Components: &openapi3.Components{Schemas: openapi3.Schemas{
			"WithEnum": {Value: &openapi3.Schema{
				Type:     &openapi3.Types{openapi3.TypeString},
				Enum:     []any{"a", "b"},
				Nullable: true,
			}},
			"Untyped": {Value: &openapi3.Schema{
				Description: "untyped",
				AllOf:       openapi3.SchemaRefs{{Ref: "#/components/schemas/WithEnum"}},
				Nullable:    true,
			}},
		}},
	}

	normalizeNullableForOpenAPI31(doc)

	withEnum := doc.Components.Schemas["WithEnum"].Value
	require.False(t, withEnum.Nullable)
	require.Equal(t, []any{"a", "b", nil}, withEnum.Enum)
	require.True(t, withEnum.Type.Includes(openapi3.TypeNull))

	untyped := doc.Components.Schemas["Untyped"].Value
	require.False(t, untyped.Nullable)
	require.Equal(t, "untyped", untyped.Description)
	require.Len(t, untyped.AnyOf, 2)
	require.False(t, untyped.AnyOf[0].Value.Nullable)
	require.Empty(t, untyped.AnyOf[0].Value.Description)
	require.True(t, untyped.AnyOf[1].Value.Type.Includes(openapi3.TypeNull))
}

type customErrorEnvelope struct {
	Error struct {
		Code string `json:"code"`
	} `json:"error"`
}

func TestWithDefaultErrorResponseType(t *testing.T) {
	s := NewServer(
		WithDefaultErrorResponseType(customErrorEnvelope{}),
		WithEngineOptions(WithOpenAPIConfig(OpenAPIConfig{DisableLocalSave: true, DisableMessages: true})),
	)
	Get(s, "/thing", func(c ContextNoBody) (string, error) { return "", nil })

	doc := s.OutputOpenAPISpec()
	require.NotContains(t, doc.Components.Schemas, "HTTPError")
	for _, status := range []string{"400", "500"} {
		ref := doc.Paths.Find("/thing").Get.Responses.Value(status).Value.Content["application/json"].Schema.Ref
		require.Equal(t, "#/components/schemas/customErrorEnvelope", ref)
	}
}

func TestResponseOptionsUseEngineContentTypes(t *testing.T) {
	s := NewServer(WithEngineOptions(
		WithResponseContentType("application/json"),
		WithOpenAPIConfig(OpenAPIConfig{DisableLocalSave: true, DisableMessages: true}),
	))
	Get(s, "/thing", func(c ContextNoBody) (string, error) { return "", nil },
		OptionAddResponse(409, "Conflict", Response{Type: HTTPError{}}),
		OptionDefaultResponse("Error", Response{Type: HTTPError{}}),
	)

	responses := s.OutputOpenAPISpec().Paths.Find("/thing").Get.Responses
	for _, status := range []string{"400", "409", "500", "default"} {
		content := responses.Value(status).Value.Content
		require.Contains(t, content, "application/json", status)
		require.NotContains(t, content, "application/xml", status)
	}
}
