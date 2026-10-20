package facturae

import (
	"testing"

	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/num"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakeAmount(t *testing.T) {
	rate := &currency.ExchangeRate{From: "USD", To: "EUR", Amount: num.MakeAmount(882457, 6)}
	assert.Equal(t, Amount{TotalAmount: "100.00"}, makeAmount(num.MakeAmount(100, 0), nil))
	assert.Equal(t, Amount{TotalAmount: "100.00", EquivalentInEuros: "88.25"}, makeAmount(num.MakeAmount(100, 0), rate))
}

func TestNewExchangeRateDetails(t *testing.T) {
	op := cal.MakeDate(2021, 12, 8)
	assert.Nil(t, newExchangeRateDetails(nil, op))

	rate := &currency.ExchangeRate{From: "USD", To: "EUR", Amount: num.MakeAmount(882457, 6)}
	assert.Equal(t, &ExchangeRateDetails{ExchangeRate: "0.882457", ExchangeRateDate: "2021-12-08"}, newExchangeRateDetails(rate, op), "operation date without a publication date")

	rate.At = cal.NewDateTime(2021, 12, 7, 15, 0, 0)
	rate.Amount = num.MakeAmount(8824571234, 10)
	assert.Equal(t, &ExchangeRateDetails{ExchangeRate: "0.88245712", ExchangeRateDate: "2021-12-07"}, newExchangeRateDetails(rate, op))
}

func TestGoblExchangeRates(t *testing.T) {
	d := &InvoiceIssueData{ExchangeRateDetails: &ExchangeRateDetails{ExchangeRate: "0.882457", ExchangeRateDate: "2021-12-07"}}
	rates, err := goblExchangeRates(d, "USD")
	require.NoError(t, err)
	assert.Equal(t, []*currency.ExchangeRate{{
		From: "USD", To: "EUR", At: cal.NewDateTime(2021, 12, 7, 0, 0, 0), Amount: num.MakeAmount(882457, 6),
	}}, rates)

	rates, err = goblExchangeRates(d, "EUR")
	require.NoError(t, err)
	assert.Nil(t, rates, "euro invoices need no rate")

	d.ExchangeRateDetails.ExchangeRate = "x"
	_, err = goblExchangeRates(d, "USD")
	assert.ErrorContains(t, err, "exchange rate: parsing amount")
}
