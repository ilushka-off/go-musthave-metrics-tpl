package pool

import "testing"

type item struct {
	n    int
	tags []string
}

func (i *item) Reset() {
	if i == nil {
		return
	}
	i.n = 0
	i.tags = i.tags[:0]
}

func TestPool_PutResetsObject(t *testing.T) {
	p := New(func() *item { return &item{} })

	x := p.Get()
	x.n = 42
	x.tags = append(x.tags, "a", "b")
	p.Put(x)

	// Put must have reset x regardless of whether the pool hands it back.
	if x.n != 0 || len(x.tags) != 0 {
		t.Fatalf("object not reset: %+v", x)
	}
	if cap(x.tags) < 2 {
		t.Fatalf("slice capacity should be kept, got %d", cap(x.tags))
	}
}

func TestPool_GetCreatesNew(t *testing.T) {
	calls := 0
	p := New(func() *item { calls++; return &item{} })

	if p.Get() == nil {
		t.Fatal("Get returned nil")
	}
	if calls != 1 {
		t.Fatalf("expected constructor to be called once, got %d", calls)
	}
}

func TestPool_NilNewFn(t *testing.T) {
	p := New[*item](nil)

	if x := p.Get(); x != nil {
		t.Fatalf("expected zero value from empty pool, got %+v", x)
	}
}
