run: ringo
	./ringo -bpm 130 -samples "samples/*.wav" -beat 5/4

ringo: $(wildcard *.go)
	go build
