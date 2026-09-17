package facturae_test

import (
	"testing"

	"github.com/invopop/gobl.es.facturae/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvoiceInvoiceTotals(t *testing.T) {
	t.Run("should contain the total amounts block", func(t *testing.T) {
		doc, err := test.LoadGOBL("invoice-vat.json")
		require.NoError(t, err)

		xmlInvoice := doc.Invoices.List[0]

		assert.Nil(t, err)
		assert.Equal(t, "4202.00", xmlInvoice.InvoiceTotals.TotalGrossAmountBeforeTaxes)
		assert.Equal(t, "", xmlInvoice.InvoiceTotals.TotalGeneralDiscounts)
		assert.Equal(t, "4202.00", xmlInvoice.InvoiceTotals.TotalGrossAmount)
		assert.Equal(t, "882.42", xmlInvoice.InvoiceTotals.TotalTaxOutputs)
		assert.Equal(t, "0.00", xmlInvoice.InvoiceTotals.TotalTaxesWithheld)
		assert.Equal(t, "5084.42", xmlInvoice.InvoiceTotals.InvoiceTotal)
		assert.Equal(t, "5084.42", xmlInvoice.InvoiceTotals.TotalOutstandingAmount)
		assert.Equal(t, "5084.42", xmlInvoice.InvoiceTotals.TotalExecutableAmount)
	})
}
