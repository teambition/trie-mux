test:
	go test
	go test ./mux

cover:
	rm -f *.coverprofile
	go test -coverprofile=trie.coverprofile
	go test -coverprofile=mux.coverprofile ./mux
	gover
	go tool cover -html=gover.coverprofile

.PHONY: test cover
