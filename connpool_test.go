// SPDX-FileCopyrightText: 2022-2024 The go-mail Authors
//
// SPDX-License-Identifier: MIT

package mail

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"
)

func TestNewConnPool(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serverPort := TestServerPortBase + 10
	featureSet := "250-AUTH PLAIN\r\n250-8BITMIME\r\n250-DSN\r\n250 SMTPUTF8"
	go func() {
		if err := simpleSMTPServer(ctx, featureSet, true, serverPort); err != nil {
			t.Errorf("failed to start test server: %s", err)
			return
		}
	}()
	time.Sleep(time.Millisecond * 300)

	pool, err := newConnPool(serverPort)
	if err != nil {
		t.Errorf("failed to create connection pool: %s", err)
	}
	defer pool.Close()
	if pool == nil {
		t.Errorf("connection pool is nil")
		return
	}
	if pool.Size() != 5 {
		t.Errorf("expected 5 connections, got %d", pool.Size())
	}
	conn, err := pool.Get()
	if err != nil {
		t.Errorf("failed to get connection: %s", err)
	}
	if _, err := conn.Write([]byte("EHLO test.localhost.localdomain\r\nQUIT\r\n")); err != nil {
		t.Errorf("failed to write quit command to first connection: %s", err)
	}
}

func TestConnPool_Get_Type(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serverPort := TestServerPortBase + 11
	featureSet := "250-AUTH PLAIN\r\n250-8BITMIME\r\n250-DSN\r\n250 SMTPUTF8"
	go func() {
		if err := simpleSMTPServer(ctx, featureSet, true, serverPort); err != nil {
			t.Errorf("failed to start test server: %s", err)
			return
		}
	}()
	time.Sleep(time.Millisecond * 300)

	pool, err := newConnPool(serverPort)
	if err != nil {
		t.Errorf("failed to create connection pool: %s", err)
	}
	defer pool.Close()

	conn, err := pool.Get()
	if err != nil {
		t.Errorf("failed to get new connection from pool: %s", err)
		return
	}

	_, ok := conn.(*PoolConn)
	if !ok {
		t.Error("received connection from pool is not of type PoolConn")
		return
	}
	if _, err := conn.Write([]byte("EHLO test.localhost.localdomain\r\nQUIT\r\n")); err != nil {
		t.Errorf("failed to write quit command to first connection: %s", err)
	}
}

func TestConnPool_Get(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serverPort := TestServerPortBase + 12
	featureSet := "250-AUTH PLAIN\r\n250-8BITMIME\r\n250-DSN\r\n250 SMTPUTF8"
	go func() {
		if err := simpleSMTPServer(ctx, featureSet, true, serverPort); err != nil {
			t.Errorf("failed to start test server: %s", err)
			return
		}
	}()
	time.Sleep(time.Millisecond * 300)

	p, _ := newConnPool(serverPort)
	defer p.Close()

	conn, err := p.Get()
	if err != nil {
		t.Errorf("failed to get new connection from pool: %s", err)
		return
	}
	if _, err = conn.Write([]byte("EHLO test.localhost.localdomain\r\nQUIT\r\n")); err != nil {
		t.Errorf("failed to write quit command to first connection: %s", err)
	}

	if p.Size() != 4 {
		t.Errorf("getting new connection from pool failed. Expected pool size: 4, got %d", p.Size())
	}

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			wgconn, err := p.Get()
			if err != nil {
				t.Errorf("failed to get new connection from pool: %s", err)
			}
			if _, err = wgconn.Write([]byte("EHLO test.localhost.localdomain\r\nQUIT\r\n")); err != nil {
				t.Errorf("failed to write quit command to first connection: %s", err)
			}
		}()
	}
	wg.Wait()

	if p.Size() != 0 {
		t.Errorf("Get error. Expecting 0, got %d", p.Size())
	}

	conn, err = p.Get()
	if err != nil {
		t.Errorf("failed to get new connection from pool: %s", err)
	}
	if _, err = conn.Write([]byte("EHLO test.localhost.localdomain\r\nQUIT\r\n")); err != nil {
		t.Errorf("failed to write quit command to first connection: %s", err)
	}
	p.Close()
}

func newConnPool(port int) (Pool, error) {
	netDialer := net.Dialer{}
	return NewConnPool(context.Background(), 5, 30, netDialer.DialContext, "tcp",
		fmt.Sprintf("127.0.0.1:%d", port))
}
