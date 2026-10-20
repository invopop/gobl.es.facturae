package facturae

import (
	"fmt"
	"strings"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/regimes/es"
	"github.com/invopop/gobl/rules"
	"github.com/invopop/gobl/rules/is"
	"github.com/invopop/gobl/tax"
)

var invoiceCorrectionDefinitions = tax.CorrectionSet{
	{
		Schema: bill.ShortSchemaInvoice,
		Extensions: []cbc.Key{
			ExtKeyCorrection,
		},
	},
}

func billInvoiceRules() *rules.Set {
	return rules.For(new(bill.Invoice),
		rules.Assert("07", "invoice must be in EUR or provide exchange rate for conversion (FacturaE: InvoiceIssueData/ExchangeRateDetails; Art. 10.1 RD 1496/2003)", currency.CanConvertTo(currency.EUR)),
		rules.Field("supplier", partyRules("supplier", "SellerParty", "08", "09", "10")...),
		rules.Field("customer",
			// FacturaE only includes customers with a tax ID
			rules.When(
				is.Func("tax ID present", partyHasTaxID),
				partyRules("customer", "BuyerParty", "11", "12", "13")...,
			),
			rules.Field("tax_id",
				rules.When(
					tax.IdentityIn(es.CountryCode),
					rules.Field("code",
						rules.Assert("01", "customer tax ID code is required for Spanish customers (FacturaE: BuyerParty/TaxIdentification/TaxIdentificationNumber)", is.Present),
					),
				),
			),
		),
		rules.Field("tax",
			rules.Assert("02", "tax object is required with ext document type and invoice classes (FacturaE: InvoiceHeader/InvoiceDocumentType, InvoiceHeader/InvoiceClass)", is.Present),
			rules.Field("ext",
				rules.Assert("03", fmt.Sprintf("tax ext require '%s' and '%s' extensions (FacturaE: InvoiceHeader/InvoiceDocumentType, InvoiceHeader/InvoiceClass)", ExtKeyDocType, ExtKeyInvoiceClass),
					tax.ExtensionsRequire(
						ExtKeyDocType,
						ExtKeyInvoiceClass,
					),
				),
			),
		),
		rules.When(
			bill.InvoiceTypeIn(es.InvoiceCorrectionTypes...),
			rules.Field("preceding",
				rules.Assert("04", fmt.Sprintf("preceding document reference is required for %s invoices (FacturaE: InvoiceHeader/Corrective)", strings.Join(cbc.KeyStrings(es.InvoiceCorrectionTypes), ", ")),
					is.Present,
				),
				rules.Each(
					rules.Field("issue_date",
						rules.Assert("05", "preceding document issue date is required (FacturaE: Corrective/TaxPeriod)", is.Present),
					),
					rules.Field("ext",
						rules.Assert("06", fmt.Sprintf("preceding document ext require '%s' extension (FacturaE: Corrective/ReasonCode)", ExtKeyCorrection),
							tax.ExtensionsRequire(ExtKeyCorrection),
						),
					),
				),
			),
		),
	)
}
