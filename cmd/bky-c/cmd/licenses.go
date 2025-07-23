package cmd

import (
	"embed"
	"fmt"
	"io/fs"

	"github.com/spf13/cobra"
)

//go:embed licenses/*
var licenses embed.FS

var licensesCmd = &cobra.Command{
	Use:   "licenses",
	Short: "Print embedded third-party license info",
	RunE: func(cmd *cobra.Command, args []string) error {
		return fs.WalkDir(licenses, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			data, err := licenses.ReadFile(path)
			if err != nil {
				return err
			}

			path = path[len("licenses/"):]
			fmt.Printf("----- %s -----\n%s\n\n", path, string(data))
			return nil
		})
	},
}

func init() {
	rootCmd.AddCommand(licensesCmd)
}
