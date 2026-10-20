package facturae

import (
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/rules"
	"github.com/invopop/gobl/rules/is"
)

// partyRules ensures a party has what FacturaE requires to describe it: an
// address with a country, and a person when it is an individual. Each party
// is given its own codes, starting from the first one provided.
func partyRules(name string, codes ...rules.Code) []rules.Def {
	return []rules.Def{
		rules.Field("addresses",
			rules.Assert(codes[0], name+" address is required", is.Present),
		),
		rules.Assert(codes[1], name+" address country is required", is.Func("address country", partyHasAddressCountry)),
		rules.Assert(codes[2], name+" person is required for individuals", is.Func("individual person", partyHasIndividualPerson)),
	}
}

// IndividualTaxCode returns true when the tax code identifies a natural
// person rather than a legal entity, which FacturaE determines from its first
// character.
// https://asepyme.com/diferencias-entre-cif-nif-dni
func IndividualTaxCode(code cbc.Code) bool {
	if code == "" {
		return false
	}
	switch code[0] {
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'K', 'L', 'M', 'X', 'Y', 'Z':
		return true
	}
	return false
}

func partyHasAddressCountry(val any) bool {
	p, ok := val.(*org.Party)
	if !ok || p == nil || len(p.Addresses) == 0 {
		return true // missing addresses handled by their own rule
	}
	return p.Addresses[0] != nil && !p.Addresses[0].Country.Empty()
}

func partyHasIndividualPerson(val any) bool {
	p, ok := val.(*org.Party)
	if !ok || p == nil || p.TaxID == nil || !IndividualTaxCode(p.TaxID.Code) {
		return true
	}
	return len(p.People) > 0 && p.People[0] != nil && p.People[0].Name != nil
}

func partyHasTaxID(val any) bool {
	p, ok := val.(*org.Party)
	return ok && p != nil && p.TaxID != nil
}
