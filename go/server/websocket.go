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

package server

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"gitlab.com/crankykernel/cryptoxscanner/log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

var wsConnectionTracker *WsConnectionTracker

func init() {
	wsConnectionTracker = NewWsConnectionTracker()
}

type WsConnectionTracker struct {
	Paths   map[string]map[*WebSocketClient]bool
	Clients map[*WebSocketClient]map[string]bool
	Lock    sync.RWMutex
}

func NewWsConnectionTracker() *WsConnectionTracker {
	return &WsConnectionTracker{
		Paths:   make(map[string]map[*WebSocketClient]bool),
		Clients: make(map[*WebSocketClient]map[string]bool),
	}
}

func (w *WsConnectionTracker) Add(path string, conn *WebSocketClient) {
	w.Lock.Lock()
	if w.Paths[path] == nil {
		w.Paths[path] = map[*WebSocketClient]bool{}
	}
	if w.Clients[conn] == nil {
		w.Clients[conn] = map[string]bool{}
	}
	w.Paths[path][conn] = true
	w.Clients[conn][path] = true
	defer w.Lock.Unlock()
}

func (w *WsConnectionTracker) Del(path string, conn *WebSocketClient) {
	w.Lock.Lock()

	w.Paths[path][conn] = false
	delete(w.Paths[path], conn)

	w.Clients[conn][path] = false
	delete(w.Clients[conn], path)

	defer w.Lock.Unlock()
}

type WebSocketClient struct {
	// The websocket connection.
	conn *websocket.Conn

	// The http request.
	r *http.Request

	// Data written into this Channel will be sent to the client.
	closeChannel chan bool
}

func NewWebSocketClient(c *websocket.Conn, r *http.Request) *WebSocketClient {
	return &WebSocketClient{
		conn:         c,
		closeChannel: make(chan bool, 1),
		r:            r,
	}
}

func (c *WebSocketClient) GetRemoteAddr() string {
	remoteAddr := c.r.Header.Get("x-forwarded-for")
	if remoteAddr != "" {
		return remoteAddr
	}
	remoteAddr = c.r.Header.Get("x-real-ip")
	if remoteAddr != "" {
		return remoteAddr
	}
	return c.r.RemoteAddr
}

func (c *WebSocketClient) GetRemoteHost() string {
	remoteAddr := c.GetRemoteAddr()
	return strings.Split(remoteAddr, ":")[0]
}

func (c *WebSocketClient) WriteTextMessage(msg []byte) error {
	return c.conn.WriteMessage(websocket.TextMessage, msg)
}

type TickerWebSocketHandler struct {
	upgrader      websocket.Upgrader
	clientsLock   sync.RWMutex
	binanceRunner *BinanceRunner
	source        *WsSourceCache
}

func NewWebSocketHandler(binanceRunner *BinanceRunner, source *WsSourceCache) *TickerWebSocketHandler {
	handler := TickerWebSocketHandler{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
			EnableCompression: true,
		},
		source:        source,
		binanceRunner: binanceRunner,
	}
	return &handler
}

func (h *TickerWebSocketHandler) Upgrade(w http.ResponseWriter, r *http.Request) (*WebSocketClient, error) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	return NewWebSocketClient(conn, r), nil
}

