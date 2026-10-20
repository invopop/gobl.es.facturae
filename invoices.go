package facturae

import (
	addon "github.com/invopop/gobl.es.facturae/addon"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
)

// Invoices contains info about a batch of invoices.
// In our case there will only be one invoice per batch
type Invoices struct {
	List []*Invoice `xml:"Invoice"`
}

// Invoice contains info about a single invoice
type Invoice struct {
	InvoiceHeader    *InvoiceHeader    `xml:"InvoiceHeader"`
	InvoiceIssueData *InvoiceIssueData `xml:"InvoiceIssueData"`
	TaxesOutputs     *TaxSummary       `xml:"TaxesOutputs,omitempty"`
	TaxesWithheld    *TaxSummary       `xml:"TaxesWithheld,omitempty"`
	InvoiceTotals    *InvoiceTotals    `xml:"InvoiceTotals"`
	Items            *Items            `xml:"Items"`
	PaymentDetails   *PaymentDetails   `xml:"PaymentDetails,omitempty"`
	LegalLiterals    *LegalLiterals    `xml:"LegalLiterals,omitempty"`
	AdditionalData   *AdditionalData   `xml:"AdditionalData,omitempty"`
}

// InvoiceHeader contains general information of a single invoice
type InvoiceHeader struct {
	InvoiceNumber       string      `xml:"InvoiceNumber"`
	InvoiceSeriesCode   string      `xml:"InvoiceSeriesCode,omitempty"`
	InvoiceDocumentType string      `xml:"InvoiceDocumentType"`
	InvoiceClass        string      `xml:"InvoiceClass"`
	Corrective          *Corrective `xml:"Corrective,omitempty"`
}

// PeriodDates is used in corrective tax periods to define a date
// range.
type PeriodDates struct {
	StartDate string `xml:"StartDate"`
	EndDate   string `xml:"EndDate"`
}

// InvoiceIssueData contains information about dates, lang and currencies
type InvoiceIssueData struct {
	IssueDate                    string               `xml:"IssueDate"`
	OperationDate                string               `xml:"OperationDate,omitempty"`
	PlaceOfIssue                 string               `xml:"PlaceOfIssue,omitempty"`
	InvoiceingPeriod             *PeriodDates         `xml:"InvoiceingPeriod,omitempty"`
	InvoiceCurrencyCode          string               `xml:"InvoiceCurrencyCode"`
	ExchangeRateDetails          *ExchangeRateDetails `xml:"ExchangeRateDetails,omitempty"`
	TaxCurrencyCode              string               `xml:"TaxCurrencyCode"`
	LanguageName                 string               `xml:"LanguageName"`                           // FIXME: [JUANJO] are we going to support multiple languages?
	InvoiceDescription           string               `xml:"InvoiceDescription,omitempty"`           // TODO
	ReceiverTransactionReference string               `xml:"ReceiverTransactionReference,omitempty"` // TODO
	FileReference                string               `xml:"FileReference,omitempty"`                // TODO
	ReceiverContractReference    string               `xml:"ReceiverContractReference,omitempty"`    // TODO
}

// LegalLiterals contains an array of legal texts to add to the Invoice in certain situations.
type LegalLiterals struct {
	LegalReference []string `xml:"LegalReference"`
}

// ExchangeRateDetails describes how to exchange from the invoices currency
// to euros.
type ExchangeRateDetails struct {
	ExchangeRate     string `xml:"ExchangeRate"`
	ExchangeRateDate string `xml:"ExchangeRateDate"`
}

// NewInvoice creates a new invoice with facturae format
func (d *Document) newInvoice(invoice *bill.Invoice) *Invoice {
	valueDate := &invoice.IssueDate
	if invoice.ValueDate != nil {
		valueDate = invoice.ValueDate
	}
	rate := euroRate(invoice)
	xmlInvoice := &Invoice{
		InvoiceHeader: newInvoiceHeader(invoice),
		InvoiceIssueData: &InvoiceIssueData{
			IssueDate:           invoice.IssueDate.String(),
			OperationDate:       valueDate.String(),
			InvoiceCurrencyCode: string(invoice.Currency),
			ExchangeRateDetails: newExchangeRateDetails(rate, *valueDate),
			TaxCurrencyCode:     string(invoice.Currency),
			LanguageName:        "es",
		},
		InvoiceTotals:  newInvoiceTotals(invoice),
		AdditionalData: newAdditionalData(invoice),
		PaymentDetails: newPaymentDetails(invoice.Payment),
		LegalLiterals:  newLegalLiterals(invoice),
		Items:          newItems(invoice.Lines, invoice.Totals.Taxes, rate),
	}
	if rate != nil {
		xmlInvoice.InvoiceIssueData.TaxCurrencyCode = string(currency.EUR)
	}
	xmlInvoice.setTaxes(invoice.Totals.Taxes, rate)

	return xmlInvoice
}

