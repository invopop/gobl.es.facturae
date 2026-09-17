package facturae

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/pay"
)

// PaymentDetails contains info about how the invoice should
// be paid
type PaymentDetails struct {
	Installments []*Installment `xml:"Installment"`
}

// Installment contains info about each of the payment terms
type Installment struct {
	InstallmentDueDate              string
	InstallmentAmount               string
	PaymentMeans                    string       `xml:",omitempty"`
	AccountToBeCredited             *BankAccount `xml:",omitempty"`
	DebitReconciliationReference    string       `xml:",omitempty"`
	AccountToBeDebited              *BankAccount `xml:",omitempty"`
	CollectionAdditionalInformation string       `xml:",omitempty"`
}

// BankAccount contains info needed to pay by transfer or direct debit
type BankAccount struct {
	IBAN                  string   `xml:",omitempty"`
	AccountNumber         string   `xml:",omitempty"`
	BranchInSpainAddress  *Address `xml:",omitempty"`
	OverseasBranchAddress *Address `xml:",omitempty"`
	BIC                   string   `xml:",omitempty"`
}

// TODO: move this to the GOBL project directly.
var facturaePaymentMethodCodes = map[cbc.Key]string{
	pay.MeansKeyCash:           "01",
	pay.MeansKeyDirectDebit:    "02",
	pay.MeansKeyCreditTransfer: "04",
	pay.MeansKeyCard:           "19",
	pay.MeansKeyOnline:         "13",
}

func newPaymentDetails(paymentInfo *bill.PaymentDetails) *PaymentDetails {
	if paymentInfo == nil {
		return nil
	}
	terms := paymentInfo.Terms
	if terms == nil {
		return nil
	}
	if len(terms.DueDates) == 0 {
		return nil
	}

	instructions := paymentInfo.Instructions

	xmlInstallments := make([]*Installment, len(terms.DueDates))
	for i, installment := range terms.DueDates {
		xmlInstallment := &Installment{
			InstallmentDueDate:              installment.Date.String(),
			InstallmentAmount:               optionalAmount(installment.Amount),
			PaymentMeans:                    facturaePaymentMethodCodes[instructions.Key],
			CollectionAdditionalInformation: mergeNotes(paymentInfo.Terms.Notes, installment.Notes),
		}

		if instructions.Key == pay.MeansKeyCreditTransfer {
			if len(instructions.CreditTransfer) > 0 {
				xmlInstallment.AccountToBeCredited = newCreditBankAccount(instructions.CreditTransfer[0])
			}
		}

		if instructions.Key == pay.MeansKeyDirectDebit {
			xmlInstallment.AccountToBeDebited = newDebitBankAccount(instructions.DirectDebit)
			xmlInstallment.DebitReconciliationReference = instructions.DirectDebit.Ref
		}

		if instructions.Key == pay.MeansKeyOnline {
			if len(instructions.Online) > 0 {
				if len(xmlInstallment.CollectionAdditionalInformation) > 0 {
					xmlInstallment.CollectionAdditionalInformation += "\n"
				}
				xmlInstallment.CollectionAdditionalInformation += instructions.Online[0].URL
			}
		}

		xmlInstallments[i] = xmlInstallment
	}

	return &PaymentDetails{
		Installments: xmlInstallments,
	}
}

func newCreditBankAccount(info *pay.CreditTransfer) *BankAccount {
	if info == nil {
		return nil
	}
	return &BankAccount{
		IBAN:          info.IBAN.String(),
		BIC:           info.BIC.String(),
		AccountNumber: info.Number.String(),
	}
}

func newDebitBankAccount(info *pay.DirectDebit) *BankAccount {
	if info == nil {
		return nil
	}
	return &BankAccount{
		AccountNumber: info.Account,
	}
}

func mergeNotes(termNotes string, installmentNotes string) string {
	notes := ""
	if termNotes != "" {
		notes = termNotes
		if installmentNotes != "" {
			notes += "\n"
		}
	}

	return notes + installmentNotes
}
