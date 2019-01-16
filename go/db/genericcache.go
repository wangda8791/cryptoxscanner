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

package db

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"gitlab.com/crankykernel/cryptoxscanner/log"
	"os"
	"sync"
	"time"
)

const defaultCacheTtl = 3600 * 2

var genericCacheMap map[string]*GenericCache

var genericCacheMapLock sync.Mutex

type GenericCache struct {
	name       string
	db         *sql.DB
	tx         *sql.Tx
	lastCommit time.Time
	itemCount  uint64
	lock       sync.RWMutex
}

func init() {
	genericCacheMap = make(map[string]*GenericCache)
}

func OpenGenericCache(name string) (*GenericCache, error) {
	genericCacheMapLock.Lock()
	defer genericCacheMapLock.Unlock()
	if cache, ok := genericCacheMap[name]; ok {
		return cache, nil
	}

	filename := fmt.Sprintf("./%s.sqlite", name)

	if _, err := os.Stat(filename); err != nil {
		log.Infof("Creating generic cache database %s.", filename)
	} else {
		log.Infof("Opening generic cache database %s.", filename)
	}

	db, err := sql.Open("sqlite3",
		fmt.Sprintf("%s?cache=shared&mode=rwc&_busy_timeout=3000", filename))
	if err != nil {
		return nil, err
	}

	cache := &GenericCache{
		name: name,
		db:   db,
	}

	if err := cache.migrate(); err != nil {
		return nil, err
	}

	genericCacheMap[name] = cache
	return cache, nil
}

func (c *GenericCache) AddItem(timestamp time.Time, itemType string, body []byte) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.lastCommit.IsZero() {
		c.lastCommit = time.Now()
	}

	if c.tx == nil {
		tx, err := c.db.Begin()
		if err != nil {
			log.WithError(err).WithFields(log.Fields{
				"cache": c.name,
			}).Errorf("Failed to begin transaction.")
			return
		}
		c.tx = tx
	}

	sql := fmt.Sprintf(`insert into cache (timestamp, type, data) values
		(?, ?, ?)`)
	_, err := c.tx.Exec(sql, timestamp.Unix(), itemType, body)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"cache": c.name,
		}).Errorf("Failed to execute statement.")
		c.tx.Rollback()
		c.tx = nil
		return
	}
	c.itemCount += 1

	if time.Now().Sub(c.lastCommit) > time.Second {
		start := time.Now()
		n, err := c.expireItems(c.tx)
		if err != nil {
			log.WithError(err).WithFields(log.Fields{
				"cache": c.name,
			}).Errorf("Failed to purge expired items.")
		}
		if err := c.tx.Commit(); err != nil {
			log.WithError(err).WithFields(log.Fields{
				"cache": c.name,
			}).Errorf("Failed to commit transaction.")
			c.tx = nil
			return
		}
		log.WithFields(log.Fields{
			"duration": time.Now().Sub(start),
			"cache":    c.name,
			"deleted":  n,
		}).Infof("Committed %d items.", c.itemCount)
		c.tx = nil
		c.itemCount = 0
		c.lastCommit = time.Now()
	}
}

func (c *GenericCache) expireItems(tx *sql.Tx) (int64, error) {
	sql := fmt.Sprintf("delete from cache where timestamp < ?")
	timestamp := time.Now().Add(-1 * defaultCacheTtl * time.Second)
	res, err := tx.Exec(sql, timestamp.Unix())
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	return n, err
}

func (c *GenericCache) QueryAgeLessThan(itemType string, seconds int64) (*sql.Rows, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	age := time.Now().Unix() - seconds
	sql := fmt.Sprintf("select data from cache where timestamp > ? and type = ? order by timestamp")
	rows, err := c.db.Query(sql, age, itemType)
	return rows, err
}

func (c *GenericCache) migrate() error {
	var version = 0
	tx, err := c.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	row := tx.QueryRow("select max(version) from schema")
	if err := row.Scan(&version); err != nil {
		log.Infof("Initializing database for cache %s", c.name)
		_, err := tx.Exec("create table schema (version integer not null primary key, timestamp timestamp)")
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to create schema table: %v", err)
		}
		if err := c.incrementVersion(tx, 0); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to insert into schema table: %v", err)
		}
		version = 0
	} else {
		log.Printf("Found database version %d.", version)
	}

	if version < 1 {
		log.Infof("Migrating database to v1.")
		_, err := tx.Exec(`
create table cache (timestamp integer, type string, data blob);
create index cache_index on cache (timestamp, type);
`)
		if err != nil {
			tx.Rollback()
			return err
		}
		if err := c.incrementVersion(tx, 1); err != nil {
			tx.Rollback()
			return err
		}
	}

	tx.Commit()
	return nil
}

func (c *GenericCache) incrementVersion(tx *sql.Tx, version int) error {
	_, err := tx.Exec("insert into schema values (?, 'now')", version)
	return err
}