// euroRate provides the rate that converts the invoice's amounts into euros,
// the tax currency, or nil when the invoice is already in euros.
func euroRate(inv *bill.Invoice) *currency.ExchangeRate {
	return currency.MatchExchangeRate(inv.ExchangeRates, inv.Currency, currency.EUR)
}

// newExchangeRateDetails describes the rate to euros. Without a publication
// date, the rate is taken as the one in force on the operation date, as
// required by Art. 79.eleven of Law 37/1992.
func newExchangeRateDetails(rate *currency.ExchangeRate, operation cal.Date) *ExchangeRateDetails {
	if rate == nil {
		return nil
	}
	date := operation
	if rate.At != nil {
		date = rate.At.Date()
	}
	r := rate.Amount
	if r.Exp() > 8 {
		r = r.Rescale(8) // the schema allows up to eight decimals
	}
	return &ExchangeRateDetails{
		ExchangeRate:     r.String(),
		ExchangeRateDate: date.String(),
	}
}

// setTaxes performs a set of steps to convert the GOBL tax list into something
// that FacturaE expects.
func (inv *Invoice) setTaxes(taxes *tax.Total, rate *currency.ExchangeRate) {
	if taxes == nil {
		return
	}
	regular := make([]*Tax, 0)
	retained := make([]*Tax, 0)
	// First loop for bases
	for _, ct := range taxes.Categories {
		for _, rt := range ct.Rates {
			percent := rt.Percent
			if percent == nil {
				percent = num.NewPercentage(0, 0)
			}
			tax := &Tax{
				Code:   categoryTaxCodeMap[ct.Code],
				Rate:   percent.StringWithoutSymbol(),
				Base:   makeAmount(rt.Base, rate),
				Amount: makeAmount(rt.Amount, rate),
			}
			if ct.Retained {
				retained = append(retained, tax)
			} else {
				if rt.Surcharge != nil {
					st := rt.Surcharge
					p := st.Percent
					p = p.Rescale(4) // we need 2 decimal places
					tax.Surcharge = p.StringWithoutSymbol()
					v := makeAmount(st.Amount, rate)
					tax.SurchargeAmount = &v
				}
				regular = append(regular, tax)
			}
		}
	}
	if len(regular) > 0 {
		inv.TaxesOutputs = &TaxSummary{
			List: regular,
		}
	}
	if len(retained) > 0 {
		inv.TaxesWithheld = &TaxSummary{
			List: retained,
		}
	}
}

func newInvoiceHeader(inv *bill.Invoice) *InvoiceHeader {
	h := &InvoiceHeader{
		InvoiceNumber:     inv.Code.String(),
		InvoiceSeriesCode: inv.Series.String(),
	}

	if inv.Tax != nil {
		h.InvoiceDocumentType = inv.Tax.Ext.Get(addon.ExtKeyDocType).String()
		h.InvoiceClass = inv.Tax.Ext.Get(addon.ExtKeyInvoiceClass).String()
	}

	// Only one preceding document currently supported
	if len(inv.Preceding) > 0 {
		h.Corrective = newCorrective(inv)
	}

	return h
}

func newLegalLiterals(inv *bill.Invoice) *LegalLiterals {
	lits := make([]string, 0)
	for _, n := range inv.Notes {
		if n.Key == org.NoteKeyLegal {
			lits = append(lits, n.Text)
		}
	}
	if len(lits) == 0 {
		return nil
	}
	return &LegalLiterals{
		LegalReference: lits,
	}
}

func newPeriodDates(p *cal.Period) *PeriodDates {
	if p == nil {
		return nil
	}
	return &PeriodDates{
		StartDate: p.Start.String(),
		EndDate:   p.End.String(),
	}
}
