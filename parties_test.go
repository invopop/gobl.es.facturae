package facturae_test

import (
	"testing"

	"github.com/invopop/gobl.es.facturae/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPartiesInfo(t *testing.T) {
	t.Run("should contain the provider party info", func(t *testing.T) {
		doc, err := test.LoadGOBL("invoice-vat-irpf.json")
		require.NoError(t, err)

		assert.Equal(t, "F", doc.Parties.Seller.TaxID.PersonTypeCode)
		assert.Equal(t, "R", doc.Parties.Seller.TaxID.ResidenceTypeCode)
		assert.Equal(t, "37221735F", doc.Parties.Seller.TaxID.TaxIdentificationNumber)
		assert.Equal(t, "Maria Remedios", doc.Parties.Seller.Individual.Name)
		assert.Equal(t, "Sanchez", doc.Parties.Seller.Individual.FirstSurname)
		assert.Equal(t, "Nuñez", doc.Parties.Seller.Individual.SecondSurname)
		assert.Equal(t, "Campo Real, 74", doc.Parties.Seller.Individual.AddressInSpain.Address)
		assert.Equal(t, "28023", doc.Parties.Seller.Individual.AddressInSpain.PostCode)
		assert.Equal(t, "Torrejón De La Calzada", doc.Parties.Seller.Individual.AddressInSpain.Town)
		assert.Equal(t, "Madrid", doc.Parties.Seller.Individual.AddressInSpain.Province)
		assert.Equal(t, "ESP", doc.Parties.Seller.Individual.AddressInSpain.CountryCode)
		assert.Equal(t, "msohrjnb3@caramail.com", doc.Parties.Seller.Individual.ContactDetails.ElectronicMail)
		assert.Equal(t, "+34612123123", doc.Parties.Seller.Individual.ContactDetails.Telephone)
	})
}

func TestPartiesInfoCustomer(t *testing.T) {
	t.Run("should contain the customer party info", func(t *testing.T) {
		doc, err := test.LoadGOBL("invoice-vat.json")
		require.NoError(t, err)

		assert.Equal(t, "J", doc.Parties.Buyer.TaxID.PersonTypeCode)
		assert.Equal(t, "R", doc.Parties.Buyer.TaxID.ResidenceTypeCode)
		assert.Equal(t, "B77436020", doc.Parties.Buyer.TaxID.TaxIdentificationNumber)
		assert.Equal(t, "Moniward Sl", doc.Parties.Buyer.LegalEntity.CorporateName)
		assert.Equal(t, "Plaza Horno, 35", doc.Parties.Buyer.LegalEntity.AddressInSpain.Address)
		assert.Equal(t, "45083", doc.Parties.Buyer.LegalEntity.AddressInSpain.PostCode)
		assert.Equal(t, "Nombela", doc.Parties.Buyer.LegalEntity.AddressInSpain.Town)
		assert.Equal(t, "Toledo", doc.Parties.Buyer.LegalEntity.AddressInSpain.Province)
		assert.Equal(t, "ESP", doc.Parties.Buyer.LegalEntity.AddressInSpain.CountryCode)
		assert.Equal(t, "bfn25xf3p@lycos.co.uk", doc.Parties.Buyer.LegalEntity.ContactDetails.ElectronicMail)
	})
}

func TestAdministrativeCentres(t *testing.T) {
	t.Run("should contain administrative centres for FACe invoices", func(t *testing.T) {
		doc, err := test.LoadGOBL("invoice-face.json")
		require.NoError(t, err)

		centres := doc.Parties.Buyer.AdministrativeCentres
		require.NotNil(t, centres)
		require.Len(t, centres.Centres, 3)

		// Role 01 - Oficina Contable (from customer identities)
		assert.Equal(t, "L01280796", centres.Centres[0].CentreCode)
		assert.Equal(t, "01", centres.Centres[0].RoleTypeCode)
		assert.Equal(t, "Plaza de la Villa, 4", centres.Centres[0].AddressInSpain.Address)
		assert.Equal(t, "28005", centres.Centres[0].AddressInSpain.PostCode)
		assert.Equal(t, "Madrid", centres.Centres[0].AddressInSpain.Town)

		// Role 02 - Órgano Gestor (from delivery.receiver)
		assert.Equal(t, "LA0007407", centres.Centres[1].CentreCode)
		assert.Equal(t, "02", centres.Centres[1].RoleTypeCode)
		assert.Equal(t, "Calle Alcalá, 3", centres.Centres[1].AddressInSpain.Address)
		assert.Equal(t, "28014", centres.Centres[1].AddressInSpain.PostCode)

		// Role 03 - Unidad Tramitadora (from ordering.buyer)
		assert.Equal(t, "LA0007408", centres.Centres[2].CentreCode)
		assert.Equal(t, "03", centres.Centres[2].RoleTypeCode)
		assert.Equal(t, "Gran Vía, 10", centres.Centres[2].AddressInSpain.Address)
		assert.Equal(t, "28013", centres.Centres[2].AddressInSpain.PostCode)
	})

	t.Run("should not include administrative centres when not present", func(t *testing.T) {
		doc, err := test.LoadGOBL("invoice-vat.json")
		require.NoError(t, err)

		assert.Nil(t, doc.Parties.Buyer.AdministrativeCentres)
	})
}

func TestOverseasAddressIndividual(t *testing.T) {
	t.Run("should contain overseas address for individual supplier", func(t *testing.T) {
		doc, err := test.LoadGOBL("invoice-overseas-individual.json")
		require.NoError(t, err)

		seller := doc.Parties.Seller
		require.NotNil(t, seller.Individual)
		require.NotNil(t, seller.Individual.OverseasAddress)
		assert.Nil(t, seller.Individual.AddressInSpain)

		// Verify overseas address fields
		assert.Equal(t, "Rue de la Paix, 42", seller.Individual.OverseasAddress.Address)
		assert.Equal(t, "75002 Paris", seller.Individual.OverseasAddress.PostCodeAndTown)
		assert.Equal(t, "Île-de-France", seller.Individual.OverseasAddress.Province)
		assert.Equal(t, "FRA", seller.Individual.OverseasAddress.CountryCode)
	})
}
