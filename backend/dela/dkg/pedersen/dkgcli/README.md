# DKGCLI

DKGCLI is a CLI tool for using the DKG protocol. Here is a complete scenario:

```sh
# Install the CLI
go install .

# Run 3 nodes. Do that in 3 different sessions
LLVL=info dkgcli --config /tmp/node141 start --routing tree --listen tcp://127.0.0.1:3101
LLVL=info dkgcli --config /tmp/node142 start --routing tree --listen tcp://127.0.0.1:3102
LLVL=info dkgcli --config /tmp/node143 start --routing tree --listen tcp://127.0.0.1:3103

# Exchange certificates
dkgcli --config /tmp/node142 minogrpc join --address //127.0.0.1:3101 $(dkgcli --config /tmp/node141 minogrpc token)
dkgcli --config /tmp/node143 minogrpc join --address //127.0.0.1:3101 $(dkgcli --config /tmp/node141 minogrpc token)

# Initialize DKG on each node. Do that in a 4th session.
dkgcli --config /tmp/node141 dkg listen
dkgcli --config /tmp/node142 dkg listen
dkgcli --config /tmp/node143 dkg listen

# Do the setup in one of the node:
dkgcli --config /tmp/node141 dkg setup --authority $(cat /tmp/node141/dkgauthority) --authority $(cat /tmp/node142/dkgauthority) --authority $(cat /tmp/node143/dkgauthority)
# dkgcli --config /tmp/node141 dkg setup \
#     --authority $(cat /tmp/node141/dkgauthority) \
#     --authority $(cat /tmp/node142/dkgauthority) \
#     --authority $(cat /tmp/node143/dkgauthority)

# Encrypt a message:
dkgcli --config /tmp/node142 dkg encrypt --message deadbeef

# Decrypt a message
dkgcli --config /tmp/node143 dkg decrypt --encrypted <...>

# Issue Master Credential
dkgcli --config /tmp/node142 dkg issueMasterCredential --idhash hashofidentification

# Issue Event Credential
dkgcli --config /tmp/node142 dkg issueEventCredential --masterCredential masterCredential 