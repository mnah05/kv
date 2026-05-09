package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/mnah05/kv/handler"
	"github.com/mnah05/kv/store"
)

var (
	st             *store.Store
	checkpointStop = make(chan struct{})
	checkpointWg   sync.WaitGroup
)

func main() {
	var err error
	st, err = store.Open("/tmp/demo.log")
	if err != nil {
		log.Fatal(err)
	}

	checkpointWg.Add(1)
	go periodicCheckpoint()

	mux := http.NewServeMux()
	mux.HandleFunc("/put", handler.HandlePut(st))
	mux.HandleFunc("/get", handler.HandleGet(st))
	mux.HandleFunc("/delete", handler.HandleDelete(st))

	errCh := make(chan error, 1)
	srv := &http.Server{Addr: ":8080", Handler: mux}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	log.Println("listening on :8080")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigCh:
	case err := <-errCh:
		log.Printf("server error: %v", err)
	}

	log.Println("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}

	close(checkpointStop)
	checkpointWg.Wait()

	if err := st.Checkpoint(); err != nil {
		log.Printf("final checkpoint failed: %v", err)
	}

	if err := st.Close(); err != nil {
		log.Printf("store close error: %v", err)
	}
}

func periodicCheckpoint() {
	defer checkpointWg.Done()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := st.Checkpoint(); err != nil {
				log.Printf("checkpoint failed: %v", err)
			}
		case <-checkpointStop:
			return
		}
	}
}
