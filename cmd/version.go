package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ma-tf/ogle/internal/version"
)

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long:  `Display the version, commit hash, and build date of ogle.`,
		Run: func(_ *cobra.Command, _ []string) {
			_, _ = fmt.Fprint(os.Stdout, version.ASCIIArt)
			_, _ = fmt.Fprintf(
				os.Stdout,
				"\nogle %s (commit: %s, built: %s)\n",
				version.Version,
				version.Commit,
				version.Date,
			)
		},
	}
}
