package facturae_test

import (
	"testing"

	"github.com/invopop/gobl.es.facturae/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaymentsInfo(t *testing.T) {
	t.Run("should contain payment details information", func(t *testing.T) {
		doc, err := test.LoadGOBL("invoice-vat.json")
		require.NoError(t, err)

		xmlInvoice := doc.Invoices.List[0]

		pd := xmlInvoice.PaymentDetails
		require.NotNil(t, pd)
		require.NotNil(t, pd.Installments)
		require.NotNil(t, pd.Installments[0])
		pi := pd.Installments[0]
		assert.Equal(t, "2021-12-30", pi.InstallmentDueDate)
		assert.Equal(t, "5084.42", pi.InstallmentAmount)
		assert.Equal(t, "04", pi.PaymentMeans)
		assert.Equal(t, "ES25 0188 2570 7185 4470 4761", pi.AccountToBeCredited.IBAN)
		assert.Contains(t, pi.CollectionAdditionalInformation, "payment term note")
	})
}
