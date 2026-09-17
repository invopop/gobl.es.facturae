package main

import (
	"fmt"

	facturae "github.com/invopop/gobl.es.facturae"
	"github.com/invopop/xmldsig"
	"github.com/spf13/cobra"
)

type convertOpts struct {
	*rootOpts
	cert     string
	password string
}

func convert(o *rootOpts) *convertOpts {
	return &convertOpts{rootOpts: o}
}

func (c *convertOpts) cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "convert [infile] [outfile]",
		Short: "Convert a GOBL JSON into a FacturaE XML",
		RunE:  c.runE,
	}
	f := cmd.Flags()
	f.StringVarP(&c.cert, "cert", "c", "", "Certificate for signing in pkcs12 format")
	f.StringVarP(&c.password, "password", "p", "", "Password of the certificate")

	return cmd
}

func (c *convertOpts) runE(cmd *cobra.Command, args []string) error {
	// ctx := commandContext(cmd)

	input, err := openInput(cmd, args)
	if err != nil {
		return err
	}
	defer input.Close() // nolint:errcheck

	out, err := c.openOutput(cmd, args)
	if err != nil {
		return err
	}
	defer out.Close() // nolint:errcheck

	opts := []facturae.Option{}

	// Add certificate
	if c.cert != "" {
		cert, err := xmldsig.LoadCertificate(c.cert, c.password)
		if err != nil {
			return fmt.Errorf("loading xmldisg certificate: %w", err)
		}
		opts = append(opts, facturae.WithCertificate(cert))
	}

	doc, err := facturae.LoadGOBL(input, opts...)
	if err != nil {
		return err
	}

	data, err := doc.Bytes()
	if err != nil {
		return fmt.Errorf("generating facturae xml: %w", err)
	}

	if _, err = out.Write(data); err != nil {
		return fmt.Errorf("writing facturae xml: %w", err)
	}

	return nil
}
