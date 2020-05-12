package audio

import (
	"github.com/gordonklaus/portaudio"
)

type Source interface {
	Process([][]float32)
}

type Ticker interface {
	Tick(numSamples int)
}

func NewSink() (*Sink, error) {
	if err := portaudio.Initialize(); err != nil {
		return nil, err
	}
	var s Sink
	stream, err := portaudio.OpenDefaultStream(0, 2, sampleRate, bufferSize, s.Process)
	if err != nil {
		return nil, err
	}
	s.stream = stream
	return &s, nil
}

func (s *Sink) Start() error {
	return s.stream.Start()
}

type Sink struct {
	sources []Source
	tickers []Ticker
	stream  *portaudio.Stream
}

func (s *Sink) Stop() error {
	s.stream.Close()
	portaudio.Terminate()
	return nil
}

func (s *Sink) AddSources(sources ...Source) {
	s.sources = append(s.sources, sources...)
}

func (s *Sink) AddTicker(ticker Ticker) {
	s.tickers = append(s.tickers, ticker)
}

func (s *Sink) Process(samples [][]float32) {
	for i := range samples {
		for j := range samples[i] {
			samples[i][j] = 0.
		}
	}
	for _, ticker := range s.tickers {
		ticker.Tick(len(samples[0]))
	}
	for _, source := range s.sources {
		source.Process(samples)
	}
}
