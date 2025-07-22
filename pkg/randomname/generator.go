package randomname

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

var adjectives = []string{
	"brave", "calm", "eager", "fancy", "gentle", "happy", "jolly", "kind",
	"lively", "nice", "proud", "silly", "witty", "zealous", "mighty", "swift",
	"sharp", "bold", "courageous", "resilient", "daring", "bright", "creative",
	"innovative", "dynamic", "energetic", "vibrant", "radiant", "sincere", "honest",
	"steadfast", "ardent", "spirited", "graceful", "gritty", "focused", "optimistic",
	"robust", "stalwart", "resolute", "vigorous",
}

var nouns = []string{
	"squirrel", "tiger", "eagle", "dolphin", "panther", "lion", "panda", "koala",
	"whale", "shark", "wolf", "falcon", "otter", "rabbit", "bear", "fox", "hedgehog",
	"owl", "leopard", "cheetah", "hyena", "buffalo", "zebra", "giraffe", "coyote",
	"raccoon", "badger", "moose", "stallion", "gazelle", "mongoose", "cougar", "jaguar",
	"bison", "viper", "python", "cobra", "lizard", "frog", "beaver", "porcupine",
	"skunk", "antelope", "hamster", "gerbil",
}

var (
	rnd = rand.New(rand.NewSource(time.Now().UnixNano()))
	mu  sync.Mutex
	// used keeps track of generated names to ensure uniqueness within a session
	used = make(map[string]bool)
)

// generate is a helper function that handles the common logic for generating names.
// If withSuffix is true, it adds a 6-character hexadecimal suffix to the name.
// This version reserves the candidate name immediately to avoid race conditions:
// 1. Generate and reserve the candidate under lock.
// 2. Release the lock and execute the callback.
// 3. If the callback rejects the candidate, remove it from the reservation.
func generate(check func(name string) bool, withSuffix bool) string {
	for {
		// Generate candidate name and reserve it.
		mu.Lock()
		adj := adjectives[rnd.Intn(len(adjectives))]
		noun := nouns[rnd.Intn(len(nouns))]
		var candidate string

		if withSuffix {
			suffix := rnd.Intn(1 << 24) // random 24-bit number
			candidate = fmt.Sprintf("%s-%s-%06x", adj, noun, suffix)
		} else {
			candidate = fmt.Sprintf("%s-%s", adj, noun)
		}

		if used[candidate] {
			mu.Unlock()
			continue
		}

		// Reserve the candidate immediately.
		used[candidate] = true
		mu.Unlock()

		// Execute callback outside the lock.
		if check != nil && !check(candidate) {
			// Callback rejected the candidate, so remove it.
			mu.Lock()
			delete(used, candidate)
			mu.Unlock()
			continue
		}

		return candidate
	}
}

// Generate returns a random name in the format "adjective-noun-xxxxxx".
// The "xxxxxx" suffix is a 6-character hexadecimal number, making collisions extremely unlikely.
// It ensures uniqueness within the current session and allows for external validation through
// the optional check callback.
func Generate(check func(name string) bool) string {
	return generate(check, true)
}

// GenerateSimple returns a random name in the format "adjective-noun"
// without the hexadecimal suffix. Note that this has a higher chance of collisions
// due to the smaller namespace.
func GenerateSimple(check func(name string) bool) string {
	return generate(check, false)
}

// Reset clears the internal cache of used names.
func Reset() {
	mu.Lock()
	used = make(map[string]bool)
	mu.Unlock()
}
