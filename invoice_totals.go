package facturae

import (
	addon "github.com/invopop/gobl.es.facturae/addon"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/pay"
	"github.com/invopop/gobl/tax"
)

// InvoiceTotals contains the summary of the amounts in an invoice
type InvoiceTotals struct {
	TotalGrossAmount            string                `xml:"TotalGrossAmount"` // Without included taxes
	GeneralDiscounts            *Discounts            `xml:"GeneralDiscounts,omitempty"`
	GeneralSurcharges           *Charges              `xml:"GeneralSurcharges,omitempty"`
	TotalGeneralDiscounts       string                `xml:"TotalGeneralDiscounts,omitempty"`
	TotalGeneralSurcharges      string                `xml:"TotalGeneralSurcharges,omitempty"`
	TotalGrossAmountBeforeTaxes string                `xml:"TotalGrossAmountBeforeTaxes"`
	TotalTaxOutputs             string                `xml:"TotalTaxOutputs"`
	TotalTaxesWithheld          string                `xml:"TotalTaxesWithheld"`
	InvoiceTotal                string                `xml:"InvoiceTotal"`
	Subsidies                   *Subsidies            `xml:"Subsidies,omitempty"`
	PaymentsOnAccount           *PaymentsOnAccount    `xml:"PaymentsOnAccount,omitempty"`
	ReimbursableExpenses        *ReimbursableExpenses `xml:"ReimbursableExpenses,omitempty"`
	TotalOutstandingAmount      string                `xml:"TotalOutstandingAmount"` // InvoiceTotal - (Total subvenciones + TotalPaymentsOnAccount)
	TotalPaymentsOnAccount      string                `xml:"TotalPaymentsOnAccount,omitempty"`
	TotalExecutableAmount       string                `xml:"TotalExecutableAmount"`
	TotalReimbursableExpenses   string                `xml:"TotalReimbursableExpenses,omitempty"`
}

// Subsidies is currently a placeholder, as we don't use it yet.
type Subsidies struct {
	Subsidy []*Subsidy `xml:"Subsidy"`
}

// Subsidy is a single subsidy entry
type Subsidy struct {
	SubsidyDescription string `xml:"SubsidyDescription"`
	SubsidyRate        string `xml:"SubsidyRate,omitempty"`
	SubsidyAmount      string `xml:"SubsidyAmount"`
}

// PaymentsOnAccount stores payments made in advance.
type PaymentsOnAccount struct {
	PaymentOnAccount []*PaymentOnAccount `xml:"PaymentOnAccount"`
}

// PaymentOnAccount is a single advance payment.
type PaymentOnAccount struct {
	PaymentOnAccountDate   cal.Date `xml:"PaymentOnAccountDate"`
	PaymentOnAccountAmount string   `xml:"PaymentOnAccountAmount"`
}

// ReimbursableExpenses is a group of reimbursable expenses.
type ReimbursableExpenses struct {
	ReimbursableExpenses []*ReimbursableExpense `xml:"ReimbursableExpenses"`
}

// ReimbursableExpense stores info about a single reimbursable expense.
type ReimbursableExpense struct {
	ReimbursableExpensesSellerParty *TaxID    `xml:"ReimbursableExpensesSellerParty,omitempty"`
	ReimbursableExpensesBuyerParty  *TaxID    `xml:"ReimbursableExpensesBuyerParty,omitempty"` // never used!
	IssueDate                       *cal.Date `xml:"IssueDate,omitempty"`
	InvoiceNumber                   string    `xml:"InvoiceNumber,omitempty"`
	InvoiceSeriesCode               string    `xml:"InvoiceSeriesCode,omitempty"`
	ReimbursableExpensesAmount      string    `xml:"ReimbursableExpensesAmount"`
}

