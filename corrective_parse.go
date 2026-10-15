package facturae

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"

	addon "github.com/invopop/gobl.es.facturae/addon"
)

// goblPreceding converts the corrected invoice's details. Its tax period is
// kept only when it differs from the issue date, which is the default on
// export.
func goblPreceding(c *Corrective) (*org.DocumentRef, error) {
	ref := &org.DocumentRef{
		Series: cbc.Code(c.InvoiceSeriesCode),
		Code:   cbc.Code(c.InvoiceNumber),
		Reason: c.AdditionalReasonDescription,
	}
	var err error
	if ref.IssueDate, err = parseDate(c.InvoiceIssueDate); err != nil {
		return nil, err
	}
	if c.ReasonCode != "" {
		ref.Ext = tax.ExtensionsOf(cbc.CodeMap{addon.ExtKeyCorrection: cbc.Code(c.ReasonCode)})
	}
	if c.TaxPeriod != nil {
		start, err := parseDate(c.TaxPeriod.StartDate)
		if err != nil {
			return nil, err
		}
		end, err := parseDate(c.TaxPeriod.EndDate)
		if err != nil {
			return nil, err
		}
		if !sameDates(ref.IssueDate, start, end) {
			ref.Period = &cal.Period{Start: start, End: end}
		}
	}
	return ref, nil
}

func sameDates(ref *cal.Date, dates ...*cal.Date) bool {
	for _, d := range dates {
		if ref == nil || d == nil || *d != *ref {
			return false
		}
	}
	return true
}

// correctionType provides the type of a corrective invoice: a full
// replacement is corrective, and corrected items only are a credit or debit
// note, depending on the sign of the total.
func correctionType(c *Corrective, negative bool) cbc.Key {
	switch {
	case c != nil && c.CorrectionMethod == "01":
		return bill.InvoiceTypeCorrective
	case negative:
		return bill.InvoiceTypeCreditNote
	case c != nil && c.CorrectionMethod == "02":
		return bill.InvoiceTypeDebitNote
	}
	return bill.InvoiceTypeCorrective
}
