package facturae_test

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	facturae "github.com/invopop/gobl.es.facturae"
	"github.com/invopop/gobl.es.facturae/test"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/uuid"
	"github.com/invopop/xmldsig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var updateOut = flag.Bool("update", false, "Update the expected output files in test/data")

// staticUUID replaces the random UUIDs of imported documents, so that their
// output can be compared.
const staticUUID uuid.UUID = "0195ce71-dc9c-72c8-bf2c-9890a4a9f0a2"

// signingTime is a fixed timestamp used for deterministic XAdES signatures in
// the examples so that the generated XML files can be compared across runs.
var signingTime = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

func TestConvertExamples(t *testing.T) {
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

			outPath := filepath.Join(test.GetDataPath(), "convert", "out", strings.TrimSuffix(example, ".json")+".xml")

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
	examples, err := filepath.Glob(filepath.Join(test.GetDataPath(), "convert", "*.json"))
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

func TestParseExamples(t *testing.T) {
	examples, err := filepath.Glob(filepath.Join(test.GetDataPath(), "parse", "*.xml"))
	require.NoError(t, err)
	require.NotEmpty(t, examples)

	for _, example := range examples {
		name := filepath.Base(example)
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(example)
			require.NoError(t, err)
			doc, err := facturae.Decode(data)
			require.NoError(t, err)
			env, err := facturae.Import(doc)
			require.NoError(t, err)

			inv, ok := env.Extract().(*bill.Invoice)
			require.True(t, ok)
			declared := doc.Invoices.List[0].InvoiceTotals.TotalExecutableAmount
			due := inv.Totals.Payable
			if inv.Totals.Due != nil {
				due = *inv.Totals.Due
			}
			assert.Equal(t, strings.TrimPrefix(declared, "-"), due.String(), "amount due matches the document")

			env.Head.UUID = staticUUID
			inv.UUID = staticUUID
			require.NoError(t, env.Calculate())
			out, err := json.MarshalIndent(env, "", "\t")
			require.NoError(t, err)

			outPath := filepath.Join(test.GetDataPath(), "parse", "out", strings.TrimSuffix(name, ".xml")+".json")
			if *updateOut {
				require.NoError(t, os.WriteFile(outPath, out, 0644))
				return
			}
			expected, err := os.ReadFile(outPath)
			require.NoError(t, err, "output file %s missing, run tests with `--update` flag to create", filepath.Base(outPath))
			assert.JSONEq(t, string(expected), string(out), "output file %s does not match, run tests with `--update` flag to update", filepath.Base(outPath))
		})
	}
}

// TestRoundTrip checks that every example exported to FacturaE imports back
// with the same type, lines and totals.
func TestRoundTrip(t *testing.T) {
	examples, err := lookupExamples()
	require.NoError(t, err)

	for _, example := range examples {
		t.Run(example, func(t *testing.T) {
			src, err := test.LoadTestEnvelope(example)
			require.NoError(t, err)
			srcInv := src.Extract().(*bill.Invoice)
			before, err := json.Marshal(src)
			require.NoError(t, err)

			doc, err := facturae.Export(src)
			require.NoError(t, err)
			after, err := json.Marshal(src)
			require.NoError(t, err)
			assert.JSONEq(t, string(before), string(after), "export leaves the envelope unchanged")
			data, err := facturae.Encode(doc)
			require.NoError(t, err)
			doc, err = facturae.Decode(data)
			require.NoError(t, err)
			env, err := facturae.Import(doc)
			require.NoError(t, err)

			inv := env.Extract().(*bill.Invoice)
			assert.Equal(t, srcInv.Type, inv.Type)
			assert.Equal(t, srcInv.Currency, inv.Currency)
			assert.Equal(t, srcInv.ExchangeRates, inv.ExchangeRates)
			assert.Len(t, inv.Lines, len(srcInv.Lines))
			assert.Equal(t, srcInv.Totals.Payable.String(), inv.Totals.Payable.String())
		})
	}
}
