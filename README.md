# GOBL to FacturaE Toolkit

Convert GOBL documents into the Spain's FacturaE format.

Copyright [Invopop Ltd.](https://invopop.com) 2023. Released publicly under the [Apache License Version 2.0](LICENSE). For commercial licenses please contact the [dev team at invopop](mailto:dev@invopop.com). In order to accept contributions to this library we will require transferring copyrights to Invopop Ltd.

[![Lint](https://github.com/invopop/gobl.es.facturae/actions/workflows/lint.yaml/badge.svg)](https://github.com/invopop/gobl.es.facturae/actions/workflows/lint.yaml)
[![Test Go](https://github.com/invopop/gobl.es.facturae/actions/workflows/test.yaml/badge.svg)](https://github.com/invopop/gobl.es.facturae/actions/workflows/test.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/invopop/gobl.facturae)](https://goreportcard.com/report/github.com/invopop/gobl.facturae)
[![GoDoc](https://godoc.org/github.com/invopop/gobl.facturae?status.svg)](https://godoc.org/github.com/invopop/gobl.facturae)
![Latest Tag](https://img.shields.io/github/v/tag/invopop/gobl.es.facturae)

## Usage

### Go

There are a couple of entry points to build a new FacturaE document. If you already have a GOBL Envelope available in Go, you could convert and output to a data file like this:

```golang
doc, err := facturae.NewInvoice(env)
if err != nil {
    panic(err)
}

data, err := doc.Bytes()
if err != nil {
    panic(err)
}

if err = os.WriteFile("./test.xml", data, 0644); err != nil {
    panic(err)
}
```

If you're loading from a file, you can use the `LoadGOBL` convenience method:

```golang
doc, err := facturae.LoadGOBL(file)
if err != nil {
    panic(err)
}
// do something with doc
```

Outputting to a FacturaE XML is most useful when the document is signed. Use a certificate to sign the document as follows:

```golang
// import from github.com/invopop/xmldsig
cert, err := xmldsig.LoadCertificate(filename, password)
if err != nil {
    panic(err)
}

doc, err := facturae.NewInvoice(env, facturae.WithCertificate(cert))
if err != nil {
    panic(err)
}
```

### CLI

The command line interface is useful when working with languages other than Go.

#### Installation

```bash
go install github.com/invopop/gobl.facturae/cmd/gobl.facturae@latest
```

#### Usage

Convert a GOBL JSON invoice to FacturaE XML:

```bash
gobl.facturae convert input.json output.xml
```

With a digital certificate:

```bash
gobl.facturae convert -c cert.p12 -p password input.json output.xml
```

The command also supports pipes:

```bash
cat input.json | gobl.facturae convert > output.xml
```

## Development

### Architecture Overview

The conversion process follows these steps:

1. **Load GOBL Envelope**: Parse the input JSON containing a GOBL invoice
2. **Remove Included Taxes**: If the invoice has `prices_include` set (e.g., VAT included in prices), the converter calls `invoice.RemoveIncludedTaxes()` to recalculate amounts from base prices. This is required because FacturaE expects taxes to be separate from base amounts.
3. **Invert Credit Notes**: If the document is a credit note, amounts are inverted to match FacturaE expectations
4. **Map to FacturaE Structure**: Convert GOBL structures to FacturaE XML elements
5. **Sign (Optional)**: Apply digital signature if a certificate is provided

**Important**: The `RemoveIncludedTaxes()` operation recalculates all amounts from item-level prices, which can introduce minor rounding differences (typically ±0.01) compared to the original GOBL totals. Test fixtures account for these expected differences.

### Testing

#### Dependencies

For XML schema validation during testing, this package uses [go-xsd-validate](https://github.com/terminalstatic/go-xsd-validate), which depends on the libxml2 C library. This dependency is **optional** and only required when running tests with the `xsdvalidate` build tag.

**Debian/Ubuntu:**
```bash
sudo apt-get install libxml2-dev
```

**macOS:**
```bash
brew install libxml2
```

Tests will run without libxml2, but schema validation will be skipped unless you explicitly enable it with the `-tags xsdvalidate` flag.

#### Running Tests

Run all tests (without schema validation):
```bash
go test ./...
```

Run tests with XSD schema validation (requires libxml2):
```bash
go test -tags xsdvalidate ./...
```

Run tests for a specific example:
```bash
go test -v -run "TestXMLGeneration/should_convert_invoice-vat.json"
```

#### Test Fixtures

Test data is organized in the `test/data/` directory:

- `*.json` - GOBL invoice envelopes (input)
- `out/*.xml` - Expected FacturaE XML output (fixtures)
- `schema/facturaev3_2_2.xsd` - Official FacturaE XML schema

#### Updating Fixtures

When you make changes that affect XML output, update the fixtures:

```bash
go test -tags xsdvalidate -run TestXMLGeneration --update
```

This will:
1. Convert all JSON examples to FacturaE XML
2. Validate each XML against the schema (requires libxml2)
3. Update the fixtures in `test/data/out/`

**Note**: The `-tags xsdvalidate` flag enables schema validation to ensure generated XML is valid according to the official FacturaE schema. Without this flag, fixtures will be updated but validation will be skipped.

#### Manual Testing with Certificates

For automated testing, XML documents are not signed. To generate signed documents for manual testing:

```bash
mage -v convertXML
```

Digital certificates for testing are available in `/test/certificates`.

### Validation

To validate generated XML documents and digital signatures, use the official Spanish government validator:
- https://face.gob.es/es/facturas/validar-visualizar-facturas

### Code Structure

- **Go structures** use the same naming conventions as the FacturaE XML schema where possible. This means XML tag names are often omitted in struct definitions, making it easier to map between Go code and XML output.
- **Amount handling** uses `num.Amount` from GOBL for precise decimal arithmetic, with the `MinimalString()` method used for XML output to avoid trailing zeros.

### Common Issues and Troubleshooting

#### Rounding Differences in Tests

When working with invoices that have `prices_include` tax settings, you may notice small rounding differences (typically ±0.01) between:
- The original GOBL `payable` amount
- The FacturaE `TotalOutstandingAmount`/`TotalExecutableAmount`

This is **expected behavior** because:
1. GOBL invoices with included taxes store the gross (tax-included) amounts
2. FacturaE requires net (tax-excluded) amounts
3. The conversion calls `RemoveIncludedTaxes()` which recalculates everything from item-level unit prices
4. This multi-step recalculation can accumulate minor rounding differences

**Solution**: Test fixtures should reflect the amounts **after** conversion, not the original GOBL amounts. Update fixtures using `go test --update`.

#### Schema Validation

To enable XSD schema validation during testing:

1. Install libxml2 development libraries (see Dependencies section above)
2. Run tests with the `-tags xsdvalidate` flag
3. Without this flag, tests will run but skip schema validation

This is particularly useful when updating fixtures to ensure generated XML conforms to the official FacturaE schema.

#### Test Failures After GOBL Updates

When upgrading the GOBL dependency:
1. Amount calculation logic may change
2. Run `go test --update` to regenerate all fixtures
3. Manually verify a few key examples to ensure conversion is still correct
4. Check the official FacturaE validator for complex cases

## Current Conversion Limitations

The FacturaE format is quite complex due to the number of local requirements in Spain.

- _Payment Information_: FacturaE requires each payment instruction to have a Due Date. The GOBL invoice allows these details to be independent. If you require payment instructions to appear on a FacturaE document, there must be a due date.
