package facturae

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/tax"
)

// Items contains info about all the lines in an invoice
type Items struct {
	InvoiceLine []*InvoiceLine `xml:"InvoiceLine"`
}

// InvoiceLine contains info about a specific line
type InvoiceLine struct {
	ItemDescription               string               `xml:"ItemDescription"`
	Quantity                      string               `xml:"Quantity"`
	UnitOfMeasure                 string               `xml:"UnitOfMeasure,omitempty"`
	UnitPriceWithoutTax           string               `xml:"UnitPriceWithoutTax"`
	TotalCost                     string               `xml:"TotalCost"` // Unit Price * quantity
	DiscountsAndRebates           *Discounts           `xml:"DiscountsAndRebates,omitempty"`
	Charges                       *Charges             `xml:"Charges,omitempty"`
	GrossAmount                   string               `xml:"GrossAmount"` // total cost - discount + charges
	TaxesWithheld                 *TaxSummary          `xml:"TaxesWithheld,omitempty"`
	TaxesOutputs                  *TaxSummary          `xml:"TaxesOutputs,omitempty"`
	AdditionalLineItemInformation string               `xml:"AdditionalLineItemInformation,omitempty"`
	SpecialTaxableEvent           *SpecialTaxableEvent `xml:"SpecialTaxableEvent,omitempty"`
}

// SpecialTaxableEvent stores details as to way the invoice line does
// not need taxes to be applied.
type SpecialTaxableEvent struct {
	SpecialTaxableEventCode   SpecialTaxableEventCode `xml:"SpecialTaxableEventCode"`
	SpecialTaxableEventReason string                  `xml:"SpecialTaxableEventReason"`
}

// SpecialTaxableEventCode used for special invoice lines that do not need taxes
// to be applied.
type SpecialTaxableEventCode string

// Special taxable event codes
const (
	TaxableAndExemptCode SpecialTaxableEventCode = "01"
	NonTaxableCode       SpecialTaxableEventCode = "02"
)

func newItems(lines []*bill.Line, taxes *tax.Total) *Items {
	xmlLines := make([]*InvoiceLine, 0, len(lines))
	for _, line := range lines {
		if line.Total == nil {
			// skip empty lines
			continue
		}
		xmlLines = append(xmlLines, newInvoiceLine(line, taxes))
	}
	return &Items{
		InvoiceLine: xmlLines,
	}
}

// newInvoiceLine unfortunately has to do more work than expected. The FacturaE format
// requires each line to show it's tax information. This is unfortunate as it
// can very easily lead to rounding errors and GOBL calculates taxes based on
// totals instead of per line.
// To try and get around this issue, we add an extra 2 points to exponents in
// Amounts so that we get a bit more precision.
func newInvoiceLine(line *bill.Line, taxes *tax.Total) *InvoiceLine {
	xmlLine := &InvoiceLine{
		ItemDescription:     line.Item.Name,
		Quantity:            line.Quantity.String(),
		UnitPriceWithoutTax: line.Item.Price.MinimalString(),
		TotalCost:           amount(*line.Sum),
		DiscountsAndRebates: newLineDiscounts(line.Discounts),
		Charges:             newLineCharges(line.Charges),
		GrossAmount:         amount(*line.Total),
	}
	xmlLine.addTaxes(*line.Total, taxes, line.Taxes)

	return xmlLine
}

// addTaxes reverse calculates what taxes should be assigned for each line.
func (l *InvoiceLine) addTaxes(total num.Amount, taxes *tax.Total, rates tax.Set) {
	// make sure total is not going to give is rounding errors
	if total.Exp() < 4 {
		total = total.Rescale(total.Exp() + 2)
	}
	regular := make([]*Tax, 0)
	retained := make([]*Tax, 0)
	for _, rate := range rates {
		ct := taxes.Category(rate.Category)
		if ct == nil {
			continue // skip, as we don't know what to do here!
		}

		tax := new(Tax)
		tax.Code = categoryTaxCodeMap[ct.Code]
		tax.Base = makeAmount(total)
		percent := rate.Percent
		if percent == nil {
			percent = num.NewPercentage(0, 0)
		}
		tax.Rate = percent.StringWithoutSymbol()
		tax.Amount = makeAmount(percent.Of(total))
		if rate.Surcharge != nil {
			p := *rate.Surcharge
			p = p.Rescale(4)
			tax.Surcharge = p.StringWithoutSymbol()
			v := makeAmount(rate.Surcharge.Of(total))
			tax.SurchargeAmount = &v
		}
		if ct.Retained {
			retained = append(retained, tax)
		} else {
			regular = append(regular, tax)
		}
	}

	// Update the line
	if len(regular) > 0 {
		l.TaxesOutputs = &TaxSummary{
			List: regular,
		}
	}
	if len(retained) > 0 {
		l.TaxesWithheld = &TaxSummary{
			List: retained,
		}
	}
}
