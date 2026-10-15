package facturae

import (
	"github.com/invopop/gobl/org"
)

// ContactDetails store info about how to contact the party
type ContactDetails struct {
	Telephone                string `xml:"Telephone,omitempty"`
	TeleFax                  string `xml:"TeleFax,omitempty"`
	WebAddress               string `xml:"WebAddress,omitempty"`
	ElectronicMail           string `xml:"ElectronicMail,omitempty"`
	ContactPersons           string `xml:"ContactPersons,omitempty"`
	CnoCnae                  string `xml:"CnoCnae,omitempty"`     // Code of economic activity from INE
	INETownCode              string `xml:"INETownCode,omitempty"` // Code of town allocated by INE
	AdditionalContactDetails string `xml:"AdditionalContactDetails,omitempty"`
}

func newContactDetails(party *org.Party) *ContactDetails {
	email := ""
	if len(party.Emails) > 0 {
		email = party.Emails[0].Address
	}

	phoneNumber := ""
	if len(party.Telephones) > 0 {
		phoneNumber = party.Telephones[0].Number
	}

	contactPerson := ""
	if len(party.People) > 0 {
		contactPerson = party.People[0].Name.Given + " " + party.People[0].Name.Surname
	}

	return &ContactDetails{
		ElectronicMail: email,
		Telephone:      phoneNumber,
		ContactPersons: contactPerson,
	}
}
