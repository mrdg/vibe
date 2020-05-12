.PHONY: demo
demo run: vibe
	./vibe -run demo/demo.txt

new: vibe
	./vibe -run demo/new.txt

vibe: $(wildcard *.go **/*.go)
	go build

test:
	go test ./...

clean:
	rm -f vibe
