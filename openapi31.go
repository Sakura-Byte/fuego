package fuego

import (
	"github.com/getkin/kin-openapi/openapi3"
)

// normalizeNullableForOpenAPI31 rewrites OpenAPI 3.0 `nullable: true` into the
// OpenAPI 3.1 form, where null is one of the schema's types.
//
// The schema generator (kin-openapi's openapi3gen) still marks pointer fields
// with the 3.0 `nullable` keyword, which OpenAPI 3.1 removed. Tools that read
// the document as 3.1 ignore that keyword and treat those fields as never null.
//
// A typed schema gets "null" added to its `type` list (and to its `enum`, if it
// has one). A schema with no type of its own, such as a composition, is wrapped
// as `anyOf: [<schema>, {type: null}]`.
func normalizeNullableForOpenAPI31(doc *openapi3.T) {
	if doc == nil {
		return
	}
	n := nullableNormalizer{seen: map[*openapi3.Schema]bool{}}

	if doc.Components != nil {
		for _, schemaRef := range doc.Components.Schemas {
			n.schemaRef(schemaRef)
		}
		for _, paramRef := range doc.Components.Parameters {
			n.parameterRef(paramRef)
		}
		for _, headerRef := range doc.Components.Headers {
			n.headerRef(headerRef)
		}
		for _, bodyRef := range doc.Components.RequestBodies {
			n.requestBodyRef(bodyRef)
		}
		for _, responseRef := range doc.Components.Responses {
			n.responseRef(responseRef)
		}
	}

	if doc.Paths != nil {
		for _, pathItem := range doc.Paths.Map() {
			n.pathItem(pathItem)
		}
	}
	for _, pathItem := range doc.Webhooks {
		n.pathItem(pathItem)
	}
}

type nullableNormalizer struct {
	seen map[*openapi3.Schema]bool
}

func (n nullableNormalizer) pathItem(pathItem *openapi3.PathItem) {
	if pathItem == nil {
		return
	}
	for _, paramRef := range pathItem.Parameters {
		n.parameterRef(paramRef)
	}
	for _, operation := range pathItem.Operations() {
		for _, paramRef := range operation.Parameters {
			n.parameterRef(paramRef)
		}
		n.requestBodyRef(operation.RequestBody)
		if operation.Responses != nil {
			for _, responseRef := range operation.Responses.Map() {
				n.responseRef(responseRef)
			}
		}
	}
}

func (n nullableNormalizer) parameterRef(ref *openapi3.ParameterRef) {
	if ref == nil || ref.Value == nil {
		return
	}
	n.schemaRef(ref.Value.Schema)
	n.content(ref.Value.Content)
}

func (n nullableNormalizer) headerRef(ref *openapi3.HeaderRef) {
	if ref == nil || ref.Value == nil {
		return
	}
	n.schemaRef(ref.Value.Schema)
	n.content(ref.Value.Content)
}

func (n nullableNormalizer) requestBodyRef(ref *openapi3.RequestBodyRef) {
	if ref == nil || ref.Value == nil {
		return
	}
	n.content(ref.Value.Content)
}

func (n nullableNormalizer) responseRef(ref *openapi3.ResponseRef) {
	if ref == nil || ref.Value == nil {
		return
	}
	n.content(ref.Value.Content)
	for _, headerRef := range ref.Value.Headers {
		n.headerRef(headerRef)
	}
}

func (n nullableNormalizer) content(content openapi3.Content) {
	for _, mediaType := range content {
		if mediaType != nil {
			n.schemaRef(mediaType.Schema)
		}
	}
}

func (n nullableNormalizer) schemaRefs(refs openapi3.SchemaRefs) {
	for _, ref := range refs {
		n.schemaRef(ref)
	}
}

func (n nullableNormalizer) schemaRef(ref *openapi3.SchemaRef) {
	if ref == nil || ref.Value == nil {
		return
	}
	schema := ref.Value
	if n.seen[schema] {
		return
	}
	n.seen[schema] = true

	for _, property := range schema.Properties {
		n.schemaRef(property)
	}
	n.schemaRef(schema.Items)
	n.schemaRef(schema.AdditionalProperties.Schema)
	n.schemaRef(schema.Not)
	n.schemaRefs(schema.AllOf)
	n.schemaRefs(schema.AnyOf)
	n.schemaRefs(schema.OneOf)

	if !schema.Nullable {
		return
	}
	schema.Nullable = false

	if schema.Type == nil || len(schema.Type.Slice()) == 0 {
		// No type to extend: allow null through a composition instead.
		inner := *schema
		*schema = openapi3.Schema{
			Description: inner.Description,
			AnyOf: openapi3.SchemaRefs{
				{Value: &inner},
				{Value: &openapi3.Schema{Type: &openapi3.Types{openapi3.TypeNull}}},
			},
		}
		inner.Description = ""
		n.seen[&inner] = true
		return
	}

	if !schema.Type.Includes(openapi3.TypeNull) {
		types := openapi3.Types(append(schema.Type.Slice(), openapi3.TypeNull))
		schema.Type = &types
	}
	if len(schema.Enum) > 0 && !enumIncludesNull(schema.Enum) {
		schema.Enum = append(schema.Enum, nil)
	}
}

func enumIncludesNull(values []any) bool {
	for _, value := range values {
		if value == nil {
			return true
		}
	}
	return false
}
