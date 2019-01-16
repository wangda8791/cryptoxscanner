// The MIT License (MIT)
//
// Copyright (c) 2018-2019 Cranky Kernel
//
// Permission is hereby granted, free of charge, to any person
// obtaining a copy of this software and associated documentation
// files (the "Software"), to deal in the Software without
// restriction, including without limitation the rights to use, copy,
// modify, merge, publish, distribute, sublicense, and/or sell copies
// of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be
// included in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
// MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS
// BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN
// ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package binance

import (
	"encoding/json"
	"gitlab.com/crankykernel/cryptotrader/binance"
	"gitlab.com/crankykernel/cryptoxscanner/log"
	"time"
)

type StreamClient struct {
	name    string
	client  *binance.StreamClient
	streams []string
}

func NewStreamClient(name string, streams ...string) *StreamClient {
	return &StreamClient{
		name:    name,
		streams: streams,
	}
}

func (s *StreamClient) ReadNext() ([]byte, error) {
	_, body, err := s.client.Next()
	return body, err
}

func (s *StreamClient) Decode(buf []byte) (*binance.CombinedStreamMessage, error) {
	var message binance.CombinedStreamMessage
	err := json.Unmarshal(buf, &message)
	return &message, err
}

func (s *StreamClient) Run(channel chan *binance.CombinedStreamMessage) {
	for {
		// Connect, runs in its own loop until connected.
		log.Printf("binance: connecting to stream [%s]\n", s.name)
		s.Connect()
		log.Printf("binance: connected to stream [%s]\n", s.name)

		// Read loop.
	ReadLoop:
		for {
			body, err := s.ReadNext()
			if err != nil {
				log.Printf("binance: read error on stream [%s]: %v\n",
					s.name, err)
				break ReadLoop
			}

			message, err := s.Decode(body)
			if err != nil {
				log.Printf("binance: failed to decode message on stream [%s]: %v\n",
					s.name, err)
				goto ReadLoop
			}

			channel <- message
		}

		time.Sleep(1 * time.Second)
	}
}

func (s *StreamClient) Connect() {
	for {
		client, err := binance.OpenStreams(s.streams...)
		if err == nil {
			s.client = client
			return
		}
		log.Printf("binance: failed to connect to stream [%s]: %v\n",
			s.name, err)
		time.Sleep(1 * time.Second)
	}
}
