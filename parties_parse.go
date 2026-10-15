package facturae

import (
	"strings"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
)

// goblParty converts a FacturaE party, a company or a person, into a GOBL
// party.
func goblParty(p *Party) *org.Party {
	if p == nil {
		return nil
	}
	out := new(org.Party)
	switch {
	case p.LegalEntity != nil:
		le := p.LegalEntity
		out.Name = le.CorporateName
		out.Addresses = goblAddresses(le.AddressInSpain, le.OverseasAddress)
		goblContactDetails(out, le.ContactDetails, true)
	case p.Individual != nil:
		in := p.Individual
		name := &org.Name{
			Given:    in.Name,
			Surname:  in.FirstSurname,
			Surname2: in.SecondSurname,
		}
		out.Name = joinNonEmpty(" ", name.Given, name.Surname, name.Surname2)
		out.People = []*org.Person{{Name: name}}
		out.Addresses = goblAddresses(in.AddressInSpain, in.OverseasAddress)
		goblContactDetails(out, in.ContactDetails, false)
	}
	out.TaxID = goblTaxID(p.TaxID, out.Addresses)
	return out
}

// goblTaxID provides the tax identity of a party. FacturaE only says whether
// a party resides in Spain, so others take their country from their address.
func goblTaxID(t *TaxID, addrs []*org.Address) *tax.Identity {
	if t == nil {
		return nil
	}
	id := &tax.Identity{Code: cbc.Code(t.TaxIdentificationNumber)}
	switch {
	case t.ResidenceTypeCode == "R":
		id.Country = l10n.ES.Tax()
	case len(addrs) > 0:
		id.Country = l10n.TaxCountryCode(addrs[0].Country)
	}
	return id
}

func goblContactDetails(out *org.Party, cd *ContactDetails, people bool) {
	if cd == nil {
		return
	}
	if cd.ElectronicMail != "" {
		out.Emails = []*org.Email{{Address: cd.ElectronicMail}}
	}
	if cd.Telephone != "" {
		out.Telephones = []*org.Telephone{{Number: cd.Telephone}}
	}
	if people {
		if name := strings.TrimSpace(cd.ContactPersons); name != "" {
			given, surname, _ := strings.Cut(name, " ")
			out.People = []*org.Person{{Name: &org.Name{Given: given, Surname: surname}}}
		}
	}
}

func goblAddresses(es *Address, overseas *OverseasAddress) []*org.Address {
	switch {
	case es != nil:
		addr := goblAddressLine(es.Address)
		addr.Code = cbc.Code(es.PostCode)
		addr.Locality = es.Town
		addr.Region = es.Province
		addr.Country = countryFromAlpha3(es.CountryCode)
		return []*org.Address{addr}
	case overseas != nil:
		addr := goblAddressLine(overseas.Address)
		addr.Code, addr.Locality = splitPostCodeAndTown(overseas.PostCodeAndTown)
		addr.Region = overseas.Province
		addr.Country = countryFromAlpha3(overseas.CountryCode)
		return []*org.Address{addr}
	}
	return nil
}

// goblAddressLine reads the street and number from an address line written
// as "street, number[, block, floor, door]".
func goblAddressLine(line string) *org.Address {
	parts := strings.Split(line, ", ")
	addr := &org.Address{Street: parts[0]}
	if len(parts) > 1 {
		addr.Number = parts[1]
	}
	if len(parts) > 2 {
		addr.StreetExtra = strings.Join(parts[2:], ", ")
	}
	return addr
}

// splitPostCodeAndTown separates a leading post code, which contains digits,
// from the town.
func splitPostCodeAndTown(s string) (cbc.Code, string) {
	code, town, ok := strings.Cut(strings.TrimSpace(s), " ")
	if !ok || !strings.ContainsAny(code, "0123456789") {
		return "", strings.TrimSpace(s)
	}
	return cbc.Code(code), town
}

// goblAdministrativeCentres places the FACe centres of the buyer: the
// accounting office as a customer identity, the managing body as the
// delivery receiver, and the processing unit as the ordering buyer.
func goblAdministrativeCentres(inv *bill.Invoice, buyer *Party) {
	if buyer == nil || buyer.AdministrativeCentres == nil || inv.Customer == nil {
		return
	}
	for _, c := range buyer.AdministrativeCentres.Centres {
		switch c.RoleTypeCode {
		case RoleTypeFiscal:
			inv.Customer.Identities = append(inv.Customer.Identities, &org.Identity{
				Scope: org.IdentityScopeTax,
				Code:  cbc.Code(c.CentreCode),
			})
		case RoleTypeReceiver:
			inv.Delivery = &bill.DeliveryDetails{Receiver: goblCentreParty(c)}
		case RoleTypePayer:
			inv.Ordering = &bill.Ordering{Buyer: goblCentreParty(c)}
		}
	}
}

func goblCentreParty(c *AdministrativeCentre) *org.Party {
	return &org.Party{
		Identities: []*org.Identity{{Code: cbc.Code(c.CentreCode)}},
		Addresses:  goblAddresses(c.AddressInSpain, c.OverseasAddress),
	}
}

func joinNonEmpty(sep string, parts ...string) string {
	list := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			list = append(list, p)
		}
	}
	return strings.Join(list, sep)
}
