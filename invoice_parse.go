package facturae

import (
	"errors"
	"fmt"
	"strings"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/regimes/es"
	"github.com/invopop/gobl/schema"
	"github.com/invopop/gobl/tax"

	addon "github.com/invopop/gobl.es.facturae/addon"
)

// ErrUnsupportedBatch is returned when importing a document that does not
// contain exactly one invoice.
var ErrUnsupportedBatch = errors.New("facturae documents must contain a single invoice")

// Import converts the FacturaE document into a GOBL envelope holding an
// invoice with the es-facturae-v3 addon, calculated but not validated. The
// document must contain a single invoice.
func Import(doc *Document) (*gobl.Envelope, error) {
	inv, err := goblInvoice(doc)
	if err != nil {
		return nil, err
	}
	env := gobl.NewEnvelope()
	if env.Document, err = schema.NewObject(inv); err != nil {
		return nil, err
	}
	if err := env.Calculate(); err != nil {
		return nil, err
	}
	return env, nil
}

func goblInvoice(doc *Document) (*bill.Invoice, error) {
	if doc.Invoices == nil || len(doc.Invoices.List) != 1 {
		return nil, ErrUnsupportedBatch
	}
	if doc.Parties == nil {
		return nil, errors.New("facturae document has no parties")
	}
	xi := doc.Invoices.List[0]
	if xi.InvoiceHeader == nil || xi.InvoiceIssueData == nil {
		return nil, errors.New("facturae invoice has no header or issue data")
	}
	h := xi.InvoiceHeader
	d := xi.InvoiceIssueData

	inv := &bill.Invoice{
		Addons:   tax.WithAddons(addon.V3),
		Type:     bill.InvoiceTypeStandard,
		Series:   cbc.Code(h.InvoiceSeriesCode),
		Code:     cbc.Code(h.InvoiceNumber),
		Currency: currency.Code(d.InvoiceCurrencyCode),
	}
	var err error
	if inv.Supplier, err = goblParty(doc.Parties.Seller); err != nil {
		return nil, fmt.Errorf("seller: %w", err)
	}
	if inv.Customer, err = goblParty(doc.Parties.Buyer); err != nil {
		return nil, fmt.Errorf("buyer: %w", err)
	}
	goblAdministrativeCentres(inv, doc.Parties.Buyer)

	issueDate, err := parseDate(d.IssueDate)
	if err != nil {
		return nil, err
	}
	if issueDate == nil {
		return nil, errors.New("facturae invoice has no issue date")
	}
	inv.IssueDate = *issueDate
	operationDate, err := parseDate(d.OperationDate)
	if err != nil {
		return nil, err
	}
	if operationDate != nil && *operationDate != *issueDate {
		inv.ValueDate = operationDate
	}

	if inv.ExchangeRates, err = goblExchangeRates(d, inv.Currency); err != nil {
		return nil, err
	}

	inv.SetTags(documentTags(h)...)

	if inv.Lines, err = goblLines(xi.Items); err != nil {
		return nil, err
	}
	goblRateKeys(inv)
	if t := xi.InvoiceTotals; t != nil {
		if inv.Discounts, err = goblDiscounts(t.GeneralDiscounts); err != nil {
			return nil, err
		}
		if inv.Charges, err = goblCharges(t.GeneralSurcharges); err != nil {
			return nil, err
		}
	}
	if inv.Payment, err = goblPayment(xi.PaymentDetails, xi.InvoiceTotals); err != nil {
		return nil, err
	}
	inv.Notes = goblNotes(xi)

	if h.Corrective != nil || isCorrectiveClass(h.InvoiceClass) {
		if err := goblCorrection(inv, xi); err != nil {
			return nil, err
		}
	}
	return inv, nil
}

// goblCorrection sets the type and preceding document of a corrective
// invoice. Credit notes are written with negative amounts, so they are
// inverted back.
func goblCorrection(inv *bill.Invoice, xi *Invoice) error {
	h := xi.InvoiceHeader
	if h.Corrective != nil {
		ref, err := goblPreceding(h.Corrective)
		if err != nil {
			return err
		}
		inv.Preceding = []*org.DocumentRef{ref}
	}
	negative := xi.InvoiceTotals != nil && strings.HasPrefix(strings.TrimSpace(xi.InvoiceTotals.InvoiceTotal), "-")
	inv.Type = correctionType(h.Corrective, negative)
	if inv.Type != bill.InvoiceTypeCreditNote {
		return nil
	}
	if err := inv.Calculate(); err != nil {
		return fmt.Errorf("calculating credit note: %w", err)
	}
	if err := inv.Invert(); err != nil {
		return fmt.Errorf("inverting credit note: %w", err)
	}
	return nil
}

func isCorrectiveClass(class string) bool {
	return class == "OR" || class == "CR"
}

// documentTags provides the tags that the document type and invoice class
// stand for. The addon sets both extensions back from the tags.
func documentTags(h *InvoiceHeader) []cbc.Key {
	var tags []cbc.Key
	switch h.InvoiceDocumentType {
	case "FA":
		tags = append(tags, tax.TagSimplified)
	case "AF":
		tags = append(tags, tax.TagSelfBilled)
	}
	switch h.InvoiceClass {
	case "OC":
		tags = append(tags, es.TagSummary)
	case "CO", "CR":
		tags = append(tags, es.TagCopy)
	case "CC":
		tags = append(tags, es.TagCopy, es.TagSummary)
	}
	return tags
}

// goblNotes converts the legal literals into legal notes, and the additional
// information into a general note.
func goblNotes(xi *Invoice) []*org.Note {
	var notes []*org.Note
	if xi.LegalLiterals != nil {
		for _, l := range xi.LegalLiterals.LegalReference {
			notes = append(notes, &org.Note{Key: org.NoteKeyLegal, Text: l})
		}
	}
	if ad := xi.AdditionalData; ad != nil && ad.InvoiceAdditionalInformation != "" {
		notes = append(notes, &org.Note{Text: ad.InvoiceAdditionalInformation})
	}
	return notes
}

// goblExchangeRates reads the rate that converts the invoice currency into
// euros, published on the given date.
func goblExchangeRates(d *InvoiceIssueData, cur currency.Code) ([]*currency.ExchangeRate, error) {
	if d.ExchangeRateDetails == nil || cur == currency.EUR {
		return nil, nil
	}
	amount, err := parseAmount(d.ExchangeRateDetails.ExchangeRate)
	if err != nil {
		return nil, fmt.Errorf("exchange rate: %w", err)
	}
	rate := &currency.ExchangeRate{From: cur, To: currency.EUR, Amount: amount}
	date, err := parseDate(d.ExchangeRateDetails.ExchangeRateDate)
	if err != nil {
		return nil, fmt.Errorf("exchange rate: %w", err)
	}
	if date != nil {
		rate.At = cal.NewDateTime(date.Year, date.Month, date.Day, 0, 0, 0)
	}
	return []*currency.ExchangeRate{rate}, nil
}
