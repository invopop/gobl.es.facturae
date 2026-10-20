package facturae

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
)

func TestResidenceTypeCode(t *testing.T) {
	assert.Equal(t, "R", residenceTypeCode(l10n.ES.Tax()))
	assert.Equal(t, "U", residenceTypeCode(l10n.EL.Tax()))
	assert.Equal(t, "U", residenceTypeCode(l10n.PT.Tax()))
	assert.Equal(t, "E", residenceTypeCode(l10n.US.Tax()))
}

func TestNewTaxIDPrefix(t *testing.T) {
	assert.Equal(t, "FR12345678901", newTaxID("12345678901", "FR", false).TaxIdentificationNumber)
	assert.Equal(t, "FR12345678901", newTaxID("FR12345678901", "FR", true).TaxIdentificationNumber)
	assert.Equal(t, "EL123456789", newTaxID("123456789", "EL", false).TaxIdentificationNumber)
	assert.Equal(t, "B23103039", newTaxID("B23103039", "ES", false).TaxIdentificationNumber)
	assert.Equal(t, "ESB23103039", newTaxID("B23103039", "ES", true).TaxIdentificationNumber)
	assert.Equal(t, "123456789", newTaxID("12-3456789", "US", false).TaxIdentificationNumber)
}

func TestIntraCommunity(t *testing.T) {
	inv := func(s, c l10n.TaxCountryCode) *bill.Invoice {
		return &bill.Invoice{
			Supplier: &org.Party{TaxID: &tax.Identity{Country: s}},
			Customer: &org.Party{TaxID: &tax.Identity{Country: c}},
		}
	}
	assert.True(t, intraCommunity(inv("ES", "FR")))
	assert.True(t, intraCommunity(inv("EL", "ES")))
	assert.False(t, intraCommunity(inv("ES", "ES")))
	assert.False(t, intraCommunity(inv("ES", "US")))
	assert.False(t, intraCommunity(&bill.Invoice{Supplier: &org.Party{TaxID: &tax.Identity{Country: "ES"}}}))
}
