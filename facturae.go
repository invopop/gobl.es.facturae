package facturae

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"

	"github.com/invopop/gobl"
	addon "github.com/invopop/gobl.es.facturae/addon"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/xmlctx"
	"github.com/invopop/xmldsig"
	"github.com/invopop/xmldsig/profiles/facturae"
)

// Namespaces used for FacturaE. DSig stuff is handled in the signatures.
const (
	NamespaceFacturaE = "http://www.facturae.gob.es/formato/Versiones/Facturaev3_2_2.xml"
	NamespaceDSig     = "http://www.w3.org/2000/09/xmldsig#"
)

// ErrUnsupportedDocument is returned when decoding data that is not a
// FacturaE 3.2.2 document.
var ErrUnsupportedDocument = errors.New("unsupported document: expected FacturaE 3.2.2")

// XAdES Signer Roles used for FacturaE
const (
	XAdESSupplier   xmldsig.XAdESSignerRole = "supplier"
	XAdESCustomer   xmldsig.XAdESSignerRole = "customer"
	XAdESThirdParty xmldsig.XAdESSignerRole = "third party"
)

// Document is a pseudo-model for containing the XML document being created.
type Document struct {
	env     *gobl.Envelope `xml:"-"` // Envelope to convert.
	invoice *bill.Invoice  `xml:"-"` // Invoice contained in envelope.

	XMLName     xml.Name `xml:"fe:Facturae"`
	FeNamespace string   `xml:"xmlns:fe,attr"`

	FileHeader *FileHeader        `xml:"FileHeader"`
	Parties    *Parties           `xml:"Parties"`
	Invoices   *Invoices          `xml:"Invoices"`
	Signature  *xmldsig.Signature `xml:"ds:Signature,omitempty"`
}

type options struct {
	certificate  *xmldsig.Certificate
	addTimestamp bool
	thirdParty   *ThirdParty
	signingOpts  []xmldsig.Option
}

// Option defines a callback configuration option used to customize the
// conversion to XML process
type Option func(*options)

// WithCertificate will use the provided certificate to sign the XML document.
func WithCertificate(cert *xmldsig.Certificate) Option {
	return func(opts *options) {
		opts.certificate = cert
	}
}

// WithTimestamp will ensure the XML document is timestamped
func WithTimestamp(val bool) Option {
	return func(opts *options) {
		opts.addTimestamp = val
	}
}

// WithThirdParty adds optional information about who is manipulating and signing
// the document.
func WithThirdParty(tp *ThirdParty) Option {
	return func(opts *options) {
		opts.thirdParty = tp
	}
}

// WithSigning allows passing additional xmldsig options that will be appended
// to the signing call. Useful to inject a deterministic signing time or other
// parameters in tests.
func WithSigning(opts ...xmldsig.Option) Option {
	return func(o *options) {
		o.signingOpts = append(o.signingOpts, opts...)
	}
}

