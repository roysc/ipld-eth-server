// Copyright © 2021 Vulcanize, Inc
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"time"

	validator "github.com/cerc-io/eth-ipfs-state-validator/v4/pkg"
	ipfsethdb "github.com/cerc-io/ipfs-ethdb/v4/postgres"
	"github.com/cerc-io/ipld-eth-server/v4/pkg/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	s "github.com/cerc-io/ipld-eth-server/v4/pkg/serve"
)

const GroupName = "statedb-validate"
const CacheExpiryInMins = 8 * 60 // 8 hours
const CacheSizeInMB = 16         // 16 MB

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "valdiate state",
	Long:  `This command validates the trie for the given state root`,
	Run: func(cmd *cobra.Command, args []string) {
		subCommand = cmd.CalledAs()
		logWithCommand = *log.WithField("SubCommand", subCommand)
		validate()
	},
}

func validate() {
	config, err := s.NewConfig()
	if err != nil {
		logWithCommand.Fatal(err)
	}

	stateRootStr := viper.GetString("stateRoot")
	if stateRootStr == "" {
		logWithCommand.Fatal("must provide a state root for state validation")
	}

	stateRoot := common.HexToHash(stateRootStr)
	cacheSize := viper.GetInt("cacheSize")

	ethDB := ipfsethdb.NewDatabase(config.DB, ipfsethdb.CacheConfig{
		Name:           GroupName,
		Size:           cacheSize * 1024 * 1024,
		ExpiryDuration: time.Minute * time.Duration(CacheExpiryInMins),
	})

	val := validator.NewValidator(nil, ethDB)
	if err = val.ValidateTrie(stateRoot); err != nil {
		log.Fatal("Error validating state root")
	}

	stats := ethDB.(*ipfsethdb.Database).GetCacheStats()
	log.Debugf("groupcache stats %+v", stats)

	log.Info("Successfully validated state root")
}

func init() {
	rootCmd.AddCommand(validateCmd)

	addDatabaseFlags(validateCmd)

	validateCmd.PersistentFlags().String("state-root", "", "root of the state trie we wish to validate")
	viper.BindPFlag("stateRoot", validateCmd.PersistentFlags().Lookup("state-root"))

	validateCmd.PersistentFlags().Int("cache-size", CacheSizeInMB, "cache size in MB")
	viper.BindPFlag("cacheSize", validateCmd.PersistentFlags().Lookup("cache-size"))
}
