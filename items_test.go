package facturae_test

import (
	"testing"

	"github.com/invopop/gobl.es.facturae/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvoiceLineIncludingTaxes(t *testing.T) {
	doc, err := test.LoadGOBL("invoice-vat-irpf-inc.json")
	require.NoError(t, err)

	xmlInvoice := doc.Invoices.List[0]

	assert.Nil(t, err)

	line1 := xmlInvoice.Items.InvoiceLine[0]
	assert.Equal(t, "20", line1.Quantity)
	assert.Equal(t, "Operations and development - day rate", line1.ItemDescription)
	assert.Equal(t, "165.2893", line1.UnitPriceWithoutTax)
	assert.Equal(t, "3305.79", line1.TotalCost)
	assert.Equal(t, "just because", line1.DiscountsAndRebates.Items[0].Reason)
	assert.Equal(t, "165.29", line1.DiscountsAndRebates.Items[0].Amount)
	assert.Equal(t, "3140.50", line1.GrossAmount)

	tax1 := line1.TaxesOutputs.List[0]
	tax2 := line1.TaxesWithheld.List[0]
	assert.Equal(t, "01", tax1.Code)
	assert.Equal(t, "3140.50", tax1.Base.TotalAmount)
	assert.Equal(t, "21.0", tax1.Rate)
	assert.Equal(t, "659.50", tax1.Amount.TotalAmount)
	assert.Equal(t, "04", tax2.Code)
	assert.Equal(t, "3140.50", tax2.Base.TotalAmount)
	assert.Equal(t, "15.0", tax2.Rate)
	assert.Equal(t, "471.07", tax2.Amount.TotalAmount)

	line2 := xmlInvoice.Items.InvoiceLine[1]
	tax1 = line2.TaxesOutputs.List[0]
	assert.Equal(t, "2", line2.Quantity)
	assert.Equal(t, "Additional Overtime", line2.ItemDescription)
	assert.Equal(t, "83.4711", line2.UnitPriceWithoutTax)
	assert.Equal(t, "166.94", line2.TotalCost)
	assert.Equal(t, "166.94", line2.GrossAmount)
	assert.Equal(t, "01", tax1.Code)
	assert.Equal(t, "166.94", tax1.Base.TotalAmount)
	assert.Equal(t, "21.0", tax1.Rate)
	assert.Equal(t, "35.06", tax1.Amount.TotalAmount)

	line3 := xmlInvoice.Items.InvoiceLine[2]
	tax1 = line3.TaxesOutputs.List[0]
	assert.Equal(t, "1", line3.Quantity)
	assert.Equal(t, "Extra food costs", line3.ItemDescription)
	assert.Equal(t, "31.8182", line3.UnitPriceWithoutTax)
	assert.Equal(t, "31.82", line3.TotalCost)
	assert.Equal(t, "31.82", line3.GrossAmount)
	assert.Equal(t, "01", tax1.Code)
	assert.Equal(t, "31.82", tax1.Base.TotalAmount)
	assert.Equal(t, "10.0", tax1.Rate)
	assert.Equal(t, "3.18", tax1.Amount.TotalAmount)
}

func TestInvoiceLineTaxNotIncluded(t *testing.T) {
	doc, err := test.LoadGOBL("invoice-vat-irpf.json")
	require.NoError(t, err)

	xmlInvoice := doc.Invoices.List[0]

	line1 := xmlInvoice.Items.InvoiceLine[0]
	tax1 := line1.TaxesOutputs.List[0]
	tax2 := line1.TaxesWithheld.List[0]
	assert.Equal(t, "20", line1.Quantity)
	assert.Equal(t, "Operations and development - day rate", line1.ItemDescription)
	assert.Equal(t, "200", line1.UnitPriceWithoutTax)
	assert.Equal(t, "4000.00", line1.TotalCost)
	assert.Equal(t, "just because", line1.DiscountsAndRebates.Items[0].Reason)
	assert.Equal(t, "200.00", line1.DiscountsAndRebates.Items[0].Amount)
	assert.Equal(t, "3800.00", line1.GrossAmount)
	assert.Equal(t, "01", tax1.Code)
	assert.Equal(t, "3800.00", tax1.Base.TotalAmount)
	assert.Equal(t, "21.0", tax1.Rate)
	assert.Equal(t, "798.00", tax1.Amount.TotalAmount)
	assert.Equal(t, "04", tax2.Code)
	assert.Equal(t, "3800.00", tax2.Base.TotalAmount)
	assert.Equal(t, "15.0", tax2.Rate)
	assert.Equal(t, "570.00", tax2.Amount.TotalAmount)
}
