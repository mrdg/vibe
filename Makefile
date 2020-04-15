demo run: ringo
	./ringo -path demo -run demo/weird.txt

ringo: $(wildcard *.go dub/*.go)
	go build

test:
	go test ./...
