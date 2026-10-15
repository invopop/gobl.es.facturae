# GOBL FacturaE

Convert GOBL invoices to and from Spain's [FacturaE](https://www.facturae.gob.es/Paginas/Index.aspx) 3.2.2 format, and define the `es-facturae-v3` GOBL addon.

Copyright [Invopop Ltd.](https://invopop.com) 2023. Released publicly under the [Apache License Version 2.0](LICENSE). For commercial licenses please contact the [dev team at invopop](mailto:dev@invopop.com). In order to accept contributions to this library we will require transferring copyrights to Invopop Ltd.

[![Lint](https://github.com/invopop/gobl.es.facturae/actions/workflows/lint.yaml/badge.svg)](https://github.com/invopop/gobl.es.facturae/actions/workflows/lint.yaml)
[![Test Go](https://github.com/invopop/gobl.es.facturae/actions/workflows/test.yaml/badge.svg)](https://github.com/invopop/gobl.es.facturae/actions/workflows/test.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/invopop/gobl.es.facturae)](https://goreportcard.com/report/github.com/invopop/gobl.es.facturae)
[![GoDoc](https://godoc.org/github.com/invopop/gobl.es.facturae?status.svg)](https://godoc.org/github.com/invopop/gobl.es.facturae)
![Latest Tag](https://img.shields.io/github/v/tag/invopop/gobl.es.facturae)

## Addon

The [`addon`](./addon) package defines the `es-facturae-v3` GOBL addon: the FacturaE extensions (document type, invoice class, correction reason and subsidy), the scenarios that set them from the invoice type and tags, and the validation rules. Documents that declare the addon need the package imported to be calculated or validated. Importing the root package imports it too:

```golang
import _ "github.com/invopop/gobl.es.facturae/addon"
```

## Usage

The package uses the same terms as the other GOBL converters:

| Function | Direction |
| --- | --- |
| `Export(env, opts...)` | GOBL envelope to FacturaE document |
| `Encode(doc)` | FacturaE document to XML bytes |
| `Decode(data)` | XML bytes to FacturaE document |
| `Import(doc)` | FacturaE document to GOBL envelope |

### Export

```golang
doc, err := facturae.Export(env)
if err != nil {
    panic(err)
}

data, err := facturae.Encode(doc)
if err != nil {
    panic(err)
}
```

The envelope is not changed: taxes included in prices are removed, and credit notes inverted, on a copy.

FacturaE documents are usually signed. Use a certificate to sign the document as follows:

```golang
// import from github.com/invopop/xmldsig
cert, err := xmldsig.LoadCertificate(filename, password)
if err != nil {
    panic(err)
}

doc, err := facturae.Export(env, facturae.WithCertificate(cert))
```

`WithTimestamp`, `WithThirdParty` and `WithSigning` add a timestamp, an issuing third party, and further signing options.

### Import

```golang
doc, err := facturae.Decode(data)
if err != nil {
    panic(err)
}

env, err := facturae.Import(doc)
```

`Import` provides a calculated envelope with an invoice that declares the `es-facturae-v3` addon. Signatures are read but not verified.

### GOBL convert register

Importing this package registers the `es-facturae-v3` format with the GOBL [`convert`](https://github.com/invopop/gobl/tree/main/convert) register, for importing and exporting invoices:

```golang
import (
    "github.com/invopop/gobl/convert"
    _ "github.com/invopop/gobl.es.facturae"
)

out, err := convert.Export(env, "es-facturae-v3")
res, err := convert.Import(data) // res.Envelope
```

Invoices must include the `es-facturae-v3` addon and a customer to be exported. Otherwise the export is refused with `convert.ErrNotSupported` and rules faults, such as `ES-FACTURAE-V3-BILL-INVOICE-01` on `$addons`. Documents exported through the register are not signed: use `Export` with `WithCertificate` to sign them. Attachments are not embedded yet, so exports with `convert.WithAttachments` are refused.

### Command line

This package no longer includes a command-line tool. The `gobl` CLI from [gobl.dev](https://github.com/invopop/gobl.dev) converts to and from FacturaE through the GOBL `convert` register. Its output is not signed.

```bash
go install github.com/invopop/gobl.dev/cmd/gobl@latest

gobl convert invoice.json invoice.xml --to es-facturae-v3
gobl convert invoice.xml
```

## Import limitations

Import covers the FacturaE 3.2.2 that this package exports. Documents from other issuers may use parts of the schema it does not read yet.

- Only FacturaE 3.2.2 is detected. FacturaE 3.2.1 uses a different namespace.
- Documents must contain a single invoice. Batches of several invoices are refused with `ErrUnsupportedBatch`.
- Tax rates are read as percentages. For Spanish suppliers, the rate key, such as `general` or `general+eqs`, is recovered when the percentage and surcharge match a rate on the invoice's date.
- Address lines are read as the street, the number, and any further parts as extra street details.
- Some details are not written to FacturaE, so they cannot be read back: the names of FACe administrative centres, due date percentages, descriptions of payments on account, and the order of advances.
- Totals are calculated by GOBL rather than taken from the document.

## Testing

Test data lives in `test/data`:

- `convert/*.json`: GOBL envelopes, exported to the signed FacturaE documents in `convert/out/*.xml`.
- `parse/*.xml`: FacturaE documents, imported to the GOBL envelopes in `parse/out/*.json`.

Run the tests, and update the expected output after a change, with:

```bash
go test ./...
go test ./... --update
```

The `xsdvalidate` build tag validates exported documents against the FacturaE schema in `test/schema` when updating them. It needs the libxml2 development libraries (`libxml2-dev` on Debian and Ubuntu, `libxml2` on Homebrew):

```bash
go test -tags xsdvalidate ./... --update
```

`TestSignature` checks each signed export as a receiver would, from the encoded bytes alone: every reference digest, the XAdES signing certificate and the signature value. It also checks that changed documents are refused, and that `Decode` reads the signature back. The documents in `convert/out` are indented for review, which breaks their signatures. Use `Encode` for documents that will be sent.

Test certificates are in `test/certificates`. The Spanish government [validator](https://face.gob.es/es/facturas/validar-visualizar-facturas) checks documents and their signatures.

## Considerations

- FacturaE requires taxes separate from prices, so taxes included in prices are removed before export. This recalculates amounts from unit prices, and may differ by a cent from the original totals.
- FacturaE requires a due date for each payment instruction. Payment instructions only appear in the document when the invoice has due dates.
