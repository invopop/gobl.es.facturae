package facturae

import (
	"fmt"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
)

func goblLines(items *Items) ([]*bill.Line, error) {
	if items == nil {
		return nil, nil
	}
	lines := make([]*bill.Line, 0, len(items.InvoiceLine))
	for _, il := range items.InvoiceLine {
		line, err := goblLine(il)
		if err != nil {
			return nil, err
		}
		lines = append(lines, line)
	}
	return lines, nil
}

func goblLine(il *InvoiceLine) (*bill.Line, error) {
	qty, err := parseAmount(il.Quantity)
	if err != nil {
		return nil, err
	}
	price, err := parseAmount(il.UnitPriceWithoutTax)
	if err != nil {
		return nil, err
	}
	line := &bill.Line{
		Quantity: qty,
		Item: &org.Item{
			Name:  il.ItemDescription,
			Price: &price,
		},
	}
	if line.Discounts, err = goblLineDiscounts(il.DiscountsAndRebates); err != nil {
		return nil, err
	}
	if line.Charges, err = goblLineCharges(il.Charges); err != nil {
		return nil, err
	}
	if line.Taxes, err = goblTaxes(il.TaxesOutputs, il.TaxesWithheld); err != nil {
		return nil, err
	}
	return line, nil
}

// goblTaxes converts the taxes of a line into GOBL tax combos.
func goblTaxes(summaries ...*TaxSummary) (tax.Set, error) {
	var set tax.Set
	for _, s := range summaries {
		if s == nil {
			continue
		}
		for _, t := range s.List {
			c, err := goblTaxCombo(t)
			if err != nil {
				return nil, err
			}
			set = append(set, c)
		}
	}
	return set, nil
}

func goblTaxCombo(t *Tax) (*tax.Combo, error) {
	cat, ok := taxCodeCategory(t.Code)
	if !ok {
		return nil, fmt.Errorf("unsupported tax type code %q", t.Code)
	}
	c := &tax.Combo{Category: cat}
	var err error
	if c.Percent, err = parsePercent(t.Rate); err != nil {
		return nil, err
	}
	if c.Surcharge, err = parsePercent(t.Surcharge); err != nil {
		return nil, err
	}
	return c, nil
}

// taxCodeCategory provides the GOBL tax category for a FacturaE tax type code.
func taxCodeCategory(code string) (cbc.Code, bool) {
	for cat, c := range categoryTaxCodeMap {
		if c == code {
			return cat, true
		}
	}
	return "", false
}

func goblLineDiscounts(ds *Discounts) ([]*bill.LineDiscount, error) {
	if ds == nil {
		return nil, nil
	}
	list := make([]*bill.LineDiscount, 0, len(ds.Items))
	for _, d := range ds.Items {
		ld := &bill.LineDiscount{Reason: d.Reason}
		var err error
		if ld.Percent, err = parsePercent(d.Rate); err != nil {
			return nil, err
		}
		if ld.Amount, err = parseAmount(d.Amount); err != nil {
			return nil, err
		}
		list = append(list, ld)
	}
	return list, nil
}

func goblLineCharges(cs *Charges) ([]*bill.LineCharge, error) {
	if cs == nil {
		return nil, nil
	}
	list := make([]*bill.LineCharge, 0, len(cs.Items))
	for _, c := range cs.Items {
		lc := &bill.LineCharge{Reason: c.Reason}
		var err error
		if lc.Percent, err = parsePercent(c.Rate); err != nil {
			return nil, err
		}
		if lc.Amount, err = parseAmount(c.Amount); err != nil {
			return nil, err
		}
		list = append(list, lc)
	}
	return list, nil
}

func goblDiscounts(ds *Discounts) ([]*bill.Discount, error) {
	if ds == nil {
		return nil, nil
	}
	list := make([]*bill.Discount, 0, len(ds.Items))
	for _, d := range ds.Items {
		nd := &bill.Discount{Reason: d.Reason}
		var err error
		if nd.Percent, err = parsePercent(d.Rate); err != nil {
			return nil, err
		}
		if nd.Amount, err = parseAmount(d.Amount); err != nil {
			return nil, err
		}
		list = append(list, nd)
	}
	return list, nil
}

func goblCharges(cs *Charges) ([]*bill.Charge, error) {
	if cs == nil {
		return nil, nil
	}
	list := make([]*bill.Charge, 0, len(cs.Items))
	for _, c := range cs.Items {
		nc := &bill.Charge{Reason: c.Reason}
		var err error
		if nc.Percent, err = parsePercent(c.Rate); err != nil {
			return nil, err
		}
		if nc.Amount, err = parseAmount(c.Amount); err != nil {
			return nil, err
		}
		list = append(list, nc)
	}
	return list, nil
}

// goblRateKeys names the Spanish rate of each line tax whose percentage and
// surcharge match the rate on the invoice's date, as FacturaE only provides
// the percentages.
func goblRateKeys(inv *bill.Invoice) {
	if inv.Supplier == nil || inv.Supplier.TaxID == nil || inv.Supplier.TaxID.Country != l10n.ES.Tax() {
		return
	}
	regime := tax.RegimeDefFor(l10n.ES)
	date := inv.IssueDate
	if inv.ValueDate != nil {
		date = *inv.ValueDate
	}
	for _, line := range inv.Lines {
		for _, c := range line.Taxes {
			if c.Rate == "" && c.Percent != nil {
				c.Rate = rateKeyFor(regime.CategoryDef(c.Category), date, c)
			}
		}
	}
}

func rateKeyFor(cd *tax.CategoryDef, date cal.Date, c *tax.Combo) cbc.Key {
	if cd == nil {
		return ""
	}
	for _, rd := range cd.Rates {
		v := rd.Value(date, c.Ext)
		if v == nil || v.Disabled || !v.Percent.Equals(*c.Percent) {
			continue
		}
		if (v.Surcharge == nil) != (c.Surcharge == nil) {
			continue
		}
		if v.Surcharge != nil && !v.Surcharge.Equals(*c.Surcharge) {
			continue
		}
		return rd.Rate
	}
	return ""
}
