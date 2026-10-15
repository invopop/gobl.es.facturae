package facturae

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConverterImport(t *testing.T) {
	_, err := converter{}.Import(KeyFacturaE, []byte("<fe:Facturae/>"))
	assert.ErrorIs(t, err, ErrImportNotSupported)
}
