# Sushi NFT API golang

## How to run

You'll need to have installed:

- go `>= v1.20.3`

### first step

- run `cp config.example config.yaml`

- add your environment variables to `config.yaml` and firebase configuration to `serviceAccountKey.json`

### setup

- `go mod download` install all dependencies

### useful commands

- `go run main.go` - run the API instance
- `go run main.go worker` - run the `worker` instance
