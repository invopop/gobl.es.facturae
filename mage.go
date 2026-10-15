//go:build mage
// +build mage

package main

import (
	"github.com/invopop/gobl.es.facturae/test"
)

// ConvertXML takes previously converted GOBL JSON and produces XML files.
func ConvertXML() error {
	return test.ConvertToXML()
}
