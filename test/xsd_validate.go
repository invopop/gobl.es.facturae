package test

import (
	"path/filepath"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// ValidateAgainstSchema validates the given data against the FacturaE schema.
func ValidateAgainstSchema(t *testing.T, data []byte) {
	t.Helper()
	ctx := t.Context()

	// Load the XSD schema
	schemaPath := filepath.Join(GetTestPath(), "schema", "main.xsd")
	compileErrs := helium.NewErrorCollector(ctx, helium.ErrorLevelNone)
	schema, err := xsd.NewCompiler().
		FS(helium.PermissiveFS()).
		ErrorHandler(compileErrs).
		CompileFile(ctx, schemaPath)
	require.NoError(t, err, "%v", compileErrs.Errors())

	// Parse the XML document
	doc, err := helium.NewParser().Parse(ctx, data)
	require.NoError(t, err)

	// Validate the document against the schema
	collector := helium.NewErrorCollector(ctx, helium.ErrorLevelNone)
	err = xsd.NewValidator(schema).ErrorHandler(collector).Validate(ctx, doc)
	if err != nil {
		errs := collector.Errors()
		if len(errs) == 0 {
			require.NoError(t, err)
		}
		t.Errorf("Schema validation failed with %d errors:", len(errs))
		for _, e := range errs {
			t.Errorf("  - %s", e.Error())
		}
		t.FailNow()
	}
}
