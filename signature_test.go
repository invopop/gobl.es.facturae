package facturae_test

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	_ "crypto/sha1"   // hashFor
	_ "crypto/sha256" // hashFor
	_ "crypto/sha512" // hashFor
	"crypto/x509"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/beevik/etree"
	facturae "github.com/invopop/gobl.es.facturae"
	"github.com/invopop/gobl.es.facturae/test"
	"github.com/invopop/xmldsig"
	dsig "github.com/russellhaering/goxmldsig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignature(t *testing.T) {
	examples, err := lookupExamples()
	require.NoError(t, err)
	cert, err := test.LoadCertificate()
	require.NoError(t, err)
	opts := []facturae.Option{
		facturae.WithThirdParty(test.ThirdParty()),
		facturae.WithCertificate(cert),
		facturae.WithSigning(
			xmldsig.WithCurrentTime(func() time.Time { return signingTime }),
		),
	}

	for _, example := range examples {
		t.Run(example, func(t *testing.T) {
			doc, err := test.NewDocumentFrom(example, opts...)
			require.NoError(t, err)
			data, err := facturae.Encode(doc)
			require.NoError(t, err)

			require.NoError(t, verifySignature(data, cert))

			t.Run("decodes the signature", func(t *testing.T) {
				out, err := facturae.Decode(data)
				require.NoError(t, err)
				want, err := xml.Marshal(doc.Signature)
				require.NoError(t, err)
				got, err := xml.Marshal(out.Signature)
				require.NoError(t, err)
				assert.Equal(t, string(want), string(got))
			})
		})
	}

	doc, err := test.NewDocumentFrom("invoice-vat.json", opts...)
	require.NoError(t, err)
	data, err := facturae.Encode(doc)
	require.NoError(t, err)

	t.Run("refuses a changed document", func(t *testing.T) {
		out := bytes.Replace(data, []byte("<InvoiceNumber>"), []byte("<InvoiceNumber>X"), 1)
		err := verifySignature(out, cert)
		assert.ErrorContains(t, err, "digest of reference \"\"")
	})
	t.Run("refuses changed signed properties", func(t *testing.T) {
		out := bytes.Replace(data, []byte("Factura Electrónica"), []byte("Factura"), 1)
		err := verifySignature(out, cert)
		assert.ErrorContains(t, err, "SignedProperties")
	})
	t.Run("refuses a changed signed info", func(t *testing.T) {
		out := bytes.Replace(data, []byte(`URI=""`), []byte(`URI="" Id="X"`), 1)
		err := verifySignature(out, cert)
		assert.ErrorContains(t, err, "signature value")
	})
	t.Run("refuses an indented document", func(t *testing.T) {
		out, err := doc.BytesIndent()
		require.NoError(t, err)
		assert.Error(t, verifySignature(out, cert))
	})
	t.Run("refuses another certificate", func(t *testing.T) {
		other, err := xmldsig.LoadCertificate(test.GetCertificatesPath()+"invopop.p12", "invopop")
		require.NoError(t, err)
		assert.ErrorContains(t, verifySignature(data, other), "certificate")
	})
}

