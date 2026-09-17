package facturae_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	facturae "github.com/invopop/gobl.es.facturae"
	"github.com/invopop/gobl.es.facturae/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDocument(t *testing.T) {
	t.Run("should return an error if the envelope is empty", func(t *testing.T) {
		envelopeReader := strings.NewReader("")
		_, err := facturae.LoadGOBL(envelopeReader)
		assert.Error(t, err)
	})

	t.Run("should return an error if no document on the envelope", func(t *testing.T) {
		envelopeReader := strings.NewReader("{}")
		_, err := facturae.LoadGOBL(envelopeReader)
		assert.Error(t, err)
	})

	t.Run("should return a document", func(t *testing.T) {
		envelopeReader, err := os.Open(filepath.Join(test.GetDataPath(), "invoice-vat.json"))
		require.NoError(t, err)
		doc, err := facturae.LoadGOBL(envelopeReader)
		require.NoError(t, err)
		require.NotNil(t, doc.Invoices)
		require.NotNil(t, doc.Invoices.List)
		assert.NotEmpty(t, doc.Invoices.List)
	})
}

func TestTransformDocument(t *testing.T) {
	t.Run("basic invoice", func(t *testing.T) {
		doc, err := test.LoadGOBL("invoice-vat.json")
		require.NoError(t, err)
		data, err := doc.String()
		require.NoError(t, err)

		const xmlDeclaration = "<?xml version=\"1.0\" encoding=\"UTF-8\"?>"
		assert.Contains(t, data, xmlDeclaration, "contain XML declaration")

		const namespaceElement = "xmlns:fe=\"http://www.facturae.gob.es/formato/Versiones/Facturaev3_2_2.xml\""
		assert.Contains(t, data, namespaceElement, "contain root node namespace")

		assert.Contains(t, data, "<FileHeader>", "missing header info")

		assert.Contains(t, data, "<Parties>")
		assert.Contains(t, data, "<SellerParty>")
		assert.Contains(t, data, "<BuyerParty>")

		assert.Contains(t, data, "<InvoiceHeader>")
		assert.Contains(t, data, "<InvoiceIssueData>")

		assert.Contains(t, data, "<TaxesOutputs>")
		assert.NotContains(t, data, "<TaxesWithheld>")

		assert.Contains(t, data, "<InvoiceTotals>")

		assert.Contains(t, data, "<Items>")
		assert.Contains(t, data, "<InvoiceLine>")

		assert.Contains(t, data, "<AdditionalData>", "warning about rounding")
	})

	t.Run("invoice with withheld taxes", func(t *testing.T) {
		doc, err := test.LoadGOBL("invoice-vat-irpf.json")
		require.NoError(t, err)
		data, err := doc.String()
		require.NoError(t, err)

		assert.Nil(t, err)
		assert.Contains(t, data, "<TaxesOutputs>")
		assert.Contains(t, data, "<TaxesWithheld>")
	})

	t.Run("invoices with certificate", func(t *testing.T) {
		certificate, err := test.LoadCertificate()
		require.NoError(t, err)

		doc, err := test.LoadGOBL("invoice-vat.json", facturae.WithCertificate(certificate))
		require.NoError(t, err)

		data, err := doc.String()
		require.NoError(t, err)
		assert.Contains(t, data, "<ds:Signature")
	})

	t.Run("invoice with timestamp (TODO)", func(t *testing.T) {
		certificate, err := test.LoadCertificate()
		require.NoError(t, err)

		doc, err := test.LoadGOBL("invoice-vat.json", facturae.WithCertificate(certificate), facturae.WithTimestamp(true))
		require.Nil(t, err)

		data, err := doc.String()
		require.NoError(t, err)

		assert.Contains(t, data, "<xades:EncapsulatedTimeStamp>")
	})
}
