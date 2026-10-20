package facturae

import (
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/num"
)

// Amount wraps the Facturae AmountType. Monetary values rendered through
// amount are rounded to two decimals per Orden HAP/1650/2015 Annex II
// rule 6; unit prices and tax rates are the regulation's explicit exceptions
// and do not flow through this type.
type Amount struct {
	TotalAmount       string `xml:"TotalAmount"`
	EquivalentInEuros string `xml:"EquivalentInEuros,omitempty"`
}

// amount renders a GOBL amount as a Facturae monetary string, rounded to
// two decimals. Do not use for unit prices or rate/percentage fields.
func amount(a num.Amount) string {
	return a.Rescale(2).String()
}

// optionalAmount renders an amount that GOBL leaves unset, which the Facturae
// schema still expects as a figure. A missing amount is zero.
func optionalAmount(a *num.Amount) string {
	if a == nil {
		return amount(num.AmountZero)
	}
	return amount(*a)
}

// makeAmount always provides an AmountType. Invoices in other currencies
// provide the rate to euros, so that the equivalent required by Art. 10.1
// RD 1496/2003 is included.
func makeAmount(a num.Amount, rate *currency.ExchangeRate) Amount {
	out := Amount{
		TotalAmount: amount(a),
	}
	if rate != nil {
		// conversions keep the precision of the amount, so ensure cents
		if a.Exp() < 2 {
			a = a.Rescale(2)
		}
		out.EquivalentInEuros = amount(rate.Convert(a))
	}
	return out
}
