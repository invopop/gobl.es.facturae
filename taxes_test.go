package facturae_test

import (
	"testing"

	"github.com/invopop/gobl.es.facturae/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListSummary(t *testing.T) {
	t.Run("should contain the taxes summary", func(t *testing.T) {
		doc, err := test.LoadGOBL("invoice-vat-irpf-inc.json")
		require.NoError(t, err)

		xmlInvoice := doc.Invoices.List[0]

		assert.Nil(t, err)

		to0 := xmlInvoice.TaxesOutputs.List[0]
		assert.Equal(t, "01", to0.Code)
		assert.Equal(t, "3307.44", to0.Base.TotalAmount)
		assert.Equal(t, "21.0", to0.Rate)
		assert.Equal(t, "694.56", to0.Amount.TotalAmount)

		to1 := xmlInvoice.TaxesOutputs.List[1]
		assert.Equal(t, "01", to1.Code)
		assert.Equal(t, "31.82", to1.Base.TotalAmount)
		assert.Equal(t, "10.0", to1.Rate)
		assert.Equal(t, "3.18", to1.Amount.TotalAmount)

		to3 := xmlInvoice.TaxesOutputs.List[2]
		assert.Equal(t, "01", to3.Code)
		assert.Equal(t, "50.00", to3.Base.TotalAmount)
		assert.Equal(t, "0", to3.Rate)
		assert.Equal(t, "0.00", to3.Amount.TotalAmount)

		tw0 := xmlInvoice.TaxesWithheld.List[0]
		assert.Equal(t, "04", tw0.Code)
		assert.Equal(t, "3307.44", tw0.Base.TotalAmount)
		assert.Equal(t, "15.0", tw0.Rate)
		assert.Equal(t, "496.12", tw0.Amount.TotalAmount)
	})

}
