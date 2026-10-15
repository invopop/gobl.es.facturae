package facturae

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/pay"
	"github.com/invopop/gobl/regimes/es"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const badNumber = "abc"

func TestParseHelpers(t *testing.T) {
	_, err := parseAmount(badNumber)
	assert.Error(t, err)
	a, err := parseOptionalAmount("")
	assert.NoError(t, err)
	assert.Nil(t, a)
	_, err = parseOptionalAmount(badNumber)
	assert.Error(t, err)

	p, err := parsePercent("21.00")
	require.NoError(t, err)
	assert.Equal(t, "21.00%", p.String())
	p, err = parsePercent(" ")
	assert.NoError(t, err)
	assert.Nil(t, p)
	_, err = parsePercent(badNumber)
	assert.Error(t, err)

	d, err := parseDate("")
	assert.NoError(t, err)
	assert.Nil(t, d)
	_, err = parseDate("15/03/2024")
	assert.Error(t, err)

	assert.Equal(t, l10n.ISOCountryCode("ES"), countryFromAlpha3("ESP"))
	assert.Empty(t, countryFromAlpha3("XXX"))
}

func TestGoblParties(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		assert.Nil(t, goblParty(nil))
		assert.Nil(t, goblTaxID(nil, nil))
	})
	t.Run("legal entity", func(t *testing.T) {
		p := goblParty(&Party{
			TaxID: &TaxID{ResidenceTypeCode: "E", TaxIdentificationNumber: "123"},
			LegalEntity: &LegalEntity{
				CorporateName: "Acme Inc.",
				OverseasAddress: &OverseasAddress{
					Address:         "Main Street",
					PostCodeAndTown: "Springfield",
					CountryCode:     "USA",
				},
				ContactDetails: &ContactDetails{
					ElectronicMail: "a@example.com",
					Telephone:      "+1555",
					ContactPersons: "Jane Doe",
				},
			},
		})
		assert.Equal(t, "Acme Inc.", p.Name)
		assert.Equal(t, l10n.TaxCountryCode("US"), p.TaxID.Country)
		assert.Equal(t, "Springfield", p.Addresses[0].Locality)
		assert.Empty(t, p.Addresses[0].Code)
		assert.Equal(t, "Jane", p.People[0].Name.Given)
		assert.Equal(t, "Doe", p.People[0].Name.Surname)
		assert.Equal(t, "+1555", p.Telephones[0].Number)
	})
	t.Run("individual", func(t *testing.T) {
		p := goblParty(&Party{
			TaxID: &TaxID{ResidenceTypeCode: "U", TaxIdentificationNumber: "123"},
			Individual: &Individual{
				Name:           "Jean",
				FirstSurname:   "Dubois",
				ContactDetails: &ContactDetails{ContactPersons: "Jean Dubois"},
			},
		})
		assert.Equal(t, "Jean Dubois", p.Name)
		assert.Len(t, p.People, 1)
		assert.Empty(t, p.TaxID.Country, "no address to take the country from")
	})
	t.Run("address lines", func(t *testing.T) {
		a := goblAddressLine("Calle Mayor, 1, 2, B")
		assert.Equal(t, "Calle Mayor", a.Street)
		assert.Equal(t, "1", a.Number)
		assert.Equal(t, "2, B", a.StreetExtra)
		code, town := splitPostCodeAndTown("75002 Paris")
		assert.Equal(t, cbc.Code("75002"), code)
		assert.Equal(t, "Paris", town)
	})
	t.Run("administrative centres without buyer", func(t *testing.T) {
		inv := new(bill.Invoice)
		goblAdministrativeCentres(inv, nil)
		assert.Nil(t, inv.Delivery)
	})
}

func TestGoblLineErrors(t *testing.T) {
	base := func() *InvoiceLine {
		return &InvoiceLine{Quantity: "1", UnitPriceWithoutTax: "10"}
	}
	tests := map[string]func(l *InvoiceLine){
		"quantity":      func(l *InvoiceLine) { l.Quantity = badNumber },
		"price":         func(l *InvoiceLine) { l.UnitPriceWithoutTax = badNumber },
		"discount rate": func(l *InvoiceLine) { l.DiscountsAndRebates = &Discounts{Items: []*Discount{{Rate: badNumber}}} },
		"discount":      func(l *InvoiceLine) { l.DiscountsAndRebates = &Discounts{Items: []*Discount{{Amount: badNumber}}} },
		"charge rate":   func(l *InvoiceLine) { l.Charges = &Charges{Items: []*Charge{{Rate: badNumber}}} },
		"charge":        func(l *InvoiceLine) { l.Charges = &Charges{Items: []*Charge{{Amount: badNumber}}} },
		"tax code":      func(l *InvoiceLine) { l.TaxesOutputs = &TaxSummary{List: []*Tax{{Code: "99"}}} },
		"tax rate":      func(l *InvoiceLine) { l.TaxesOutputs = &TaxSummary{List: []*Tax{{Code: "01", Rate: badNumber}}} },
		"tax surcharge": func(l *InvoiceLine) {
			l.TaxesOutputs = &TaxSummary{List: []*Tax{{Code: "01", Rate: "21", Surcharge: badNumber}}}
		},
		"withheld taxes": func(l *InvoiceLine) { l.TaxesWithheld = &TaxSummary{List: []*Tax{{Code: "99"}}} },
	}
	for name, change := range tests {
		t.Run(name, func(t *testing.T) {
			l := base()
			change(l)
			_, err := goblLines(&Items{InvoiceLine: []*InvoiceLine{l}})
			assert.Error(t, err)
		})
	}
	lines, err := goblLines(nil)
	assert.NoError(t, err)
	assert.Nil(t, lines)
}

