package facturae_test

import (
	"testing"

	"github.com/invopop/gobl.es.facturae/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFacturaeHeader(t *testing.T) {
	t.Run("should contain the facturae header info", func(t *testing.T) {
		doc, err := test.LoadGOBL("invoice-vat.json")
		require.NoError(t, err)

		assert.Nil(t, err)
		assert.Equal(t, "3.2.2", doc.FileHeader.SchemaVersion)
		assert.Equal(t, "I", doc.FileHeader.Modality)
		assert.Equal(t, "EM", doc.FileHeader.InvoiceIssuerType)
		assert.Equal(t, "B23103039-TEST01001F", doc.FileHeader.Batch.BatchIdentifier)
		assert.Equal(t, 1, doc.FileHeader.Batch.InvoicesCount)
		assert.Equal(t, "5084.42", doc.FileHeader.Batch.TotalInvoicesAmount.TotalAmount)
		assert.Equal(t, "5084.42", doc.FileHeader.Batch.TotalOutstandingAmount.TotalAmount)
		assert.Equal(t, "5084.42", doc.FileHeader.Batch.TotalExecutableAmount.TotalAmount)
		assert.Equal(t, "EUR", doc.FileHeader.Batch.InvoiceCurrencyCode)
	})
}
