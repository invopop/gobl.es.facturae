package facturae

import (
	"strings"

	addon "github.com/invopop/gobl.es.facturae/addon"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
)

// TaxID stores fiscal id
type TaxID struct {
	PersonTypeCode          string `xml:"PersonTypeCode"`
	ResidenceTypeCode       string `xml:"ResidenceTypeCode"`
	TaxIdentificationNumber string `xml:"TaxIdentificationNumber"`
}

// newTaxID builds the tax identification information of a party. Numbers are
// preceded by the country code for parties resident elsewhere in the EU, and
// for Spanish parties in intra-community operations.
func newTaxID(taxNumber cbc.Code, countryCode l10n.TaxCountryCode, intraCommunity bool) *TaxID {
	rtc := residenceTypeCode(countryCode)
	num := strings.ReplaceAll(taxNumber.String(), "-", "")
	if (rtc == "U" || intraCommunity) && !strings.HasPrefix(num, countryCode.String()) {
		num = countryCode.String() + num
	}
	return &TaxID{
		PersonTypeCode:          personTypeCode(taxNumber),
		ResidenceTypeCode:       rtc,
		TaxIdentificationNumber: num,
	}
}

// intraCommunity is true when the supplier and customer are tax resident in
// different EU member states.
func intraCommunity(inv *bill.Invoice) bool {
	if inv.Supplier == nil || inv.Supplier.TaxID == nil || inv.Customer == nil || inv.Customer.TaxID == nil {
		return false
	}
	s, c := inv.Supplier.TaxID.Country, inv.Customer.TaxID.Country
	return s != c && residesInEU(s) && residesInEU(c)
}

func personTypeCode(taxNumber cbc.Code) string {
	if addon.IndividualTaxCode(taxNumber) {
		return "F"
	}
	return "J"
}

func residenceTypeCode(countryCode l10n.TaxCountryCode) string {
	if residesInSpain(countryCode) {
		return "R"
	} else if residesInEU(countryCode) {
		return "U"
	} else {
		return "E"
	}
}

func residesInSpain(countryCode l10n.TaxCountryCode) bool {
	return countryCode == l10n.ES.Tax()
}

func residesInEU(countryCode l10n.TaxCountryCode) bool {
	switch countryCode.Code() {
	case
		l10n.BE, // Belgium
		l10n.BG, // Bulgaria
		l10n.CZ, // Czechia
		l10n.DK, // Denmark
		l10n.DE, // Germany
		l10n.EE, // Estonia
		l10n.IE, // Ireland
		l10n.GR, // Greece
		l10n.EL, // Greece (tax code)
		l10n.ES, // Spain
		l10n.FR, // France
		l10n.HR, // Croatia
		l10n.IT, // Italia
		l10n.CY, // Cyprus
		l10n.LV, // Latvia
		l10n.LT, // Lithuania
		l10n.LU, // Luxembourg
		l10n.HU, // Hungary
		l10n.MT, // Malta
		l10n.NL, // Netherlands
		l10n.AT, // Austria
		l10n.PL, // Poland
		l10n.PT, // Portugal
		l10n.RO, // Romania
		l10n.SI, // Slovenia
		l10n.SK, // Slovakia
		l10n.FI, // Finland
		l10n.SE: // Sweden
		return true
	}
	return false
}
