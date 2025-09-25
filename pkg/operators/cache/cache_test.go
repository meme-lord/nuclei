package cache

import (
	"regexp"
	"testing"

	"github.com/Knetic/govaluate"
)

func TestRegexCache_SetGet(t *testing.T) {
	// ensure init
	c := Regex()
	pattern := "abc(\n)?123"
	re, err := regexp.Compile(pattern)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if err := c.Set(pattern, re); err != nil {
		t.Fatalf("set: %v", err)
	}
	got, err := c.GetIFPresent(pattern)
	if err != nil || got == nil {
		t.Fatalf("get: %v got=%v", err, got)
	}
	if got.String() != re.String() {
		t.Fatalf("mismatch: %s != %s", got.String(), re.String())
	}
}

func TestDSLCache_SetGet(t *testing.T) {
	c := DSL()
	expr := "1 + 2 == 3"
	ast, err := govaluate.NewEvaluableExpression(expr)
	if err != nil {
		t.Fatalf("dsl compile: %v", err)
	}
	if err := c.Set(expr, ast); err != nil {
		t.Fatalf("set: %v", err)
	}
	got, err := c.GetIFPresent(expr)
	if err != nil || got == nil {
		t.Fatalf("get: %v got=%v", err, got)
	}
	if got.String() != ast.String() {
		t.Fatalf("mismatch: %s != %s", got.String(), ast.String())
	}
}

func TestRegexCache_EvictionByCapacity(t *testing.T) {
	SetCapacities(3, 3)
	c := Regex()
	for i := 0; i < 5; i++ {
		k := string(rune('a' + i))
		re := regexp.MustCompile(k)
		_ = c.Set(k, re)
	}
	// last 3 keys expected to remain under LRU: 'c','d','e'
	if _, err := c.GetIFPresent("a"); err == nil {
		t.Fatalf("expected 'a' to be evicted")
	}
	if _, err := c.GetIFPresent("b"); err == nil {
		t.Fatalf("expected 'b' to be evicted")
	}
	if _, err := c.GetIFPresent("c"); err != nil {
		t.Fatalf("expected 'c' present")
	}
}

func TestSetCapacities_NoRebuildOnZero(t *testing.T) {
	// init
	SetCapacities(4, 4)
	c1 := Regex()
	_ = c1.Set("k", regexp.MustCompile("k"))
	if _, err := c1.GetIFPresent("k"); err != nil {
		t.Fatalf("expected key present: %v", err)
	}
	// zero changes should not rebuild/clear caches
	SetCapacities(0, 0)
	c2 := Regex()
	if _, err := c2.GetIFPresent("k"); err != nil {
		t.Fatalf("key lost after zero-capacity SetCapacities: %v", err)
	}
}

func TestSetCapacities_BeforeFirstUse(t *testing.T) {
	// This should not be overridden by later initCaches
	SetCapacities(2, 0)
	c := Regex()
	_ = c.Set("a", regexp.MustCompile("a"))
	_ = c.Set("b", regexp.MustCompile("b"))
	_ = c.Set("c", regexp.MustCompile("c"))
	if _, err := c.GetIFPresent("a"); err == nil {
		t.Fatalf("expected 'a' to be evicted under cap=2")
	}
}

func TestSetCapacities_ConcurrentAccess(t *testing.T) {
	SetCapacities(64, 64)
	stop := make(chan struct{})

	go func() {
		for i := 0; i < 5000; i++ {
			_ = Regex().Set("k"+string(rune('a'+(i%26))), regexp.MustCompile("a"))
			_, _ = Regex().GetIFPresent("k" + string(rune('a'+(i%26))))
			_, _ = DSL().GetIFPresent("1+2==3")
		}
		close(stop)
	}()

	for i := 0; i < 200; i++ {
		SetCapacities(64+(i%5), 64+((i+1)%5))
	}
	<-stop
}
