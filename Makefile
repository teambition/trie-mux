test:
	go test --race
	go test --race ./mux

bench:
	go test -bench=. ./mux

cover:
	rm -f *.coverprofile
	go test -coverprofile=trie.coverprofile
	go test -coverprofile=mux.coverprofile ./mux
	gover
	go tool cover -html=gover.coverprofile
	rm -f *.coverprofile

.PHONY: test cover
