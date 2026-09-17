package facturae

import "github.com/invopop/gobl/num"

// Amount wraps the Facturae AmountType. Monetary values rendered through
// amount are rounded to two decimals per Orden HAP/1650/2015 Annex II
// rule 6; unit prices and tax rates are the regulation's explicit exceptions
// and do not flow through this type.
type Amount struct {
	TotalAmount       string
	EquivalentInEuros string `xml:",omitempty"` // not used yet!
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

// makeAmount always provides an AmountType.
func makeAmount(a num.Amount) Amount {
	return Amount{
		TotalAmount: amount(a),
	}
}
