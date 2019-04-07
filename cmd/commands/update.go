package commands

import (
	"log"

	"github.com/oscartbeaumont/netlify-dynamic-dns/pkg"
	"github.com/pkg/errors"

	"github.com/spf13/cobra"
)

// Update creates the Cobra Command For The Update Action
func Update() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update your DNS record and exit.",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if accessToken, err := cmd.Flags().GetString("access-token"); err != nil {
				return errors.Wrap(err, "Error: Accessing the access-token.")
			} else if accessToken == "" {
				return errors.New("Error: Please set an access-token")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get The Command Line Flags
			accessToken, err := cmd.Flags().GetString("access-token")
			if err != nil {
				return errors.Wrap(err, "error: retrieving the access-token flag")
			}

			domain, err := cmd.Flags().GetString("domain")
			if err != nil {
				return errors.Wrap(err, "error: retrieving the domain flag")
			}

			subdomain, err := cmd.Flags().GetString("subdomain")
			if err != nil {
				return errors.Wrap(err, "error: retrieving the subdomain flag")
			}

			// Get Any Existing DNS Records For The Target Domain
			records, err := pkg.GetRecords(domain, accessToken)
			if err != nil {
				return err
			}

			// Update The A Record
			log.Println("Updating IPv4 Record...")
			ipv4, err := pkg.GetPublicIPv4()
			if err != nil {
				return errors.Wrap(err, "error: retrieving your public ipv4 address")
			}

			record := pkg.DNSRecord{
				Type:  "A",
				Name:  subdomain,
				Value: ipv4,
			}

			err = pkg.UpdateRecord(domain, accessToken, record, records)
			if err != nil {
				return err
			}

			log.Println("Updated IPv4 Record To: " + ipv4)

			// Update The AAAA Record
			if ipv6, err := cmd.Flags().GetBool("ipv6"); err == nil && ipv6 {
				log.Println("Updating IPv6 Record...")

				ipv6, err := pkg.GetPublicIPv6()
				if err != nil {
					return errors.Wrap(err, "error: retrieving your public ipv6 address")
				}

				record := pkg.DNSRecord{
					Type:  "AAAA",
					Name:  subdomain,
					Value: ipv6,
				}

				err = pkg.UpdateRecord(domain, accessToken, record, records)
				if err != nil {
					return err
				}

				log.Println("Updated IPv6 Record To: " + ipv6)
			} else if err != nil {
				return errors.Wrap(err, "error: retrieving the ipv6 flag")
			}

			return nil
		},
	}

	return cmd
}