func TestGoblDocumentDiscountsAndCharges(t *testing.T) {
	ds, err := goblDiscounts(&Discounts{Items: []*Discount{{Reason: "Promo", Rate: "5", Amount: "10.00"}}})
	require.NoError(t, err)
	assert.Equal(t, "5%", ds[0].Percent.String())
	_, err = goblDiscounts(&Discounts{Items: []*Discount{{Rate: badNumber}}})
	assert.Error(t, err)
	_, err = goblDiscounts(&Discounts{Items: []*Discount{{Amount: badNumber}}})
	assert.Error(t, err)

	_, err = goblCharges(&Charges{Items: []*Charge{{Rate: badNumber}}})
	assert.Error(t, err)
	_, err = goblCharges(&Charges{Items: []*Charge{{Amount: badNumber}}})
	assert.Error(t, err)
}

func TestRateKeys(t *testing.T) {
	inv := &bill.Invoice{
		Supplier: nil,
		Lines:    []*bill.Line{{Taxes: tax.Set{{Category: tax.CategoryVAT, Percent: num.NewPercentage(210, 3)}}}},
	}
	goblRateKeys(inv)
	assert.Empty(t, inv.Lines[0].Taxes[0].Rate, "only for Spanish suppliers")
	assert.Empty(t, rateKeyFor(nil, inv.IssueDate, inv.Lines[0].Taxes[0]))
}

func TestGoblPayment(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		p, err := goblPayment(nil, nil)
		assert.NoError(t, err)
		assert.Nil(t, p)
	})
	t.Run("direct debit", func(t *testing.T) {
		p, err := goblPayment(&PaymentDetails{Installments: []*Installment{{
			InstallmentDueDate:           "2024-04-15",
			InstallmentAmount:            "100.00",
			PaymentMeans:                 "02",
			AccountToBeDebited:           &BankAccount{AccountNumber: "ES9121000418450200051332"},
			DebitReconciliationReference: "MANDATE",
		}}}, nil)
		require.NoError(t, err)
		assert.Equal(t, pay.MeansKeyDirectDebit, p.Instructions.Key)
		assert.Equal(t, cbc.Code("MANDATE"), p.Instructions.DirectDebit.Ref)
	})
	t.Run("unknown means", func(t *testing.T) {
		p, err := goblPayment(&PaymentDetails{Installments: []*Installment{{
			InstallmentDueDate: "2024-04-15",
			PaymentMeans:       "99",
		}}}, nil)
		require.NoError(t, err)
		assert.Nil(t, p.Instructions)
	})
	for name, in := range map[string]*Installment{
		"date":   {InstallmentDueDate: badNumber},
		"amount": {InstallmentDueDate: "2024-04-15", InstallmentAmount: badNumber},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := goblPayment(&PaymentDetails{Installments: []*Installment{in}}, nil)
			assert.Error(t, err)
		})
	}
	for name, totals := range map[string]*InvoiceTotals{
		"subsidy rate":       {Subsidies: &Subsidies{Subsidy: []*Subsidy{{SubsidyRate: badNumber}}}},
		"subsidy amount":     {Subsidies: &Subsidies{Subsidy: []*Subsidy{{SubsidyAmount: badNumber}}}},
		"payment on account": {PaymentsOnAccount: &PaymentsOnAccount{PaymentOnAccount: []*PaymentOnAccount{{PaymentOnAccountAmount: badNumber}}}},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := goblPayment(nil, totals)
			assert.Error(t, err)
		})
	}
}

func TestGoblCorrective(t *testing.T) {
	t.Run("period on the issue date", func(t *testing.T) {
		ref, err := goblPreceding(&Corrective{
			InvoiceIssueDate: "2024-03-15",
			TaxPeriod:        &PeriodDates{StartDate: "2024-03-15", EndDate: "2024-03-15"},
		})
		require.NoError(t, err)
		assert.Nil(t, ref.Period)
		assert.True(t, ref.Ext.IsZero())
	})
	for name, c := range map[string]*Corrective{
		"issue date":   {InvoiceIssueDate: badNumber},
		"period start": {TaxPeriod: &PeriodDates{StartDate: badNumber}},
		"period end":   {TaxPeriod: &PeriodDates{StartDate: "2024-03-01", EndDate: badNumber}},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := goblPreceding(c)
			assert.Error(t, err)
		})
	}
	t.Run("types", func(t *testing.T) {
		assert.Equal(t, bill.InvoiceTypeCorrective, correctionType(&Corrective{CorrectionMethod: "01"}, true))
		assert.Equal(t, bill.InvoiceTypeCreditNote, correctionType(nil, true))
		assert.Equal(t, bill.InvoiceTypeDebitNote, correctionType(&Corrective{CorrectionMethod: "02"}, false))
		assert.Equal(t, bill.InvoiceTypeCorrective, correctionType(nil, false))
	})
}

