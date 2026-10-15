package facturae

import (
	"fmt"
	"strings"
	"time"

	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/num"
)

// parseAmount reads a FacturaE decimal.
func parseAmount(s string) (num.Amount, error) {
	a, err := num.AmountFromString(strings.TrimSpace(s))
	if err != nil {
		return num.Amount{}, fmt.Errorf("parsing amount %q: %w", s, err)
	}
	return a, nil
}

// parseOptionalAmount reads a FacturaE decimal that may be empty.
func parseOptionalAmount(s string) (*num.Amount, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}
	a, err := parseAmount(s)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// parsePercent reads a FacturaE rate, which is written as a number of
// percentage points, such as "21.00".
func parsePercent(s string) (*num.Percentage, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	p, err := num.PercentageFromString(s + "%")
	if err != nil {
		return nil, fmt.Errorf("parsing rate %q: %w", s, err)
	}
	return &p, nil
}

// parseDate reads a FacturaE date.
func parseDate(s string) (*cal.Date, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return nil, fmt.Errorf("parsing date %q: %w", s, err)
	}
	d := cal.DateOf(t)
	return &d, nil
}

// countryFromAlpha3 provides the ISO country code for a FacturaE country
// code, which uses ISO 3166-1 alpha-3.
func countryFromAlpha3(code string) l10n.ISOCountryCode {
	for _, cd := range l10n.Countries() {
		if cd.Alpha3 == code {
			return l10n.ISOCountryCode(cd.Code)
		}
	}
	return ""
}
