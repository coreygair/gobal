package util

import "testing"

// utility to produce an error if no panic
func assertPanic(t *testing.T, f func(), message string) {
	defer func() {
		if r := recover(); r == nil {
			t.Error(message)
		}
	}()
	f()
}

func TestCircularBufferQueue(t *testing.T) {
	q := NewRingBufferQueue[int](5)

	assertPanic(t, func() { q.Dequeue() }, "No panic when dequeue from empty.")

	q.Enqueue(1)
	x := q.Dequeue()
	if x != 1 {
		t.Errorf("Dequeue after enqueue failed: expected 1, got %d\n", x)
	}

	q.Enqueue(1)
	q.Enqueue(2)
	q.Enqueue(3)
	q.Enqueue(4)
	q.Enqueue(5)

	assertPanic(t, func() {
		q.Enqueue(6)
	}, "No panic when enquing when full.")

	var out [5]int
	for i := range out {
		out[i] = q.Dequeue()
	}
	if out != [5]int{1, 2, 3, 4, 5} {
		t.Errorf("Incorrect dequeue order: expected %v, got %v", [5]int{1, 2, 3, 4, 5}, out)
	}
}
