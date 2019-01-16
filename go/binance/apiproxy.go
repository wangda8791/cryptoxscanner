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
	"bytes"
	"fmt"
	"gitlab.com/crankykernel/cryptoxscanner/log"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

type proxyCacheEntry struct {
	timestamp time.Time
	content   []byte
	header    http.Header
}

type ApiProxy struct {
	cache map[string]*proxyCacheEntry
	lock  sync.RWMutex
}

func NewApiProxy() *ApiProxy {
	return &ApiProxy{
		cache: make(map[string]*proxyCacheEntry),
	}
}

func (p *ApiProxy) AddToCache(key string, entry proxyCacheEntry) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.cache[key] = &entry
}

func (p *ApiProxy) GetFromCache(key string) *proxyCacheEntry {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.cache[key]
}

func (p *ApiProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	url := fmt.Sprintf("https://api.binance.com%s",
		r.URL.RequestURI()[len("/api/1/binance/proxy"):])

	cached := p.GetFromCache(url)
	if cached != nil {
		if time.Now().Sub(cached.timestamp) <= time.Second*1 {
			w.Header().Add("content-type", cached.header.Get("content-type"))
			w.Header().Add("access-control-allow-origin", "*")
			io.Copy(w, bytes.NewReader(cached.content))
			return
		}
	}

	proxyRequest, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("error: %v\n", err)
	}

	proxyResponse, err := http.DefaultClient.Do(proxyRequest)
	if err != nil {
		log.Printf("error: %v", err)
		return
	}
	defer proxyResponse.Body.Close()

	w.Header().Add("content-type", proxyResponse.Header.Get("content-type"))
	w.Header().Add("access-control-allow-origin", "*")
	w.WriteHeader(proxyResponse.StatusCode)

	content, _ := ioutil.ReadAll(proxyResponse.Body)
	p.AddToCache(url, proxyCacheEntry{
		timestamp: time.Now(),
		content:   content,
		header:    proxyResponse.Header,
	})
	w.Write(content)
}
