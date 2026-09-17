package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/invopop/gobl"
	facturae "github.com/invopop/gobl.es.facturae"
	"github.com/invopop/xmldsig"
)

const certificateFile = "facturae.p12"
const certificatePassword = "invopop"

// Random third party example data.
var thirdParty = &facturae.ThirdParty{
	TaxIdentification: &facturae.TaxID{
		PersonTypeCode:          "J",
		ResidenceTypeCode:       "R",
		TaxIdentificationNumber: "B23103039",
	},
	LegalEntity: &facturae.LegalEntity{
		CorporateName: "Hypeprop S.L.",
		AddressInSpain: &facturae.Address{
			Address:     "Calle Campo Real 74",
			PostCode:    "28023",
			Town:        "Torrejón De La Calzada",
			Province:    "Madrid",
			CountryCode: "ESP",
		},
	},
}

// NewDocumentFrom creates a FacturaE Document from a GOBL file in the `test/data` folder
func NewDocumentFrom(name string, opts ...facturae.Option) (*facturae.Document, error) {
	env, err := LoadTestEnvelope(name)
	if err != nil {
		return nil, err
	}
	return facturae.NewInvoice(env, opts...)
}

// LoadTestEnvelope returns a GOBL Envelope from a file in the `test/data` folder
func LoadTestEnvelope(name string) (*gobl.Envelope, error) {
	src, _ := os.Open(filepath.Join(GetDataPath(), name))

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(src); err != nil {
		return nil, err
	}

	env := new(gobl.Envelope)
	if err := json.Unmarshal(buf.Bytes(), env); err != nil {
		return nil, err
	}

	return env, nil
}

// LoadGOBL loads a GoBL test file into structs
func LoadGOBL(name string, opts ...facturae.Option) (*facturae.Document, error) {
	envelopeReader, _ := os.Open(filepath.Join(GetDataPath(), name))
	doc, err := facturae.LoadGOBL(envelopeReader, opts...)
	if err != nil {
		return nil, err
	}
	return doc, nil
}

// ConvertToXML takes the .json invoices generated previously and converts them
// into XML FacturaE documents.
func ConvertToXML() error {
	var files []string
	err := filepath.Walk(GetDataPath(), func(path string, info os.FileInfo, err error) error {
		if filepath.Ext(path) == ".json" {
			files = append(files, filepath.Base(path))
		}
		return nil
	})
	if err != nil {
		return err
	}

	cert, err := LoadCertificate()
	if err != nil {
		return err
	}

	for _, file := range files {
		fmt.Printf("processing file: %v\n", file)

		doc, err := LoadGOBL(file,
			facturae.WithCertificate(cert),
			facturae.WithThirdParty(thirdParty),
		)
		if err != nil {
			return err
		}

		data, err := doc.Bytes()
		if err != nil {
			return fmt.Errorf("extracting document bytes: %w", err)
		}

		np := strings.TrimSuffix(file, filepath.Ext(file)) + ".xml"
		err = os.WriteFile(GetDataPath()+"/"+np, data, 0644)
		if err != nil {
			return fmt.Errorf("writing file: %w", err)
		}
	}

	return nil
}

// GetTestPath returns the path to the `test` folder
func GetTestPath() string {
	return filepath.Join(getRootFolder(), "test")
}

// GetDataPath returns the path to the `test/data` folder
func GetDataPath() string {
	return filepath.Join(GetTestPath(), "data")
}

func getRootFolder() string {
	cwd, _ := os.Getwd()

	for !isRootFolder(cwd) {
		cwd = removeLastEntry(cwd)
	}

	return cwd
}

func isRootFolder(dir string) bool {
	files, _ := os.ReadDir(dir)

	for _, file := range files {
		if file.Name() == "go.mod" {
			return true
		}
	}

	return false
}

func removeLastEntry(dir string) string {
	lastEntry := "/" + filepath.Base(dir)
	i := strings.LastIndex(dir, lastEntry)
	return dir[:i]
}

// GetCertificatesPath return the path where a test can find crypto
// certificates that can be used in tests
func GetCertificatesPath() string {
	return getRootFolder() + "/test/certificates/"
}

// LoadCertificate will return the standard test certificate info
func LoadCertificate() (*xmldsig.Certificate, error) {
	f := path.Join(GetCertificatesPath(), certificateFile)
	return xmldsig.LoadCertificate(f, certificatePassword)
}

// ThirdParty returns a random third party for testing purposes
func ThirdParty() *facturae.ThirdParty {
	return thirdParty
}
