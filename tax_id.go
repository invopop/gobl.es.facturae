package facturae

import (
	"strings"

	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
)

// TaxID stores fiscal id
type TaxID struct {
	PersonTypeCode          string `xml:"PersonTypeCode"`
	ResidenceTypeCode       string `xml:"ResidenceTypeCode"`
	TaxIdentificationNumber string `xml:"TaxIdentificationNumber"`
}

// newTaxID builds the tax identification information of a party
func newTaxID(taxNumber cbc.Code, countryCode l10n.TaxCountryCode) *TaxID {
	return &TaxID{
		PersonTypeCode:          personTypeCode(taxNumber),
		ResidenceTypeCode:       residenceTypeCode(countryCode),
		TaxIdentificationNumber: strings.ReplaceAll(taxNumber.String(), "-", ""),
	}
}

func personTypeCode(taxNumber cbc.Code) string {
	// Depending on the first character of the number we can know
	// whether is a legal entity or a person
	// https://asepyme.com/diferencias-entre-cif-nif-dni
	switch rune(taxNumber[0]) {
	case
		'0',
		'1',
		'2',
		'3',
		'4',
		'5',
		'6',
		'7',
		'8',
		'9',
		'K',
		'L',
		'M',
		'X',
		'Y',
		'Z':
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