// Export converts the invoice in the GOBL envelope into a FacturaE document,
// signed when a certificate is provided.
func Export(env *gobl.Envelope, opts ...Option) (*Document, error) {
	// prepare our options
	xmlOpts := new(options)
	for _, opt := range opts {
		opt(xmlOpts)
	}
	if env == nil || env.Document == nil {
		return nil, errors.New("expected an invoice")
	}
	// Work on a copy, as taxes included in prices are removed and credit
	// notes inverted.
	obj, err := env.Document.Clone()
	if err != nil {
		return nil, fmt.Errorf("copying document: %w", err)
	}
	invoice, ok := obj.Instance().(*bill.Invoice)
	if !ok {
		return nil, errors.New("expected an invoice")
	}

	if !addon.V3.In(invoice.GetAddons()...) {
		return nil, fmt.Errorf("missing %s addon", addon.V3.String())
	}

	if invoice.Customer == nil {
		return nil, errors.New("customer required")
	}

	// Make sure we're dealing with raw data
	if err := invoice.RemoveIncludedTaxes(); err != nil {
		return nil, fmt.Errorf("removing taxes: %w", err)
	}

	// Invert if we're dealing with a credit note
	if invoice.Type == bill.InvoiceTypeCreditNote {
		if err := invoice.Invert(); err != nil {
			return nil, fmt.Errorf("inverting invoice: %w", err)
		}
	}

	// Basic document headers
	d := &Document{
		env:         env,
		invoice:     invoice,
		FeNamespace: NamespaceFacturaE,
		Parties: &Parties{
			Seller: &Party{},
			Buyer:  &Party{},
		},
	}

	d.Parties.Seller.TaxID = newTaxID(invoice.Supplier.TaxID.Code, invoice.Supplier.TaxID.Country)
	if d.Parties.Seller.TaxID.PersonTypeCode == "F" {
		d.Parties.Seller.Individual = newIndividual(invoice.Supplier)
	} else {
		d.Parties.Seller.LegalEntity = newLegalEntity(invoice.Supplier)
	}

	if invoice.Customer != nil && invoice.Customer.TaxID != nil {
		// Simplified invoices are not supported by FacturaE, but this is where
		// they'd be handled if the situation changes.
		b := d.Parties.Buyer
		b.TaxID = newTaxID(invoice.Customer.TaxID.Code, invoice.Customer.TaxID.Country)
		b.AdministrativeCentres = newAdministrativeCentres(invoice)
		if b.TaxID.PersonTypeCode == "F" {
			b.Individual = newIndividual(invoice.Customer)
		} else {
			b.LegalEntity = newLegalEntity(invoice.Customer)
		}
	}

	d.Invoices = &Invoices{
		List: []*Invoice{d.newInvoice(invoice)},
	}

	d.FileHeader = newFileHeader(invoice, xmlOpts.thirdParty)

	if xmlOpts.certificate != nil {
		data, err := d.Canonical()
		if err != nil {
			return nil, fmt.Errorf("converting to canonincal format: %w", err)
		}

		dsigCfg := facturae.XMLDSigConfig()
		dsigCfg.OmitDataCanonicalizationTransform = true

		xadesCfg := facturae.XAdESConfig(
			xmldsig.XAdESConfig{
				Role:        XAdESThirdParty,
				Description: "Factura Electrónica",
				Policy: &xmldsig.XAdESPolicyConfig{
					URL:         "http://www.facturae.es/politica_de_firma_formato_facturae/politica_de_firma_formato_facturae_v3_1.pdf",
					Description: "Política de Firma FacturaE v3.1",
					Algorithm:   "http://www.w3.org/2000/09/xmldsig#sha1",
					Hash:        "Ohixl6upD6av8N7pEvDABhEL6hM=",
				},
			},
		)

		sigopts := []xmldsig.Option{
			xmldsig.WithDocID(env.Head.UUID.String()),
			xmldsig.WithXMLDSigConfig(dsigCfg),
			xmldsig.WithXAdESConfig(xadesCfg),
			xmldsig.WithCertificate(xmlOpts.certificate),
		}
		if xmlOpts.addTimestamp {
			sigopts = append(sigopts, xmldsig.WithTimestamp(xmldsig.TimestampFreeTSA))
		}
		sigopts = append(sigopts, xmlOpts.signingOpts...)
		sig, err := xmldsig.Sign(data, sigopts...)
		if err != nil {
			return nil, err
		}
		d.Signature = sig
	}

	return d, nil
}

// Decode parses FacturaE 3.2.2 XML data into a document. Signatures are
// read but not verified.
func Decode(data []byte) (*Document, error) {
	ns, err := rootNamespace(data)
	if err != nil {
		return nil, err
	}
	if ns != NamespaceFacturaE {
		return nil, ErrUnsupportedDocument
	}
	doc := new(Document)
	err = xmlctx.Unmarshal(data, doc, xmlctx.WithNamespaces(map[string]string{
		"":   "",
		"fe": NamespaceFacturaE,
		"ds": NamespaceDSig,
	}))
	if err != nil {
		return nil, err
	}
	return doc, nil
}

// rootNamespace provides the namespace of the data's root element.
func rootNamespace(data []byte) (string, error) {
	d := xml.NewDecoder(bytes.NewReader(data))
	for {
		tok, err := d.Token()
		if err != nil {
			return "", err
		}
		if se, ok := tok.(xml.StartElement); ok {
			return se.Name.Space, nil
		}
	}
}

// Encode provides the FacturaE XML document's bytes, including the XML
// declaration.
func Encode(doc *Document) ([]byte, error) {
	buf, err := doc.buffer(xml.Header)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// BytesIndent returns the XML document bytes with indentation for readability.
// This is useful for debugging and testing purposes, but should not be used in
// production as it will not work correctly with digital signatures.
func (d *Document) BytesIndent() ([]byte, error) {
	buf := bytes.NewBufferString(xml.Header)
	data, err := xml.MarshalIndent(d, "", "\t")
	if err != nil {
		return nil, fmt.Errorf("marshal document: %w", err)
	}
	if _, err := buf.Write(data); err != nil {
		return nil, fmt.Errorf("writing to buffer: %w", err)
	}
	return buf.Bytes(), nil
}

func (d *Document) buffer(base string) (*bytes.Buffer, error) {
	buf := bytes.NewBufferString(base)
	// data, err := xml.MarshalIndent(d, "", "  ") // not compatible with certificates
	data, err := xml.Marshal(d)
	if err != nil {
		return nil, fmt.Errorf("marshal document: %w", err)
	}
	if _, err := buf.Write(data); err != nil {
		return nil, fmt.Errorf("writing to buffer: %w", err)
	}
	return buf, nil
}

// Canonical converts a struct representation of facturae to its
// canonical representation as defined in https://www.w3.org/TR/2001/REC-xml-c14n-20010315
// (for a simpler explanation look at https://www.di-mgt.com.au/xmldsig-c14n.html)
// This is used when we need to create a hash for signing, timestamping, ...
func (d *Document) Canonical() ([]byte, error) {
	buf, err := d.buffer("")
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
