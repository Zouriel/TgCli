package cmd

import (
	"fmt"

	"tg/internal/agent"
	"tg/internal/tdlib"

	"github.com/spf13/cobra"
)

func init() {
	daemonCommand := &cobra.Command{
		Use:   "daemon",
		Short: "Run the Claude bridge: let allow-listed Telegram users drive Claude Code sessions",
		Long: "Listens on the logged-in Telegram account for messages from allow-listed\n" +
			"users and drives Claude Code on their behalf (pick a location, resume a\n" +
			"session with a summary, chat back and forth). Configure with\n" +
			"agent-locations.json and agent-allowlist.json in the tg config dir.",
		RunE: func(cmd *cobra.Command, args []string) error {
			apiID, apiHash, err := resolveTelegramCredentials()
			if err != nil {
				return err
			}

			locations, locPath, err := agent.LoadOrSeedLocations()
			if err != nil {
				return err
			}
			allow, allowPath, err := agent.LoadOrSeedAllowlist()
			if err != nil {
				return err
			}

			fmt.Println("locations:", locPath)
			fmt.Println("allowlist:", allowPath)

			tdjson, clientID, err := startTDLibClient()
			if err != nil {
				return err
			}
			defer tdjson.Close()

			if err := waitUntilReady(tdjson, clientID, apiID, apiHash); err != nil {
				return err
			}

			self, err := tdlib.FetchCurrentUser(tdjson, clientID)
			if err == nil {
				fmt.Println("Running as:", self.FirstName, self.LastName)
			}

			return agent.RunDaemon(tdjson, clientID, locations, allow)
		},
	}

	rootCmd.AddCommand(daemonCommand)
}
