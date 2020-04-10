demo run: ringo
	./ringo -sounds "demo/*.wav" -run demo/commands.txt

ringo: $(wildcard *.go dub/*.go)
	go build

test:
	go test ./...
