#!/usr/bin/env bash

# This script is creating a new chain and setting up the services needed to run
# an evoting system. It ends by starting the http server needed by the frontend
# to communicate with the blockchain. This operation is blocking. It is expected
# that the "memcoin" binary is at the root. You can build it with:
#   go build ./cli/memcoin

set -e

GREEN='\033[0;32m'
NC='\033[0m' # No Color

# Setup initial nodes/ports
PORT1=2041
NODE1=20

NODE2=`expr $NODE1 + 1`
NODE3=`expr $NODE1 + 2`
PORT2=`expr $PORT1 + 1`
PORT3=`expr $PORT1 + 2`

echo "${GREEN}[0/7]${NC} create nodes"
LLVL=info memcoin --config /tmp/node${NODE1} start --listen tcp://127.0.0.1:${PORT1}
LLVL=info memcoin --config /tmp/node${NODE2} start --listen tcp://127.0.0.1:${PORT2}
LLVL=info memcoin --config /tmp/node${NODE3} start --listen tcp://127.0.0.1:${PORT3}

echo "${GREEN}[1/7]${NC} connect nodes"
memcoin --config /tmp/node${NODE2} minogrpc join \
    --address //localhost:${PORT1} $(memcoin --config /tmp/node${NODE1} minogrpc token)
memcoin --config /tmp/node${NODE3} minogrpc join \
    --address //localhost:${PORT1} $(memcoin --config /tmp/node${NODE1} minogrpc token)

echo "${GREEN}[2/7]${NC} create a chain"
memcoin --config /tmp/node${NODE1} ordering setup\
    --member $(memcoin --config /tmp/node${NODE1} ordering export)\
    --member $(memcoin --config /tmp/node${NODE2} ordering export)\
    --member $(memcoin --config /tmp/node${NODE3} ordering export)

echo "${GREEN}[3/7]${NC} setup access rights on each node"
memcoin --config /tmp/node${NODE1} access add \
    --identity $(crypto bls signer read --path private.key --format BASE64_PUBKEY)
memcoin --config /tmp/node${NODE2} access add \
    --identity $(crypto bls signer read --path private.key --format BASE64_PUBKEY)
memcoin --config /tmp/node${NODE3} access add \
    --identity $(crypto bls signer read --path private.key --format BASE64_PUBKEY)

echo "${GREEN}[4/7]${NC} grant access on the chain"
memcoin --config /tmp/node${NODE1} pool add\
    --key private.key\
    --args go.dedis.ch/dela.ContractArg --args go.dedis.ch/dela.Access\
    --args access:grant_id --args 0300000000000000000000000000000000000000000000000000000000000000\
    --args access:grant_contract --args go.dedis.ch/dela.Evoting\
    --args access:grant_command --args all\
    --args access:identity --args $(crypto bls signer read --path private.key --format BASE64_PUBKEY)\
    --args access:command --args GRANT

memcoin --config /tmp/node${NODE1} pool add\
    --key private.key\
    --args go.dedis.ch/dela.ContractArg --args go.dedis.ch/dela.Access\
    --args access:grant_id --args 0300000000000000000000000000000000000000000000000000000000000000\
    --args access:grant_contract --args go.dedis.ch/dela.Evoting\
    --args access:grant_command --args all\
    --args access:identity --args $(crypto bls signer read --path /tmp/node1/private.key --format BASE64_PUBKEY)\
    --args access:command --args GRANT

memcoin --config /tmp/node${NODE1} pool add\
    --key private.key\
    --args go.dedis.ch/dela.ContractArg --args go.dedis.ch/dela.Access\
    --args access:grant_id --args 0300000000000000000000000000000000000000000000000000000000000000\
    --args access:grant_contract --args go.dedis.ch/dela.Evoting\
    --args access:grant_command --args all\
    --args access:identity --args $(crypto bls signer read --path /tmp/node${NODE2}/private.key --format BASE64_PUBKEY)\
    --args access:command --args GRANT    

memcoin --config /tmp/node${NODE2} pool add\
    --key private.key\
    --args go.dedis.ch/dela.ContractArg --args go.dedis.ch/dela.Access\
    --args access:grant_id --args 0300000000000000000000000000000000000000000000000000000000000000\
    --args access:grant_contract --args go.dedis.ch/dela.Evoting\
    --args access:grant_command --args all\
    --args access:identity --args $(crypto bls signer read --path /tmp/node${NODE3}/private.key --format BASE64_PUBKEY)\
    --args access:command --args GRANT

# The following is not needed anymore thanks to the "postinstall" functionality.
# See #65.

# echo "${GREEN}[5/7]${NC} init shuffle"
# ./memcoin --config /tmp/node1 shuffle init --signer /tmp/node1/private.key
# ./memcoin --config /tmp/node2 shuffle init --signer /tmp/node2/private.key
# ./memcoin --config /tmp/node3 shuffle init --signer /tmp/node3/private.key

# echo "${GREEN}[6/7]${NC} starting http proxy"
# ./memcoin --config /tmp/node1 proxy start --clientaddr 127.0.0.1:8081
# ./memcoin --config /tmp/node1 e-voting registerHandlers --signer private.key
# ./memcoin --config /tmp/node1 dkg registerHandlers

# ./memcoin --config /tmp/node2 proxy start --clientaddr 127.0.0.1:8082
# ./memcoin --config /tmp/node2 e-voting registerHandlers --signer private.key
# ./memcoin --config /tmp/node2 dkg registerHandlers

# ./memcoin --config /tmp/node3 proxy start --clientaddr 127.0.0.1:8083
# ./memcoin --config /tmp/node3 e-voting registerHandlers --signer private.key
# ./memcoin --config /tmp/node3 dkg registerHandlers

# If a form is created with ID "deadbeef" then one must set up DKG
# on each node before the form can proceed:
# ./memcoin --config /tmp/node1 dkg init --formID deadbeef
# ./memcoin --config /tmp/node2 dkg init --formID deadbeef
# ./memcoin --config /tmp/node3 dkg init --formID deadbeef
# ./memcoin --config /tmp/node1 dkg setup --formID deadbeef