// verifySignature checks a signed FacturaE document as a receiver would,
// from its bytes alone: every reference digest, the XAdES signing
// certificate, and the signature value, using the certificate in the
// document, which must be the one expected.
func verifySignature(data []byte, expected *xmldsig.Certificate) error {
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(data); err != nil {
		return err
	}
	sig := doc.Root().SelectElement("ds:Signature")
	if sig == nil {
		return errors.New("no signature")
	}
	si := sig.SelectElement("ds:SignedInfo")

	// Certificate
	certData, err := base64.StdEncoding.DecodeString(
		strings.TrimSpace(sig.FindElement("ds:KeyInfo/ds:X509Data/ds:X509Certificate").Text()),
	)
	if err != nil {
		return err
	}
	cert, err := x509.ParseCertificate(certData)
	if err != nil {
		return err
	}
	if base64.StdEncoding.EncodeToString(cert.Raw) != expected.NakedPEM() {
		return errors.New("certificate is not the expected one")
	}
	cd := sig.FindElement(".//xades:SigningCertificate/xades:Cert/xades:CertDigest")
	if cd == nil {
		return errors.New("no XAdES signing certificate")
	}
	if err := checkDigest(cd, cert.Raw); err != nil {
		return fmt.Errorf("signing certificate: %w", err)
	}

	// References
	for _, ref := range si.SelectElements("ds:Reference") {
		uri := ref.SelectAttrValue("URI", "")
		var target *etree.Element
		if uri == "" {
			// The whole document, without the enveloped signature.
			c := doc.Copy()
			c.Root().RemoveChild(c.Root().SelectElement("ds:Signature"))
			target = c.Root()
		} else {
			target = findByID(doc.Root(), strings.TrimPrefix(uri, "#"))
			if target == nil {
				return fmt.Errorf("reference %q not found", uri)
			}
		}
		c14n, err := canonicalize(target, dsig.MakeC14N10RecCanonicalizer())
		if err != nil {
			return err
		}
		if err := checkDigest(ref, c14n); err != nil {
			return fmt.Errorf("digest of reference %q (%s): %w", uri, target.Tag, err)
		}
	}

	// Signature value
	if alg := si.SelectElement("ds:CanonicalizationMethod").SelectAttrValue("Algorithm", ""); alg != string(dsig.CanonicalXML10RecAlgorithmId) {
		return fmt.Errorf("unexpected canonicalization %s", alg)
	}
	c14n, err := canonicalize(si, dsig.MakeC14N10RecCanonicalizer())
	if err != nil {
		return err
	}
	h, err := hashFor(si.SelectElement("ds:SignatureMethod").SelectAttrValue("Algorithm", ""))
	if err != nil {
		return err
	}
	value, err := base64.StdEncoding.DecodeString(strings.TrimSpace(sig.SelectElement("ds:SignatureValue").Text()))
	if err != nil {
		return err
	}
	hh := h.New()
	hh.Write(c14n)
	if err := rsa.VerifyPKCS1v15(cert.PublicKey.(*rsa.PublicKey), h, hh.Sum(nil), value); err != nil {
		return fmt.Errorf("signature value: %w", err)
	}
	return nil
}

// canonicalize provides the inclusive canonical form of an element, which
// includes the namespaces declared by its ancestors.
func canonicalize(el *etree.Element, c dsig.Canonicalizer) ([]byte, error) {
	c14n := el.Copy()
	declared := make(map[string]bool)
	for _, a := range c14n.Attr {
		if a.Space == "xmlns" || (a.Space == "" && a.Key == "xmlns") {
			declared[a.Key] = true
		}
	}
	for p := el.Parent(); p != nil; p = p.Parent() {
		for _, a := range p.Attr {
			if a.Space == "xmlns" && !declared[a.Key] {
				declared[a.Key] = true
				c14n.CreateAttr("xmlns:"+a.Key, a.Value)
			}
		}
	}
	return c.Canonicalize(c14n)
}

func findByID(el *etree.Element, id string) *etree.Element {
	if el.SelectAttrValue("Id", "") == id {
		return el
	}
	for _, c := range el.ChildElements() {
		if f := findByID(c, id); f != nil {
			return f
		}
	}
	return nil
}

// checkDigest compares the digest in an element with ds:DigestMethod and
// ds:DigestValue children against the digest of the data.
func checkDigest(el *etree.Element, data []byte) error {
	h, err := hashFor(el.FindElement("ds:DigestMethod").SelectAttrValue("Algorithm", ""))
	if err != nil {
		return err
	}
	hh := h.New()
	hh.Write(data)
	got := base64.StdEncoding.EncodeToString(hh.Sum(nil))
	if want := strings.TrimSpace(el.FindElement("ds:DigestValue").Text()); got != want {
		return fmt.Errorf("digest %s does not match %s", got, want)
	}
	return nil
}

func hashFor(alg string) (crypto.Hash, error) {
	switch alg {
	case "http://www.w3.org/2000/09/xmldsig#sha1", "http://www.w3.org/2000/09/xmldsig#rsa-sha1":
		return crypto.SHA1, nil
	case "http://www.w3.org/2001/04/xmlenc#sha256", "http://www.w3.org/2001/04/xmldsig-more#rsa-sha256":
		return crypto.SHA256, nil
	case "http://www.w3.org/2001/04/xmlenc#sha512", "http://www.w3.org/2001/04/xmldsig-more#rsa-sha512":
		return crypto.SHA512, nil
	}
	return 0, fmt.Errorf("unsupported algorithm %s", alg)
}
