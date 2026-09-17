package facturae_test

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	facturae "github.com/invopop/gobl.es.facturae"
	"github.com/invopop/gobl.es.facturae/test"
	"github.com/invopop/xmldsig"
	"github.com/stretchr/testify/require"
)

var updateOut = flag.Bool("update", false, "Update the XML files in the test/data/out directory")

// signingTime is a fixed timestamp used for deterministic XAdES signatures in
// the examples so that the generated XML files can be compared across runs.
var signingTime = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

func TestXMLGeneration(t *testing.T) {
	examples, err := lookupExamples()
	require.NoError(t, err)

	cert, err := test.LoadCertificate()
	require.NoError(t, err)

	opts := []facturae.Option{
		facturae.WithThirdParty(test.ThirdParty()),
		facturae.WithCertificate(cert),
		facturae.WithSigning(
			xmldsig.WithCurrentTime(func() time.Time { return signingTime }),
		),
	}

	for _, example := range examples {
		name := fmt.Sprintf("should convert %s example file successfully", example)

		t.Run(name, func(t *testing.T) {
			data, err := convertExample(example, opts...)
			require.NoError(t, err)

			outPath := filepath.Join(test.GetDataPath(), "out", strings.TrimSuffix(example, ".json")+".xml")

			if *updateOut {
				test.ValidateAgainstSchema(t, data)

				err = os.WriteFile(outPath, data, 0644)
				require.NoError(t, err)

				return
			}

			expected, err := os.ReadFile(outPath)

			require.False(t, os.IsNotExist(err), "output file %s missing, run tests with `--update` flag to create", filepath.Base(outPath))
			require.NoError(t, err)
			require.Equal(t, string(expected), string(data), "output file %s does not match, run tests with `--update` flag to update", filepath.Base(outPath))
		})
	}
}

func lookupExamples() ([]string, error) {
	examples, err := filepath.Glob(filepath.Join(test.GetDataPath(), "*.json"))
	if err != nil {
		return nil, err
	}

	for i, example := range examples {
		examples[i] = filepath.Base(example)
	}

	return examples, nil
}

func convertExample(example string, opts ...facturae.Option) ([]byte, error) {
	doc, err := test.NewDocumentFrom(example, opts...)
	if err != nil {
		return nil, err
	}

	return doc.BytesIndent()
}
