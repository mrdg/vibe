run: ringo
	./ringo -bpm 130 -samples "samples/*.wav" -beat 5/4 -run samples/commands.txt

ringo: $(wildcard *.go)
	go build
