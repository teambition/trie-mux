test:
	go test

cover:
	rm -f *.coverprofile
	go test -coverprofile=trie.coverprofile
	go tool cover -html=trie.coverprofile

.PHONY: test cover
