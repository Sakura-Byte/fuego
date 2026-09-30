package fuego_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Sakura-Byte/fuego"
)

type paramColor string

func (paramColor) EnumValues() []string { return []string{"red", "green"} }

type paramTier int

func (paramTier) EnumValues() []int { return []int{1, 2, 3} }

type enumParams struct {
	Color paramColor `query:"color" description:"Color filter"`
	Tier  paramTier  `query:"tier"`
	Name  string     `query:"name"`
}

func TestQueryParamsOfEnumTypesAreDocumentedAsEnums(t *testing.T) {
	s := fuego.NewServer(fuego.WithEngineOptions(fuego.WithOpenAPIConfig(fuego.OpenAPIConfig{
		DisableLocalSave: true,
		DisableSwaggerUI: true,
		DisableMessages:  true,
	})))
	fuego.Get(s, "/things", func(fuego.ContextWithParams[enumParams]) (string, error) { return "ok", nil })

	operation := s.OutputOpenAPISpec().Paths.Find("/things").Get
	enumOf := func(name string) []any {
		parameter := operation.Parameters.GetByInAndName("query", name)
		require.NotNil(t, parameter, name)
		return parameter.Schema.Value.Enum
	}

	require.Equal(t, []any{"red", "green"}, enumOf("color"))
	require.Equal(t, []any{1, 2, 3}, enumOf("tier"))
	require.Empty(t, enumOf("name"), "plain string parameters must stay unrestricted")
	require.Equal(t, "Color filter", operation.Parameters.GetByInAndName("query", "color").Description)
}

func TestParamEnumOptionRestrictsExplicitlyDeclaredParams(t *testing.T) {
	s := fuego.NewServer(fuego.WithEngineOptions(fuego.WithOpenAPIConfig(fuego.OpenAPIConfig{
		DisableLocalSave: true,
		DisableSwaggerUI: true,
		DisableMessages:  true,
	})))
	fuego.Get(s, "/things", func(fuego.ContextNoBody) (string, error) { return "ok", nil },
		fuego.OptionQuery("state", "State filter", fuego.ParamEnum("on", "off")),
	)

	parameter := s.OutputOpenAPISpec().Paths.Find("/things").Get.Parameters.GetByInAndName("query", "state")
	require.NotNil(t, parameter)
	require.Equal(t, []any{"on", "off"}, parameter.Schema.Value.Enum)
}