func TestDocumentTags(t *testing.T) {
	assert.Equal(t, []cbc.Key{tax.TagSimplified, es.TagSummary}, documentTags(&InvoiceHeader{InvoiceDocumentType: "FA", InvoiceClass: "OC"}))
	assert.Equal(t, []cbc.Key{tax.TagSelfBilled, es.TagCopy}, documentTags(&InvoiceHeader{InvoiceDocumentType: "AF", InvoiceClass: "CR"}))
	assert.Equal(t, []cbc.Key{es.TagCopy, es.TagSummary}, documentTags(&InvoiceHeader{InvoiceClass: "CC"}))
	assert.Empty(t, documentTags(&InvoiceHeader{InvoiceDocumentType: "FC", InvoiceClass: "OO"}))
}

func TestGoblInvoiceErrors(t *testing.T) {
	valid := func() *Document {
		return &Document{
			Parties: &Parties{Seller: &Party{}, Buyer: &Party{}},
			Invoices: &Invoices{List: []*Invoice{{
				InvoiceHeader:    &InvoiceHeader{InvoiceNumber: "1"},
				InvoiceIssueData: &InvoiceIssueData{IssueDate: "2024-03-15", InvoiceCurrencyCode: "EUR"},
			}}},
		}
	}
	_, err := goblInvoice(valid())
	require.NoError(t, err)

	tests := map[string]func(d *Document){
		"no invoices": func(d *Document) { d.Invoices = nil },
		"batch":       func(d *Document) { d.Invoices.List = append(d.Invoices.List, d.Invoices.List[0]) },
		"no parties":  func(d *Document) { d.Parties = nil },
		"no header":   func(d *Document) { d.Invoices.List[0].InvoiceHeader = nil },
		"issue date":  func(d *Document) { d.Invoices.List[0].InvoiceIssueData.IssueDate = badNumber },
		"no issue date": func(d *Document) {
			d.Invoices.List[0].InvoiceIssueData.IssueDate = ""
		},
		"operation date": func(d *Document) {
			d.Invoices.List[0].InvoiceIssueData.OperationDate = badNumber
		},
		"lines": func(d *Document) {
			d.Invoices.List[0].Items = &Items{InvoiceLine: []*InvoiceLine{{Quantity: badNumber}}}
		},
		"discounts": func(d *Document) {
			d.Invoices.List[0].InvoiceTotals = &InvoiceTotals{GeneralDiscounts: &Discounts{Items: []*Discount{{Amount: badNumber}}}}
		},
		"charges": func(d *Document) {
			d.Invoices.List[0].InvoiceTotals = &InvoiceTotals{GeneralSurcharges: &Charges{Items: []*Charge{{Amount: badNumber}}}}
		},
		"payment": func(d *Document) {
			d.Invoices.List[0].PaymentDetails = &PaymentDetails{Installments: []*Installment{{InstallmentDueDate: badNumber}}}
		},
		"corrective": func(d *Document) {
			d.Invoices.List[0].InvoiceHeader.Corrective = &Corrective{InvoiceIssueDate: badNumber}
		},
		"credit note": func(d *Document) {
			d.Invoices.List[0].InvoiceHeader.InvoiceClass = "OR"
			d.Invoices.List[0].InvoiceTotals = &InvoiceTotals{InvoiceTotal: "-10.00"}
			d.Invoices.List[0].Items = &Items{InvoiceLine: []*InvoiceLine{{Quantity: "-1", UnitPriceWithoutTax: "10", TaxesOutputs: &TaxSummary{List: []*Tax{{Code: "01", Rate: "21"}}}}}}
			d.Invoices.List[0].InvoiceIssueData.InvoiceCurrencyCode = "ZZZ"
		},
	}
	for name, change := range tests {
		t.Run(name, func(t *testing.T) {
			d := valid()
			change(d)
			_, err := goblInvoice(d)
			assert.Error(t, err)
		})
	}
	_, err = Import(&Document{})
	assert.ErrorIs(t, err, ErrUnsupportedBatch)
}

func TestDecodeErrors(t *testing.T) {
	_, err := Decode(nil)
	assert.Error(t, err)
	_, err = Decode([]byte(`<Facturae xmlns="urn:other"/>`))
	assert.ErrorIs(t, err, ErrUnsupportedDocument)
	_, err = Decode([]byte(`<fe:Facturae xmlns:fe="` + NamespaceFacturaE + `"><FileHeader><Batch><InvoicesCount>x</InvoicesCount></Batch></FileHeader></fe:Facturae>`))
	assert.Error(t, err)
	_, err = Export(nil)
	assert.Error(t, err)
}
