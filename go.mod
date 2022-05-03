module github.com/vulcanize/ipld-eth-server

go 1.15

require (
	github.com/ethereum/go-ethereum v1.10.17
	github.com/graph-gophers/graphql-go v1.3.0
	github.com/ipfs/go-block-format v0.0.3
	github.com/ipfs/go-cid v0.0.7
	github.com/ipfs/go-ipfs-blockstore v1.0.1
	github.com/ipfs/go-ipfs-ds-help v1.0.0
	github.com/ipfs/go-ipld-format v0.2.0
	github.com/jmoiron/sqlx v1.3.5
	github.com/joho/godotenv v1.4.0
	github.com/lib/pq v1.10.5
	github.com/machinebox/graphql v0.2.2
	github.com/mailgun/groupcache/v2 v2.3.0
	github.com/matryer/is v1.4.0 // indirect
	github.com/multiformats/go-multihash v0.1.0
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.19.0
	github.com/prometheus/client_golang v1.11.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.4.0
	github.com/spf13/viper v1.11.0
	github.com/vulcanize/eth-ipfs-state-validator/v3 v3.0.0
	github.com/vulcanize/gap-filler v0.3.1
	github.com/vulcanize/ipfs-ethdb/v3 v3.0.1
)

replace github.com/ethereum/go-ethereum v1.10.17 => github.com/vulcanize/go-ethereum v1.10.17-statediff-3.2.0
