package server_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"git.maronato.dev/maronato/finger/internal/config"
	"git.maronato.dev/maronato/finger/internal/log"
	"git.maronato.dev/maronato/finger/internal/server"
	"git.maronato.dev/maronato/finger/internal/webfinger"
)

func getPortGenerator() func() int {
	lock := &sync.Mutex{}
	port := 8080

	return func() int {
		lock.Lock()
		defer lock.Unlock()

		port++

		return port
	}
}

func TestStartServer(t *testing.T) {
	t.Parallel()

	portGenerator := getPortGenerator()

	t.Run("starts and shuts down", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
		defer cancel()

		cfg := config.NewConfig()
		l := log.NewLogger(&strings.Builder{}, cfg)

		ctx = log.WithLogger(ctx, l)

		// Use a new port
		cfg.Port = fmt.Sprint(portGenerator())

		// Start the server
		err := server.StartServer(ctx, cfg, nil)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("fails to start", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*200)
		defer cancel()

		cfg := config.NewConfig()
		l := log.NewLogger(&strings.Builder{}, cfg)

		ctx = log.WithLogger(ctx, l)

		// Use a new port
		cfg.Port = fmt.Sprint(portGenerator())

		// Use invalid host
		cfg.Host = "google.com"

		// Start the server
		err := server.StartServer(ctx, cfg, nil)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})

	t.Run("serves webfinger", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*200)
		defer cancel()

		cfg := config.NewConfig()
		l := log.NewLogger(&strings.Builder{}, cfg)

		ctx = log.WithLogger(ctx, l)

		// Use a new port
		cfg.Port = fmt.Sprint(portGenerator())

		resource := "acct:user@example.com"
		webfingers := webfinger.WebFingers{
			resource: &webfinger.WebFinger{
				Subject: resource,
				Properties: map[string]string{
					"http://webfinger.net/rel/name": "John Doe",
				},
			},
		}

		go func() {
			// Start the server
			err := server.StartServer(ctx, cfg, webfingers)
			if err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		}()

		// Wait for the server to start
		time.Sleep(time.Millisecond * 50)

		// Create a new client
		c := http.Client{}

		// Create a new request
		r, _ := http.NewRequestWithContext(ctx,
			http.MethodGet,
			"http://"+cfg.GetAddr()+"/.well-known/webfinger?resource=acct:user@example.com",
			http.NoBody,
		)

		// Send the request
		resp, err := c.Do(r)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		defer resp.Body.Close()

		// Check the status code
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, resp.StatusCode)
		}

		// Check the response body
		fingerGot := &webfinger.WebFinger{}

		// Decode the response body
		if err := json.NewDecoder(resp.Body).Decode(fingerGot); err != nil {
			t.Errorf("error decoding json: %v", err)
		}

		// Check the response body
		fingerWant := webfingers[resource]

		if !reflect.DeepEqual(fingerGot, fingerWant) {
			t.Errorf("expected %v, got %v", fingerWant, fingerGot)
		}
	})

	t.Run("serves healthcheck", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*200)
		defer cancel()

		cfg := config.NewConfig()
		l := log.NewLogger(&strings.Builder{}, cfg)

		ctx = log.WithLogger(ctx, l)

		// Use a new port
		cfg.Port = fmt.Sprint(portGenerator())

		go func() {
			// Start the server
			err := server.StartServer(ctx, cfg, nil)
			if err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		}()

		// Wait for the server to start
		time.Sleep(time.Millisecond * 50)

		// Create a new client
		c := http.Client{}

		// Create a new request
		r, _ := http.NewRequestWithContext(ctx,
			http.MethodGet,
			"http://"+cfg.GetAddr()+"/healthz",
			http.NoBody,
		)

		// Send the request
		resp, err := c.Do(r)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		defer resp.Body.Close()

		// Check the status code
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, resp.StatusCode)
		}
	})
}
