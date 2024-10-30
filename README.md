# Sushi NFT API golang

## How to run

You'll need to have installed:

- go `>= v1.20.3`

### First step

- run `cp config.example config.yaml`

- add your environment variables to `config.yaml` and firebase configuration to `serviceAccountKey.json`

**note :**

- Please update the NFT standard ``token_type`` in ``config.yaml`` to either ``ERC1155`` or ``ERC721``. If not set, the default will be ``ERC721``.

- In this action, you'll need to configure ``nft_contract_address``, ``network`` (use for alchemy) and wait for the cron job to finish crawling NFT information.

### Setup

- `go mod download` install all dependencies

### Useful commands

- `go run main.go` - run the API instance
- `go run main.go worker` - run the `worker` instance
