package cli

import (
	"fmt"
	"strconv"
	"time"

	"github.com/juanfont/headscale"
	v1 "github.com/juanfont/headscale/gen/go/headscale/v1"
	"github.com/pterm/pterm"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	// 90 days.
	DefaultAPIKeyExpiry = 90 * 24 * time.Hour
)

func init() {
	rootCmd.AddCommand(apiKeysCmd)
	apiKeysCmd.AddCommand(listAPIKeys)

	createAPIKeyCmd.Flags().
		DurationP("expiration", "e", DefaultAPIKeyExpiry, "Human-readable expiration of the key (e.g. 30m, 24h)")

	apiKeysCmd.AddCommand(createAPIKeyCmd)

	expireAPIKeyCmd.Flags().StringP("prefix", "p", "", "ApiKey prefix")
	err := expireAPIKeyCmd.MarkFlagRequired("prefix")
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}
	apiKeysCmd.AddCommand(expireAPIKeyCmd)
}

var apiKeysCmd = &cobra.Command{
	Use:     "apikeys",
	Short:   "Handle the Api keys in Headscale",
	Aliases: []string{"apikey", "api"},
}

var listAPIKeys = &cobra.Command{
	Use:     "list",
	Short:   "List the Api keys for headscale",
	Aliases: []string{"ls", "show"},
	Run: func(cmd *cobra.Command, args []string) {
		output, _ := cmd.Flags().GetString("output")

		ctx, client, conn, cancel := getHeadscaleCLIClient()
		defer cancel()
		defer conn.Close()

		request := &v1.ListApiKeysRequest{}

		response, err := client.ListApiKeys(ctx, request)
		if err != nil {
			ErrorOutput(
				err,
				fmt.Sprintf("Error getting the list of keys: %s", err),
				output,
			)

			return
		}

		if output != "" {
			SuccessOutput(response.ApiKeys, "", output)

			return
		}

		tableData := pterm.TableData{
			{"ID", "Prefix", "Expiration", "Created"},
		}
		for _, key := range response.ApiKeys {
			expiration := "-"

			if key.GetExpiration() != nil {
				expiration = ColourTime(key.Expiration.AsTime())
			}

			tableData = append(tableData, []string{
				strconv.FormatUint(key.GetId(), headscale.Base10),
				key.GetPrefix(),
				expiration,
				key.GetCreatedAt().AsTime().Format(HeadscaleDateTimeFormat),
			})

		}
		err = pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
		if err != nil {
			ErrorOutput(
				err,
				fmt.Sprintf("Failed to render pterm table: %s", err),
				output,
			)

			return
		}
	},
}

var createAPIKeyCmd = &cobra.Command{
	Use:   "create",
	Short: "Creates a new Api key",
	Long: `
Creates a new Api key, the Api key is only visible on creation
and cannot be retrieved again.
If you loose a key, create a new one and revoke (expire) the old one.`,
	Aliases: []string{"c", "new"},
	Run: func(cmd *cobra.Command, args []string) {
		output, _ := cmd.Flags().GetString("output")

		log.Trace().
			Msg("Preparing to create ApiKey")

		request := &v1.CreateApiKeyRequest{}

		duration, _ := cmd.Flags().GetDuration("expiration")
		expiration := time.Now().UTC().Add(duration)

		log.Trace().Dur("expiration", duration).Msg("expiration has been set")

		request.Expiration = timestamppb.New(expiration)

		ctx, client, conn, cancel := getHeadscaleCLIClient()
		defer cancel()
		defer conn.Close()

		response, err := client.CreateApiKey(ctx, request)
		if err != nil {
			ErrorOutput(
				err,
				fmt.Sprintf("Cannot create Api Key: %s\n", err),
				output,
			)

			return
		}

		SuccessOutput(response.ApiKey, response.ApiKey, output)
	},
}

var expireAPIKeyCmd = &cobra.Command{
	Use:     "expire",
	Short:   "Expire an ApiKey",
	Aliases: []string{"revoke", "exp", "e"},
	Run: func(cmd *cobra.Command, args []string) {
		output, _ := cmd.Flags().GetString("output")

		prefix, err := cmd.Flags().GetString("prefix")
		if err != nil {
			ErrorOutput(
				err,
				fmt.Sprintf("Error getting prefix from CLI flag: %s", err),
				output,
			)

			return
		}

		ctx, client, conn, cancel := getHeadscaleCLIClient()
		defer cancel()
		defer conn.Close()

		request := &v1.ExpireApiKeyRequest{
			Prefix: prefix,
		}

		response, err := client.ExpireApiKey(ctx, request)
		if err != nil {
			ErrorOutput(
				err,
				fmt.Sprintf("Cannot expire Api Key: %s\n", err),
				output,
			)

			return
		}

		SuccessOutput(response, "Key expired", output)
	},
}
