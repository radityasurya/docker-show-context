package commands

import (
	"fmt"
	"os"

	"github.com/radityasurya/docker-show-context/pkg/client"

	"github.com/spf13/viper"

	"github.com/spf13/cobra"
)

// rootCmd of the docker show context
var rootCmd = &cobra.Command{
	Use:   "docker-show-context",
	Short: "Showing your docker context",
	Run: func(cmd *cobra.Command, args []string) {
		client.Run()
	},
}

// Execute the command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}

func init() {
	rootCmd.PersistentFlags().StringP("dockerfile", "d", "Dockerfile", "The dockerfile you want to set")
	rootCmd.PersistentFlags().IntP("files-number", "n", 10, "The number of the files do you want to see")
	viper.BindPFlags(rootCmd.PersistentFlags())
}
