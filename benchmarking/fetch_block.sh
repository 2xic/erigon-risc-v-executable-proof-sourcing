source .env
BLOCK_NUMBER_DEC=${1:-23791194}
BLOCK_NUMBER=$(printf "0x%x" $BLOCK_NUMBER_DEC)

JSON_WITNESS=$(cat <<EOF
{
  "id": 1,
  "jsonrpc": "2.0",
  "method": "debug_executionWitness",
  "params": ["$BLOCK_NUMBER"]
}
EOF
)

JSON_BLOCK=$(cat <<EOF
{
  "id": 1,
  "jsonrpc": "2.0",
  "method": "eth_getBlockByNumber",
  "params": ["$BLOCK_NUMBER", true]
}
EOF
)

curl --request POST \
     --url https://api.zan.top/node/v1/eth/mainnet/$RPC_API_KEY \
     --header 'accept: application/json' \
     --header 'content-type: application/json' \
     --data "$JSON_WITNESS" > witness-$BLOCK_NUMBER_DEC.json

jq . witness-$BLOCK_NUMBER_DEC.json | sponge witness-$BLOCK_NUMBER_DEC.json

curl --request POST \
     --url https://api.zan.top/node/v1/eth/mainnet/$RPC_API_KEY \
     --header 'accept: application/json' \
     --header 'content-type: application/json' \
     --data "$JSON_BLOCK" > block-$BLOCK_NUMBER_DEC.json

jq . block-$BLOCK_NUMBER_DEC.json | sponge block-$BLOCK_NUMBER_DEC.json