func (h *TickerWebSocketHandler) Handle(w http.ResponseWriter, r *http.Request) {
	client, err := h.Upgrade(w, r)
	if err != nil {
		log.Infof("Failed to upgrade websocket connection: %v", err)
		return
	}
	log.Infof("WebSocket connnected to %s: RemoteAddr=%v; Origin=%s",
		r.URL.String(),
		client.GetRemoteAddr(),
		r.Header.Get("origin"))

	wsConnectionTracker.Add(r.URL.String(), client)
	defer wsConnectionTracker.Del(r.URL.String(), client)

	symbol := r.FormValue("symbol")
	updateInterval, err := strconv.ParseInt(r.FormValue("updateInterval"), 10, 64)
	if err != nil {
		updateInterval = 0
	}
	lastUpdate := time.Time{}

	// The read loop just reads and discards message until an error is
	// received.
	go h.readLoop(client)

	if symbol != "" {
		channel := h.binanceRunner.SubscribeSymbol(symbol)
		defer h.binanceRunner.UnsubscribeSymbol(symbol, channel)
		for {
			select {
			case filteredMessage := <-channel:
				bytes, err := json.Marshal(filteredMessage)
				if err != nil {
					log.Infof("failed to marshal filtered ticker: %v", err)
					continue
				}

				if err := client.WriteTextMessage(bytes); err != nil {
					log.Info("error: websocket write error to %s: %v", client.GetRemoteAddr(), err)
					goto Done
				}
			case <-client.closeChannel:
				goto Done
			}
		}
	} else {
		channel := h.source.Subscribe()
		defer h.source.Unsubscribe(channel)
		for {
			select {
			case trackers := <-channel:
				if trackers == nil {
					goto Done
				}
				if time.Now().Sub(lastUpdate) < time.Second*time.Duration(updateInterval) {
					continue
				}
				if err := client.conn.SetWriteDeadline(time.Now().Add(time.Second * 6)); err != nil {
					log.WithError(err).Warnf("Failed to send websocket write deadline")
				}
				if err := client.conn.WritePreparedMessage(trackers); err != nil {
					log.WithError(err).Errorf("Failed to write websocket prepared message")
					goto Done
				}
				lastUpdate = time.Now()
			case <-client.closeChannel:
				goto Done
			}
		}
	}
Done:
	client.conn.Close()
	close(client.closeChannel)
	log.Infof("WebSocket connection closed: %v", client.GetRemoteAddr())
}

func (h *TickerWebSocketHandler) readLoop(client *WebSocketClient) {
	for {
		if _, _, err := client.conn.ReadMessage(); err != nil {
			break
		}
	}
	select {
	case client.closeChannel <- true:
	default:
	}
}

type TickerStream struct {
	Tickers *[]interface{} `json:"tickers"`
}

type WsSourceCache struct {
	subscribers map[chan *websocket.PreparedMessage]bool
	source      chan *TickerTrackerMap
	builder     func(trackerMap *TickerTrackerMap) []interface{}
}

func NewWsSourceCache(source chan *TickerTrackerMap, builder func(trackerMap *TickerTrackerMap) []interface{}) *WsSourceCache {
	return &WsSourceCache{
		subscribers: map[chan *websocket.PreparedMessage]bool{},
		source:      source,
		builder:     builder,
	}
}

func (f *WsSourceCache) Subscribe() chan *websocket.PreparedMessage {
	channel := make(chan *websocket.PreparedMessage, 1)
	f.subscribers[channel] = true
	return channel
}

func (f *WsSourceCache) Unsubscribe(channel chan *websocket.PreparedMessage) {
	delete(f.subscribers, channel)
}

func (f *WsSourceCache) Run() {
	for {
		trackers := <-f.source
		message := f.builder(trackers)
		output := TickerStream{Tickers: &message}
		buf, err := json.Marshal(output)
		if err != nil {
			log.Errorf("Failed to encode monitor message as JSON: %v", err)
			continue
		}
		pm, err := websocket.NewPreparedMessage(websocket.TextMessage, buf)
		if err != nil {
			log.Errorf("Failed to prepare monitor websocket message: %v", err)
			continue
		}
		for subscriber := range f.subscribers {
			select {
			case subscriber <- pm:
			default:
			}
		}
	}
}

func WsBuildCompleteMessage(trackers *TickerTrackerMap) []interface{} {
	entries := []interface{}{}
	for key := range trackers.Trackers {
		tracker := trackers.Trackers[key]
		m := WsBuildCompleteEntry(tracker)
		if m != nil {
			entries = append(entries, m)
		}
	}
	return entries
}

