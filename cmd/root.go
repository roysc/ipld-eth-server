// VulcanizeDB
// Copyright © 2019 Vulcanize

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.

// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cerc-io/ipld-eth-server/v4/pkg/log"
	"github.com/cerc-io/ipld-eth-server/v4/pkg/prom"
)

var (
	cfgFile        string
	envFile        string
	subCommand     string
	logWithCommand log.Entry
)

var rootCmd = &cobra.Command{
	Use:              "ipld-eth-server",
	PersistentPreRun: initFuncs,
}

func Execute() {
	log.Info("----- Starting IPFS blockchain watcher -----")
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func initFuncs(cmd *cobra.Command, args []string) {
	log.Init()

	if viper.GetBool("metrics") {
		prom.Init()
	}

	if viper.GetBool("prom.http") {
		addr := fmt.Sprintf(
			"%s:%s",
			viper.GetString("prom.http.addr"),
			viper.GetString("prom.http.port"),
		)
		prom.Serve(addr)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file location")
	rootCmd.PersistentFlags().StringVar(&envFile, "env", "", "environment file location")

	rootCmd.PersistentFlags().String("client-ipcPath", "", "location of geth.ipc file")
	rootCmd.PersistentFlags().String("log-level", log.InfoLevel.String(), "log level (trace, debug, info, warn, error, fatal, panic)")
	rootCmd.PersistentFlags().String("log-file", "", "file path for logging")

	rootCmd.PersistentFlags().Bool("metrics", false, "enable metrics")

	rootCmd.PersistentFlags().Bool("prom-http", false, "enable http service for prometheus")
	rootCmd.PersistentFlags().String("prom-http-addr", "127.0.0.1", "http host for prometheus")
	rootCmd.PersistentFlags().String("prom-http-port", "8090", "http port for prometheus")

	viper.BindPFlag("log.level", rootCmd.PersistentFlags().Lookup("log-level"))
	viper.BindPFlag("log.file", rootCmd.PersistentFlags().Lookup("log-file"))

	viper.BindPFlag("metrics", rootCmd.PersistentFlags().Lookup("metrics"))

	viper.BindPFlag("prom.http", rootCmd.PersistentFlags().Lookup("prom-http"))
	viper.BindPFlag("prom.http.addr", rootCmd.PersistentFlags().Lookup("prom-http-addr"))
	viper.BindPFlag("prom.http.port", rootCmd.PersistentFlags().Lookup("prom-http-port"))
}

func initConfig() {
	if cfgFile == "" && envFile == "" {
		log.Fatal("No configuration file specified, use --config , --env flag to provide configuration")
	}

	if cfgFile != "" {
		if filepath.Ext(cfgFile) != ".toml" {
			log.Fatal("Provide .toml file for --config flag")
		}

		viper.SetConfigFile(cfgFile)
		if err := viper.ReadInConfig(); err != nil {
			log.Fatalf("Couldn't read config file: %s", err.Error())
		}

		log.Infof("Using config file: %s", viper.ConfigFileUsed())
	}

	if envFile != "" {
		if filepath.Ext(envFile) != ".env" {
			log.Fatal("Provide .env file for --env flag")
		}

		if err := godotenv.Load(envFile); err != nil {
			log.Fatalf("Failed to set environment variable from env file: %s", err.Error())
		}

		log.Infof("Using env file: %s", envFile)
	}
}
