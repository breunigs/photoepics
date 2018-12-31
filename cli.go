package main

import (
	"github.com/spf13/cobra"
)

func cmdGen() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gen",
		Short: "Generate photo sequence for the given input file",
		Run: func(cmd *cobra.Command, args []string) {
			GenPathAlong(inputFilePath, startImageKey, endImageKey)
		},
	}
	cmd.Flags().StringVarP(&inputFilePath, "input", "i", "", "input file for which to generate a photo sequence")
	cmd.MarkFlagRequired("input")
	requireApiKey(cmd)
	filterByUserName(cmd)
	filterByDate(cmd)

	cmd.Flags().StringVar(&startImageKey, "start-image", "", "The image to start from")
	cmd.MarkFlagRequired("start-image")
	cmd.Flags().StringVar(&endImageKey, "end-image", "", "The image to stop at")
	cmd.MarkFlagRequired("end-image")

	return cmd
}

func requireApiKey(cmd *cobra.Command) {
	cmd.Flags().StringVar(&mapConf.ApiKey, "api-key", "", "Mapillary API Key")
	cmd.MarkFlagRequired("api-key")
}

func filterByUserName(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&mapConf.FilterUsers, "filter-users", "", "", "only use photos from these Mapillary users. Comma separated.")
}

func filterByDate(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&mapConf.FilterNewer, "filter-newer", "", "", "only use sequences newer than this date. Format YYYY-MM-DD.")
}
