package fixture

import "testing"

func TestResetableStruct_Reset(t *testing.T) {
	s := "hello"
	child := &ResetableStruct{i: 99}

	r := &ResetableStruct{
		i:     1,
		str:   "hi",
		strP:  &s,
		s:     []int{1, 2, 3},
		m:     map[string]string{"a": "b"},
		child: child,
	}

	r.Reset()

	if r.i != 0 {
		t.Errorf("i = %d, want 0", r.i)
	}
	if r.str != "" {
		t.Errorf("str = %q, want empty", r.str)
	}
	if s != "" {
		t.Errorf("*strP = %q, want empty (pointee must be zeroed, pointer itself kept)", s)
	}
	if r.strP == nil {
		t.Error("strP must remain non-nil")
	}
	if len(r.s) != 0 {
		t.Errorf("len(s) = %d, want 0", len(r.s))
	}
	if cap(r.s) == 0 {
		t.Error("s must keep its capacity (truncated, not reallocated)")
	}
	if len(r.m) != 0 {
		t.Errorf("len(m) = %d, want 0", len(r.m))
	}
	if child.i != 0 {
		t.Errorf("child.i = %d, want 0 (nested Reset must have been called)", child.i)
	}
	if r.child == nil {
		t.Error("child pointer itself must be kept, only its contents reset")
	}
}

func TestResetableStruct_Reset_NilReceiver(t *testing.T) {
	var r *ResetableStruct
	r.Reset() // must not panic
}

func TestWithEmbedded_Reset(t *testing.T) {
	w := &WithEmbedded{
		ResetableStruct: ResetableStruct{i: 5},
		Extra:           true,
	}

	w.Reset()

	if w.i != 0 {
		t.Errorf("embedded i = %d, want 0", w.i)
	}
	if w.Extra {
		t.Error("Extra = true, want false")
	}
}
