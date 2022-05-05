set -e
set -o xtrace

export ETH_FORWARD_ETH_CALLS=false
export DB_WRITE=true
export ETH_PROXY_ON_ERROR=false
export WATCHED_ADDRESS_GAP_FILLER_ENABLED=true
export WATCHED_ADDRESS_GAP_FILLER_INTERVAL=5

export PGPASSWORD=password
export DATABASE_USER=vdbm
export DATABASE_PORT=8077
export DATABASE_PASSWORD=password
export DATABASE_HOSTNAME=127.0.0.1

# Wait for containers to be up and execute the integration test.
while [ "$(curl -s -o /dev/null -w ''%{http_code}'' localhost:8081)" != "200" ]; do echo "waiting for ipld-eth-server..." && sleep 5; done && \
          while [ "$(curl -s -o /dev/null -w ''%{http_code}'' localhost:8545)" != "200" ]; do echo "waiting for geth-statediff..." && sleep 5; done && \
          make integrationtest
