package facturae

import (
	"errors"

	"github.com/invopop/gobl"
	addon "github.com/invopop/gobl.es.facturae/addon"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/convert"
	"github.com/invopop/gobl/i18n"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/rules"
	"github.com/invopop/gobl/rules/is"
	"github.com/invopop/gobl/schema"
)

// KeyFacturaE identifies the FacturaE 3.2.2 format in the GOBL convert
// register.
const KeyFacturaE cbc.Key = "es-facturae-v3"

// ErrImportNotSupported is returned when asked to import FacturaE documents,
// which this package does not do.
var ErrImportNotSupported = errors.New("facturae import not supported")

var invoiceSchema = schema.Lookup(bill.Invoice{})

// acceptRules lists the conditions an invoice must meet to be exported.
var acceptRules = convert.Rules(KeyFacturaE,
	rules.For(new(bill.Invoice),
		convert.RequireAddons("01", addon.V3),
		rules.Field("customer",
			rules.Assert("02", "requires a customer", is.Present),
		),
	),
)

func init() {
	convert.Register(converter{})
}

// converter registers the FacturaE format with the GOBL convert register,
// for export only. Exported documents are not signed.
type converter struct{}

func (converter) Formats() []*convert.Format {
	return []*convert.Format{
		{
			Key:       KeyFacturaE,
			Name:      i18n.NewString("FacturaE 3.2.2"),
			MIME:      "application/xml",
			Syntax:    "facturae",
			Countries: []l10n.Code{l10n.ES},
			Export:    []schema.ID{invoiceSchema},
		},
	}
}

func (converter) Detect(_ *convert.Input) cbc.Key {
	return cbc.KeyEmpty
}

func (converter) Import(_ cbc.Key, _ []byte) (*gobl.Envelope, error) {
	return nil, ErrImportNotSupported
}

// Accepts checks the invoice includes the FacturaE addon and a customer.
func (converter) Accepts(_ cbc.Key, env *gobl.Envelope) error {
	if fs := acceptRules.Validate(env.Extract()); fs != nil {
		return fs
	}
	return nil
}

func (converter) Export(_ cbc.Key, env *gobl.Envelope) ([]byte, error) {
	doc, err := Export(env)
	if err != nil {
		return nil, err
	}
	return Encode(doc)
}
