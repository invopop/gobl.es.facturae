package facturae

import (
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/org"
)

// Address stores location information for addresses in Spain
type Address struct {
	Address     string `xml:"Address,omitempty"`
	PostCode    string `xml:"PostCode,omitempty"`
	Town        string `xml:"Town,omitempty"`
	Province    string `xml:"Province,omitempty"`
	CountryCode string `xml:"CountryCode,omitempty"`
}

// OverseasAddress stores location information for addresses outside Spain
type OverseasAddress struct {
	Address         string `xml:"Address,omitempty"`
	PostCodeAndTown string `xml:"PostCodeAndTown,omitempty"`
	Province        string `xml:"Province,omitempty"`
	CountryCode     string `xml:"CountryCode,omitempty"`
}

func newAddress(address *org.Address) *Address {
	return &Address{
		Address:     addressLine1(address),
		PostCode:    address.Code.String(),
		Town:        address.Locality,
		Province:    address.Region,
		CountryCode: countryCode(address.Country),
	}
}

func newOverseasAddress(address *org.Address) *OverseasAddress {
	postCodeAndTown := address.Locality
	if address.Code != "" {
		postCodeAndTown = address.Code.String() + " " + address.Locality
	}
	return &OverseasAddress{
		Address:         addressLine1(address),
		PostCodeAndTown: postCodeAndTown,
		Province:        address.Region,
		CountryCode:     countryCode(address.Country),
	}
}

func addressLine1(address *org.Address) string {
	if address.PostOfficeBox != "" {
		return address.PostOfficeBox
	}

	return joinNonEmpty(", ", address.Street, address.Number, address.Block, address.Floor, address.Door)
}

// ES is the ISO Alpha-3 code used for Spain
const ES string = "ESP"

func countryCode(country l10n.ISOCountryCode) string {
	cd := l10n.Countries().Code(country.Code())
	if cd == nil {
		return ""
	}
	return cd.Alpha3
}
