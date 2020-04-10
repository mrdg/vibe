demo run: ringo
	./ringo -samples "demo/*.wav" -run demo/commands.txt

ringo: $(wildcard *.go dub/*.go)
	go build

test:
	go test ./...
