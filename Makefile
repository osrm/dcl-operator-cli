build: cmd/witnesschain-cli/main.go 
	go build -o witnesschain-cli -ldflags "-X main.VERSION="$$(git describe --tags --abbrev=0) cmd/witnesschain-cli/main.go 

test: witnesschain-cli
	./witnesschain-cli registerProver --config-file templates/prover.json
	./witnesschain-cli deRegisterProver --config-file templates/prover.json
	./witnesschain-cli registerChallenger --config-file templates/challenger.json 
	./witnesschain-cli deRegisterChallenger --config-file templates/challenger.json

install: witnesschain-cli
	cp witnesschain-cli ${HOME}/.local/bin