package commands

import (
	"log"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// Updater creates the Cobra Command For The Updater Action
func Updater() *cobra.Command {
	return &cobra.Command{
		Use:   "updater",
		Short: "Continually update your DNS record on an interval.",
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
			interval, err := cmd.Flags().GetInt("interval")
			if err != nil {
				return errors.Wrap(err, "error: retrieving the access-token flag")
			}

			// Start The Updater
			log.Println("Started Netlify Dynamic DNS Updater. Created By Oscar Beaumont!")
			update := Update().RunE

			for {
				err := update(cmd, args)
				if err != nil {
					return err
				}
				time.Sleep(time.Duration(interval) * time.Minute)
			}
		},
	}
}
