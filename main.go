package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func main() {
	var listen string
	var token string

	var serverCmd = &cobra.Command{
		Use:   "server",
		Short: "Run http2tcp server",
		Run: func(cmd *cobra.Command, args []string) {
			if len(token) < 4 {
				println("auth token length must be greater than 4")
				os.Exit(1)
			}

			log.Println(fmt.Sprintf("http2tcp server listen on %s", listen))
			server(listen, token)
		},
	}

	serverCmd.Flags().StringVarP(&listen, "listen", "l", "", "listen address")
	serverCmd.MarkFlagRequired("listen")
	serverCmd.Flags().StringVarP(&token, "auth", "a", "", "token")
	serverCmd.MarkFlagRequired("auth")

	var serverUrl string
	var target string

	var clientCmd = &cobra.Command{
		Use:   "client",
		Short: "Run http2tcp client",
		Run: func(cmd *cobra.Command, args []string) {
			if len(token) < 4 {
				println("auth token length must be greater than 4")
				os.Exit(1)
			}

			if listen != `-` {
				log.Println(fmt.Sprintf("http2tcp client listen on %s, target %s", listen, target))
			}
			client(listen, serverUrl, token, strings.TrimPrefix(target, "http://"))
		},
	}

	clientCmd.Flags().StringVarP(&listen, "listen", "l", "", "listen address")
	clientCmd.MarkFlagRequired("listen")
	clientCmd.Flags().StringVarP(&token, "auth", "a", "", "token")
	clientCmd.MarkFlagRequired("auth")
	clientCmd.Flags().StringVarP(&serverUrl, "server", "s", "", "server address")
	clientCmd.MarkFlagRequired("server")
	clientCmd.Flags().StringVarP(&target, "target", "t", "", "target address")
	clientCmd.MarkFlagRequired("target")

	var rootCmd = &cobra.Command{
		Use: "http2tcp",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(clientCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