func newInvoiceTotals(invoice *bill.Invoice) *InvoiceTotals {
	totals := invoice.Totals

	sum := totals.Sum
	if totals.TaxIncluded != nil {
		sum = sum.Subtract(*totals.TaxIncluded)
	}

	due := totals.Payable
	outstanding := totals.Payable
	if totals.Due != nil {
		due = *totals.Due
		outstanding = *totals.Due
	}
	/*
		if totals.Outlays != nil {
			outstanding = outstanding.Subtract(*totals.Outlays)
		}
	*/

	xmlTotals := &InvoiceTotals{
		TotalGrossAmount:            amount(sum),
		GeneralDiscounts:            newDiscounts(invoice.Discounts),
		GeneralSurcharges:           newCharges(invoice.Charges),
		TotalGrossAmountBeforeTaxes: amount(totals.Total),
		InvoiceTotal:                amount(totals.TotalWithTax),
		TotalOutstandingAmount:      amount(outstanding),
		TotalExecutableAmount:       amount(due),
	}

	if totals.Discount != nil {
		xmlTotals.TotalGeneralDiscounts = amount(*totals.Discount)
	}
	if totals.Charge != nil {
		xmlTotals.TotalGeneralSurcharges = amount(*totals.Charge)
	}
	if totals.Advances != nil {
		xmlTotals.TotalPaymentsOnAccount = amount(*totals.Advances)
	}
	/*
		if totals.Outlays != nil {
			xmlTotals.TotalReimbursableExpenses = totals.Outlays.String()
		}
	*/

	if invoice.Payment != nil {
		xmlTotals.setAdvances(invoice.Payment.Advances)
	}
	// xmlTotals.setOutlays(invoice.Outlays)
	xmlTotals.setTaxTotals(totals.Taxes)

	return xmlTotals
}

func (it *InvoiceTotals) setTaxTotals(taxes *tax.Total) {
	regular := num.MakeAmount(0, 2)
	retained := num.MakeAmount(0, 2)
	for _, ct := range taxes.Categories {
		if ct.Retained {
			retained = retained.Add(ct.Amount)
		} else {
			regular = regular.Add(ct.Amount)
			if ct.Surcharge != nil {
				regular = regular.Add(*ct.Surcharge)
			}
		}
	}
	it.TotalTaxOutputs = amount(regular)
	it.TotalTaxesWithheld = amount(retained)
}

func (it *InvoiceTotals) setAdvances(advances []*pay.Record) {
	regular := make([]*PaymentOnAccount, 0)
	grants := make([]*Subsidy, 0)
	for _, a := range advances {
		if a.Ext.Get(addon.ExtKeySubsidy) == "S" {
			g := &Subsidy{
				SubsidyDescription: a.Description,
				SubsidyAmount:      amount(a.Amount),
			}
			if a.Percent != nil {
				g.SubsidyRate = a.Percent.StringWithoutSymbol()
			}
			grants = append(grants, g)
		} else {
			na := &PaymentOnAccount{
				PaymentOnAccountAmount: amount(a.Amount),
			}
			if a.Date != nil {
				na.PaymentOnAccountDate = *a.Date
			}
			regular = append(regular, na)
		}
	}
	if len(regular) > 0 {
		it.PaymentsOnAccount = &PaymentsOnAccount{
			PaymentOnAccount: regular,
		}
	}
	if len(grants) > 0 {
		it.Subsidies = &Subsidies{
			Subsidy: grants,
		}
	}
}

/*
func (it *InvoiceTotals) setOutlays(outlays []*bill.Outlay) {
	os := make([]*ReimbursableExpense, 0)
	for _, o := range outlays {
		no := &ReimbursableExpense{
			IssueDate:                  o.Date,
			InvoiceNumber:              o.Code,
			InvoiceSeriesCode:          o.Series,
			ReimbursableExpensesAmount: o.Amount.String(),
		}
		if o.Supplier != nil {
			tid := o.Supplier.TaxID
			if tid != nil {
				no.ReimbursableExpensesSellerParty = newTaxID(tid.Code, tid.Country, false)
			}
		}
		os = append(os, no)
	}
	if len(os) > 0 {
		it.ReimbursableExpenses = &ReimbursableExpenses{
			ReimbursableExpenses: os,
		}
	}
}
*/