func WsBuildCompleteEntry(tracker *TickerTracker) map[string]interface{} {
	last := tracker.LastTick()
	if last == nil {
		return nil
	}
	key := tracker.Symbol

	message := map[string]interface{}{
		"symbol": key,
		"close":  last.CurrentDayClose,
		"bid":    last.Bid,
		"ask":    last.Ask,
		"high":   last.HighPrice,
		"low":    last.LowPrice,
		"volume": last.TotalQuoteVolume,

		"price_change_pct": map[string]float64{
			"1m":  tracker.Metrics[1].PriceChangePercent,
			"5m":  tracker.Metrics[5].PriceChangePercent,
			"10m": tracker.Metrics[10].PriceChangePercent,
			"15m": tracker.Metrics[15].PriceChangePercent,
			"1h":  tracker.Metrics[60].PriceChangePercent,
			"24h": tracker.LastTick().PriceChangePercent,
		},

		"volume_change_pct": map[string]float64{
			"1m":  tracker.Metrics[1].VolumeChangePercent,
			"2m":  tracker.Metrics[2].VolumeChangePercent,
			"3m":  tracker.Metrics[3].VolumeChangePercent,
			"4m":  tracker.Metrics[4].VolumeChangePercent,
			"5m":  tracker.Metrics[5].VolumeChangePercent,
			"10m": tracker.Metrics[10].VolumeChangePercent,
			"15m": tracker.Metrics[15].VolumeChangePercent,
			"1h":  tracker.Metrics[60].VolumeChangePercent,
		},

		"timestamp": last.Timestamp(),
	}

	for _, bucket := range Buckets {
		metrics := tracker.Metrics[bucket]

		message[fmt.Sprintf("l_%d", bucket)] = metrics.Low
		message[fmt.Sprintf("h_%d", bucket)] = metrics.High

		message[fmt.Sprintf("r_%d", bucket)] = metrics.Range
		message[fmt.Sprintf("rp_%d", bucket)] = metrics.RangePercent
	}

	message["r_24"] = tracker.H24Metrics.Range
	message["rp_24"] = tracker.H24Metrics.RangePercent

	if tracker.HaveVwap {
		for i, k := range tracker.Metrics {
			message[fmt.Sprintf("vwap_%dm", i)] = Round8(k.Vwap)
		}
	}

	if tracker.HaveTotalVolume {
		for i, k := range tracker.Metrics {
			message[fmt.Sprintf("total_volume_%d", i)] = Round8(k.TotalVolume)
		}
	}

	if tracker.HaveNetVolume {
		for i, k := range tracker.Metrics {
			message[fmt.Sprintf("nv_%d", i)] = Round8(k.NetVolume)
			message[fmt.Sprintf("bv_%d", i)] = Round8(k.BuyVolume)
			message[fmt.Sprintf("sv_%d", i)] = Round8(k.SellVolume)
		}
	}

	for i, k := range tracker.Metrics {
		if !math.IsNaN(k.RSI) {
			message[fmt.Sprintf("rsi_%d", i*60)] = Round8(k.RSI)
		}
	}

	return message
}

func WsBuildMonitorMessage(trackers *TickerTrackerMap) []interface{} {
	entries := []interface{}{}
	for key := range trackers.Trackers {
		tracker := trackers.Trackers[key]
		last := tracker.LastTick()
		if last == nil {
			continue
		}
		key := tracker.Symbol

		entry := map[string]interface{}{
			"symbol": key,
			"close":  last.CurrentDayClose,
			"bid":    last.Bid,
			"ask":    last.Ask,
			"high":   last.HighPrice,
			"low":    last.LowPrice,
			"volume": last.TotalQuoteVolume,

			"price_change_pct": map[string]float64{
				"1m":  tracker.Metrics[1].PriceChangePercent,
				"5m":  tracker.Metrics[5].PriceChangePercent,
				"10m": tracker.Metrics[10].PriceChangePercent,
				"15m": tracker.Metrics[15].PriceChangePercent,
				"1h":  tracker.Metrics[60].PriceChangePercent,
				"24h": tracker.LastTick().PriceChangePercent,
			},

			"volume_change_pct": map[string]float64{
				"1m":  tracker.Metrics[1].VolumeChangePercent,
				"2m":  tracker.Metrics[2].VolumeChangePercent,
				"3m":  tracker.Metrics[3].VolumeChangePercent,
				"4m":  tracker.Metrics[4].VolumeChangePercent,
				"5m":  tracker.Metrics[5].VolumeChangePercent,
				"10m": tracker.Metrics[10].VolumeChangePercent,
				"15m": tracker.Metrics[15].VolumeChangePercent,
				"1h":  tracker.Metrics[60].VolumeChangePercent,
			},

			"timestamp": last.Timestamp(),
		}
		entries = append(entries, entry)
	}

	return entries
}
