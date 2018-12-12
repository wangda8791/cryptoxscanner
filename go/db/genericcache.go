// Copyright (C) 2018 Cranky Kernel
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

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
