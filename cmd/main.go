package main

import (
	"os"

	"github.com/oscartbeaumont/netlify-dynamic-dns/cmd/commands"
	"github.com/spf13/cobra"
)

var GitCommit string

func main() {
	// Create The Command Line
	cmd := &cobra.Command{
		Use:          "netlify-ddns",
		Short:        "A Dynamic DNS Client For Netlify Managed DNS. Created By Oscar Beaumont!",
		SilenceUsage: true,
	}

	// Commands
	cmd.AddCommand(commands.Version(GitCommit))
	cmd.AddCommand(commands.Update())
	cmd.AddCommand(commands.Updater())

	// Global Flags
	cmd.PersistentFlags().String("access-token", "", "a personal access tokens for your Netlify accounts. Can be created in 'User Settings > Applications' on the dashboard.")
	cmd.PersistentFlags().String("domain", "example.com", "the full domain for the DNS record.")
	cmd.PersistentFlags().String("subdomain", "home", "the subdomain segment for the DNS record.")
	cmd.PersistentFlags().Bool("ipv6", true, "whether the IPv6 'AAAA' DNS record should be updated.")
	cmd.PersistentFlags().Int("interval", 5, "the interval (in minutes) to update your dns record in the updater mode.")

	// Run The Command Line
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
