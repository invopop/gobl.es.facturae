package facturae

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/pay"
	"github.com/invopop/gobl/tax"

	addon "github.com/invopop/gobl.es.facturae/addon"
)

// goblPayment converts the installments and advances of the invoice into
// GOBL payment details. The payment means and accounts are taken from the
// first installment.
func goblPayment(pd *PaymentDetails, totals *InvoiceTotals) (*bill.PaymentDetails, error) {
	out := new(bill.PaymentDetails)
	if pd != nil && len(pd.Installments) > 0 {
		terms := new(pay.Terms)
		for _, in := range pd.Installments {
			dd := new(pay.DueDate)
			var err error
			if dd.Date, err = parseDate(in.InstallmentDueDate); err != nil {
				return nil, err
			}
			if dd.Amount, err = parseOptionalAmount(in.InstallmentAmount); err != nil {
				return nil, err
			}
			terms.DueDates = append(terms.DueDates, dd)
		}
		first := pd.Installments[0]
		terms.Notes = first.CollectionAdditionalInformation
		out.Terms = terms
		out.Instructions = goblInstructions(first)
	}
	advances, err := goblAdvances(totals)
	if err != nil {
		return nil, err
	}
	out.Advances = advances
	if out.Terms == nil && out.Instructions == nil && len(out.Advances) == 0 {
		return nil, nil
	}
	return out, nil
}

func goblInstructions(in *Installment) *pay.Instructions {
	key, ok := paymentMeansKey(in.PaymentMeans)
	if !ok {
		return nil
	}
	out := &pay.Instructions{Key: key}
	if a := in.AccountToBeCredited; a != nil {
		out.CreditTransfer = []*pay.CreditTransfer{{
			IBAN:   cbc.Code(a.IBAN),
			BIC:    cbc.Code(a.BIC),
			Number: cbc.Code(a.AccountNumber),
		}}
	}
	if a := in.AccountToBeDebited; a != nil {
		out.DirectDebit = &pay.DirectDebit{
			Ref:     cbc.Code(in.DebitReconciliationReference),
			Account: a.AccountNumber,
		}
	}
	return out
}

// paymentMeansKey provides the GOBL payment means for a FacturaE code.
func paymentMeansKey(code string) (cbc.Key, bool) {
	for k, c := range facturaePaymentMethodCodes {
		if c == code {
			return k, true
		}
	}
	return "", false
}

// goblAdvances converts payments on account, then subsidies, into advances,
// marking subsidies with the subsidy extension.
func goblAdvances(totals *InvoiceTotals) ([]*pay.Record, error) {
	if totals == nil {
		return nil, nil
	}
	var list []*pay.Record
	if totals.PaymentsOnAccount != nil {
		for _, p := range totals.PaymentsOnAccount.PaymentOnAccount {
			r := new(pay.Record)
			if !p.PaymentOnAccountDate.IsZero() {
				d := p.PaymentOnAccountDate
				r.Date = &d
			}
			var err error
			if r.Amount, err = parseAmount(p.PaymentOnAccountAmount); err != nil {
				return nil, err
			}
			list = append(list, r)
		}
	}
	if totals.Subsidies != nil {
		for _, s := range totals.Subsidies.Subsidy {
			r := &pay.Record{
				Description: s.SubsidyDescription,
				Ext:         tax.ExtensionsOf(cbc.CodeMap{addon.ExtKeySubsidy: "S"}),
			}
			var err error
			if r.Percent, err = parsePercent(s.SubsidyRate); err != nil {
				return nil, err
			}
			if r.Amount, err = parseAmount(s.SubsidyAmount); err != nil {
				return nil, err
			}
			list = append(list, r)
		}
	}
	return list, nil
}
