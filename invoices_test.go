package facturae_test

import (
	"testing"

	"github.com/invopop/gobl.es.facturae/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvoiceInvoiceHeader(t *testing.T) {
	t.Run("should contain the invoice header data", func(t *testing.T) {
		doc, err := test.LoadGOBL("invoice-vat.json")
		require.NoError(t, err)

		xmlInvoice := doc.Invoices.List[0]

		assert.Nil(t, err)
		assert.Equal(t, "TEST01001F", xmlInvoice.InvoiceHeader.InvoiceNumber)
		assert.Equal(t, "FC", xmlInvoice.InvoiceHeader.InvoiceDocumentType)
		assert.Equal(t, "OO", xmlInvoice.InvoiceHeader.InvoiceClass)
		assert.Equal(t, "2021-12-08", xmlInvoice.InvoiceIssueData.IssueDate)
		assert.Equal(t, "2021-12-08", xmlInvoice.InvoiceIssueData.OperationDate)
		assert.Equal(t, "EUR", xmlInvoice.InvoiceIssueData.InvoiceCurrencyCode)
		assert.Equal(t, "EUR", xmlInvoice.InvoiceIssueData.TaxCurrencyCode)
		assert.Equal(t, "es", xmlInvoice.InvoiceIssueData.LanguageName)
	})

}
