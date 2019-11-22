package main

import (
	"time"

	"sync"

	gin "gopkg.in/gin-gonic/gin.v1"
)

var once sync.Once

var session *Session

type Session struct {
	kvs map[string]interface{}
	kes map[string]time.Time
	sync.RWMutex
}

func (s *Session) Get(key string) interface{} {
	s.RLock()
	defer s.RUnlock()
	return s.kvs[key]
}

func (s *Session) Set(key string, val interface{}) {
	s.Lock()
	defer s.Unlock()
	s.kvs[key] = val
	s.kes[key] = time.Now().Add(time.Minute * 5)
}

func (s *Session) Delete(key string) {
	s.Lock()
	defer s.Unlock()
	delete(s.kes, key)
	delete(s.kvs, key)
}

func (s *Session) Expired() {
	s.Lock()
	defer s.Unlock()
	for k, e := range s.kes {
		if e.Before(time.Now()) {
			delete(s.kes, k)
			delete(s.kvs, k)
		}
	}
}

func NewSession() *Session {
	once.Do(func() {
		session = &Session{
			kvs: make(map[string]interface{}),
			kes: make(map[string]time.Time),
		}
		go func() {
			ticker := time.NewTicker(time.Hour)
			for {
				select {
				case <-ticker.C:
					session.Expired()
				}
			}
		}()
	})
	return session
}

func getsessions(c *gin.Context) *Session {
	return NewSession()
}

// Token 用户
type Token struct {
	Phone      string `json:"phone"`
	SendTxCode string `json:"sendtxcode"`
	SendTxTime int64  `json:"sendtxtime"`
}
