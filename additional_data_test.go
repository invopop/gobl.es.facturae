package facturae_test

import (
	"testing"

	"github.com/invopop/gobl.es.facturae/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdditionalData(t *testing.T) {
	t.Run("should contain a note about line amouts rounding problems", func(t *testing.T) {
		doc, err := test.LoadGOBL("invoice-vat.json")
		require.NoError(t, err)

		xmlInvoice := doc.Invoices.List[0]

		assert.Nil(t, err)
		assert.Contains(t, xmlInvoice.AdditionalData.InvoiceAdditionalInformation, "Thank you for your custom")
	})
}
