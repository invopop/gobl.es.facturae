package facturae

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/org"
)

// Administrative centre role type codes for FACe
const (
	RoleTypeFiscal   = "01" // Oficina Contable
	RoleTypeReceiver = "02" // Órgano Gestor
	RoleTypePayer    = "03" // Unidad Tramitadora
)

// Parties stores emitter and receiver of the invoice
type Parties struct {
	Seller *Party `xml:"SellerParty"`
	Buyer  *Party `xml:"BuyerParty"`
}

// Party stores the info for one of the participants of
// the invoice
type Party struct {
	TaxID                 *TaxID                 `xml:"TaxIdentification,omitempty"`
	AdministrativeCentres *AdministrativeCentres `xml:"AdministrativeCentres,omitempty"`
	LegalEntity           *LegalEntity           `xml:"LegalEntity,omitempty"`
	Individual            *Individual            `xml:"Individual,omitempty"`
}

// AdministrativeCentres contains a list of administrative centres
// required for invoicing Spanish public administrations (FACe)
type AdministrativeCentres struct {
	Centres []*AdministrativeCentre `xml:"AdministrativeCentre"`
}

// AdministrativeCentre represents an administrative unit within a
// public administration
type AdministrativeCentre struct {
	CentreCode      string           `xml:"CentreCode,omitempty"`
	RoleTypeCode    string           `xml:"RoleTypeCode,omitempty"`
	AddressInSpain  *Address         `xml:"AddressInSpain,omitempty"`
	OverseasAddress *OverseasAddress `xml:"OverseasAddress,omitempty"`
}

// LegalEntity stores party info for companies, associations, ...
type LegalEntity struct {
	CorporateName   string           `xml:"CorporateName"`
	AddressInSpain  *Address         `xml:"AddressInSpain,omitempty"`
	OverseasAddress *OverseasAddress `xml:"OverseasAddress,omitempty"`
	ContactDetails  *ContactDetails  `xml:"ContactDetails,omitempty"`
}

// Individual stores party info for people
type Individual struct {
	Name            string           `xml:"Name"`
	FirstSurname    string           `xml:"FirstSurname,omitempty"`
	SecondSurname   string           `xml:"SecondSurname,omitempty"`
	AddressInSpain  *Address         `xml:"AddressInSpain,omitempty"`
	OverseasAddress *OverseasAddress `xml:"OverseasAddress,omitempty"`
	ContactDetails  *ContactDetails  `xml:"ContactDetails,omitempty"`
}

// newIndividual creates info for people
func newIndividual(party *org.Party) *Individual {
	if len(party.People) == 0 {
		return nil
	}

	personInfo := party.People[0]
	individual := &Individual{
		Name:           personInfo.Name.Given,
		FirstSurname:   personInfo.Name.Surname,
		SecondSurname:  personInfo.Name.Surname2,
		ContactDetails: newContactDetails(party),
	}

	if len(party.Addresses) > 0 {
		addr := party.Addresses[0]
		if countryCode(addr.Country) == ES {
			individual.AddressInSpain = newAddress(addr)
		} else {
			individual.OverseasAddress = newOverseasAddress(addr)
		}
	}

	return individual
}

// newLegalEntity creates info for companies, associations, ...
func newLegalEntity(party *org.Party) *LegalEntity {
	entity := &LegalEntity{
		CorporateName:  party.Name,
		ContactDetails: newContactDetails(party),
	}

	if len(party.Addresses) > 0 {
		addr := party.Addresses[0]
		if countryCode(addr.Country) == ES {
			entity.AddressInSpain = newAddress(addr)
		} else {
			entity.OverseasAddress = newOverseasAddress(addr)
		}
	}

	return entity
}

// newAdministrativeCentres creates the administrative centres required for
// invoicing Spanish public administrations. The mapping is:
// - Role 01 (Oficina Contable): from customer identities
// - Role 02 (Órgano Gestor): from delivery.receiver
// - Role 03 (Unidad Tramitadora): from ordering.buyer
func newAdministrativeCentres(inv *bill.Invoice) *AdministrativeCentres {
	var centres []*AdministrativeCentre

	// Role 01 - Oficina Contable: from customer identities
	if inv.Customer != nil && len(inv.Customer.Identities) > 0 {
		for _, identity := range inv.Customer.Identities {
			if identity.Scope == org.IdentityScopeTax {
				centre := newAdministrativeCentre(
					identity.Code.String(),
					RoleTypeFiscal,
					inv.Customer,
				)
				if centre != nil {
					centres = append(centres, centre)
				}
			}
		}
	}

	// Role 02 - Órgano Gestor: from delivery.receiver
	if inv.Delivery != nil && inv.Delivery.Receiver != nil {
		if len(inv.Delivery.Receiver.Identities) > 0 {
			centre := newAdministrativeCentre(
				inv.Delivery.Receiver.Identities[0].Code.String(),
				RoleTypeReceiver,
				inv.Delivery.Receiver,
			)
			if centre != nil {
				centres = append(centres, centre)
			}
		}
	}

	// Role 03 - Unidad Tramitadora: from ordering.buyer
	if inv.Ordering != nil && inv.Ordering.Buyer != nil {
		if len(inv.Ordering.Buyer.Identities) > 0 {
			centre := newAdministrativeCentre(
				inv.Ordering.Buyer.Identities[0].Code.String(),
				RoleTypePayer,
				inv.Ordering.Buyer,
			)
			if centre != nil {
				centres = append(centres, centre)
			}
		}
	}

	if len(centres) == 0 {
		return nil
	}

	return &AdministrativeCentres{Centres: centres}
}

func newAdministrativeCentre(code, roleType string, party *org.Party) *AdministrativeCentre {
	if code == "" {
		return nil
	}

	centre := &AdministrativeCentre{
		CentreCode:   code,
		RoleTypeCode: roleType,
	}

	if len(party.Addresses) > 0 {
		addr := party.Addresses[0]
		if countryCode(addr.Country) == ES {
			centre.AddressInSpain = newAddress(addr)
		} else {
			centre.OverseasAddress = newOverseasAddress(addr)
		}
	}

	return centre
}
