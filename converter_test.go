package facturae_test

import (
	"testing"

	"github.com/invopop/gobl"
	facturae "github.com/invopop/gobl.es.facturae"
	addon "github.com/invopop/gobl.es.facturae/addon"
	"github.com/invopop/gobl.es.facturae/test"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/convert"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConverterFormat(t *testing.T) {
	f := convert.FormatFor(facturae.KeyFacturaE)
	require.NotNil(t, f)
	assert.Equal(t, cbc.Key("facturae"), f.Syntax)
	assert.Equal(t, []l10n.Code{l10n.ES}, f.Countries)
	assert.Len(t, f.Import, 1)
	assert.Len(t, f.Export, 1)
}

func TestConverterExport(t *testing.T) {
	t.Run("attachments", func(t *testing.T) {
		env, err := test.LoadTestEnvelope("invoice-vat.json")
		require.NoError(t, err)
		pdf := &convert.Attachment{Name: "invoice.pdf", MIME: "application/pdf", Data: []byte("%PDF")}
		_, err = convert.Export(env, facturae.KeyFacturaE, convert.WithAttachments(pdf))
		assert.ErrorIs(t, err, convert.ErrNotSupported)
	})
	t.Run("invoice", func(t *testing.T) {
		env, err := test.LoadTestEnvelope("invoice-vat.json")
		require.NoError(t, err)
		out, err := convert.Export(env, facturae.KeyFacturaE)
		require.NoError(t, err)
		assert.Equal(t, facturae.KeyFacturaE, out.Format.Key)

		doc, err := facturae.Export(env)
		require.NoError(t, err)
		data, err := facturae.Encode(doc)
		require.NoError(t, err)
		assert.Equal(t, string(data), string(out.Data))
	})

	faultCode := func(t *testing.T, err error) rules.Code {
		t.Helper()
		assert.ErrorIs(t, err, convert.ErrNotSupported)
		var ge *gobl.Error
		require.ErrorAs(t, err, &ge)
		require.NotNil(t, ge.Faults())
		return ge.Faults().First().Code()
	}

	t.Run("missing addon", func(t *testing.T) {
		env, err := test.LoadTestEnvelope("invoice-vat.json")
		require.NoError(t, err)
		env.Extract().(*bill.Invoice).SetAddons()
		_, err = convert.Export(env, facturae.KeyFacturaE)
		assert.Equal(t, rules.Code("ES-FACTURAE-V3-BILL-INVOICE-01"), faultCode(t, err))
	})

	t.Run("missing customer", func(t *testing.T) {
		env, err := test.LoadTestEnvelope("invoice-vat.json")
		require.NoError(t, err)
		env.Extract().(*bill.Invoice).Customer = nil
		_, err = convert.Export(env, facturae.KeyFacturaE)
		assert.Equal(t, rules.Code("ES-FACTURAE-V3-BILL-INVOICE-02"), faultCode(t, err))
	})

	t.Run("detected and imported", func(t *testing.T) {
		env, err := test.LoadTestEnvelope("invoice-vat.json")
		require.NoError(t, err)
		out, err := convert.Export(env, facturae.KeyFacturaE)
		require.NoError(t, err)

		f, err := convert.Detect(out.Data)
		require.NoError(t, err)
		assert.Equal(t, facturae.KeyFacturaE, f.Key)

		imported, err := convert.Import(out.Data)
		require.NoError(t, err)
		inv := imported.Extract().(*bill.Invoice)
		assert.Equal(t, env.Extract().(*bill.Invoice).Totals.Payable.String(), inv.Totals.Payable.String())
		assert.True(t, addon.V3.In(inv.GetAddons()...))
	})

	t.Run("other documents not detected", func(t *testing.T) {
		for _, data := range []string{`<Invoice xmlns="urn:example"/>`, `{}`, ``} {
			_, err := convert.Detect([]byte(data))
			assert.ErrorIs(t, err, convert.ErrUnknownFormat)
		}
	})

	t.Run("invalid document", func(t *testing.T) {
		_, err := convert.Import([]byte(`<fe:Facturae xmlns:fe="` + facturae.NamespaceFacturaE + `"><`))
		assert.Error(t, err)
	})
}
