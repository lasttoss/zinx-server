package handlers

import (
	"sync"
	"testing"
)

// TestRoomCreatesOneRoomPerName is the crash this code used to have: the router is registered as
// an empty literal, so the room map starts out nil, and writing a nil map panics the process.
func TestRoomCreatesOneRoomPerName(t *testing.T) {
	router := &ChatRouter{}

	room := router.room("default")
	if room == nil {
		t.Fatal("room() returned nil for a fresh router")
	}
	if room.Name != "default" {
		t.Errorf("room name = %q, want default", room.Name)
	}
	if again := router.room("default"); again != room {
		t.Error("room() built a second room for the same name: the clients would be split")
	}
	if other := router.room("guild"); other == room {
		t.Error("room() returned the same room for two different names")
	}
}

// TestRoomIsSafeForClientsJoiningAtOnce: one goroutine per connection, so a dozen players opening
// the chat at the same time is a dozen writers on the map. Run with -race, this fails on the
// unsynchronised version with "concurrent map writes".
func TestRoomIsSafeForClientsJoiningAtOnce(t *testing.T) {
	router := &ChatRouter{}

	var wg sync.WaitGroup
	rooms := make([]string, 32)
	for i := range rooms {
		rooms[i] = "default"
		if i%4 == 0 {
			rooms[i] = "guild"
		}
	}
	for _, name := range rooms {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			if router.room(name) == nil {
				t.Error("room() returned nil under concurrency")
			}
		}(name)
	}
	wg.Wait()

	router.mu.Lock()
	defer router.mu.Unlock()
	if len(router.rooms) != 2 {
		t.Errorf("rooms = %d, want 2: concurrent joins must not create duplicates", len(router.rooms))
	}
}
