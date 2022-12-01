package hashing

import (
	"fmt"
	"testing"
)

func TestConsistentHashAdd(t *testing.T) {
	ch := NewConsistentHash[int](func(i int) string {
		return fmt.Sprint(i)
	})

	ch.Add(10, 1)

	v := ch.RingLookup(100)
	if v != 10 {
		t.Errorf("Failed get after add: expected 10 got %d\n", v)
	}

	ch.Add(20, 2)

	v2 := ch.RingLookup(100)
	if v2 != 10 && v2 != 20 {
		t.Errorf("Failed get after add: expected 10 or 20 got %d\n", v)
	}

	if len(ch.ring) != 3 {
		t.Errorf("Failed ring len after remove: expected 3 got %d\n", len(ch.ring))
	}
}

func TestConsistentHashRemove(t *testing.T) {
	ch := NewConsistentHash[int](func(i int) string {
		return fmt.Sprint(i)
	})

	ch.Add(10, 1)
	ch.Add(20, 1)
	ch.Add(30, 2)

	ch.Remove(10)
	ch.Remove(30)

	v := ch.RingLookup(100)
	if v != 20 {
		t.Errorf("Failed get after remove: expected 20 got %d\n", v)
	}

	if len(ch.ring) != 1 {
		t.Errorf("Failed ring len after remove: expected 1 got %d\n", len(ch.ring))
	}
}
