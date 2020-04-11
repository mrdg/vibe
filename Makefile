demo run: ringo
	./ringo -path demo -run demo/commands.txt

ringo: $(wildcard *.go dub/*.go)
	go build

test:
	go test ./...
