package randomname

import (
	"fmt"
	"regexp"
	"sync"
	"testing"
	"time"
)

func TestGenerate(t *testing.T) {
	pattern := regexp.MustCompile(`^[a-z]+-[a-z]+-[0-9a-f]{6}$`)

	name := Generate(nil)
	if !pattern.MatchString(name) {
		t.Errorf("Generated name %q doesn't match pattern %v", name, pattern)
	}

	name = Generate(func(string) bool { return true })
	if !pattern.MatchString(name) {
		t.Errorf("Generated name %q doesn't match pattern %v", name, pattern)
	}
}

func TestGenerateSimple(t *testing.T) {
	pattern := regexp.MustCompile(`^[a-z]+-[a-z]+$`)

	name := GenerateSimple(nil)
	if !pattern.MatchString(name) {
		t.Errorf("Generated name %q doesn't match pattern %v", name, pattern)
	}

	name = GenerateSimple(func(string) bool { return true })
	if !pattern.MatchString(name) {
		t.Errorf("Generated name %q doesn't match pattern %v", name, pattern)
	}
}

func TestUniqueness(t *testing.T) {
	// Reset the used map before testing
	Reset()

	// Generate a set of names and ensure they're unique
	iterations := 1000
	names := make(map[string]bool)

	for i := 0; i < iterations; i++ {
		name := Generate(nil)
		if names[name] {
			t.Errorf("Generate() produced duplicate name: %s", name)
		}
		names[name] = true
	}

	// Test simple names as well
	Reset()
	simpleNames := make(map[string]bool)

	// Since there are fewer possible combinations for simple names,
	// we'll test with a smaller set
	iterations = 100
	for i := 0; i < iterations; i++ {
		name := GenerateSimple(nil)
		if simpleNames[name] {
			t.Errorf("GenerateSimple() produced duplicate name: %s", name)
		}
		simpleNames[name] = true
	}
}

func TestCallbackRejection(t *testing.T) {
	Reset()

	// Create a deterministic pattern to test
	attempts := 0
	check := func(name string) bool {
		attempts++
		// Accept only after 3 attempts
		return attempts > 3
	}

	// Generate a name - should take exactly 4 attempts
	name := Generate(check)
	if attempts != 4 {
		t.Errorf("Expected 4 attempts, got %d", attempts)
	}

	// Verify the accepted name is in the used map
	if !used[name] {
		t.Errorf("Accepted name %s not stored in used map", name)
	}
}

func TestWordListCoverage(t *testing.T) {
	Reset()

	// Create maps to track word usage
	adjUsed := make(map[string]bool)
	nounUsed := make(map[string]bool)

	// Run a reasonable number of iterations to get good coverage
	maxIterations := len(adjectives) * len(nouns) / 10
	for i := 0; i < maxIterations; i++ {
		name := GenerateSimple(nil)
		parts := regexp.MustCompile(`-`).Split(name, 2)
		if len(parts) != 2 {
			t.Errorf("Generated name %s doesn't contain exactly one hyphen", name)
			continue
		}
		adjUsed[parts[0]] = true
		nounUsed[parts[1]] = true

		// Break early if we've seen all words
		if len(adjUsed) == len(adjectives) && len(nounUsed) == len(nouns) {
			break
		}
	}

	// Report coverage percentages instead of failing
	adjCoverage := float64(len(adjUsed)) / float64(len(adjectives)) * 100
	nounCoverage := float64(len(nounUsed)) / float64(len(nouns)) * 100

	t.Logf("Adjective coverage: %.1f%% (%d/%d)", adjCoverage, len(adjUsed), len(adjectives))
	t.Logf("Noun coverage: %.1f%% (%d/%d)", nounCoverage, len(nounUsed), len(nouns))

	// Warn if coverage is suspiciously low
	if adjCoverage < 50 {
		t.Errorf("Suspiciously low adjective coverage: %.1f%%", adjCoverage)
	}
	if nounCoverage < 50 {
		t.Errorf("Suspiciously low noun coverage: %.1f%%", nounCoverage)
	}
}

func TestConcurrency(t *testing.T) {
	Reset()

	// Reduced number of workers and iterations for faster testing
	workers := 5
	iterations := 20
	timeout := time.After(5 * time.Second)

	var wg sync.WaitGroup
	namesChan := make(chan string, workers*iterations)
	errorChan := make(chan error, workers)

	// Launch workers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				select {
				case <-timeout:
					errorChan <- fmt.Errorf("timeout reached")
					return
				default:
					name := Generate(nil)
					namesChan <- name
				}
			}
		}()
	}

	// Wait for completion or timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(namesChan)
		close(done)
	}()

	select {
	case <-timeout:
		t.Fatal("Test timed out")
	case err := <-errorChan:
		t.Fatal(err)
	case <-done:
		// Success path
	}

	// Check for uniqueness
	names := make(map[string]bool)
	for name := range namesChan {
		if names[name] {
			t.Errorf("Duplicate name generated: %s", name)
		}
		names[name] = true
	}
}
