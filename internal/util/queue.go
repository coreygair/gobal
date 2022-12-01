package util

import "errors"

type Queue[T any] interface {
	Enqueue(item T)
	Dequeue() T
	Count() int
}

type ringBufferQueue[T any] struct {
	Queue[T]

	buf   []T
	hd    int
	tl    int
	count int
}

func NewRingBufferQueue[T any](size int) *ringBufferQueue[T] {
	return &ringBufferQueue[T]{
		buf: make([]T, size),
	}
}
func (q *ringBufferQueue[T]) Enqueue(item T) {
	if q.count == len(q.buf) {
		panic(errors.New("Enqueue: out of space."))
	}

	q.buf[q.hd] = item

	q.count++
	q.hd = (q.hd + 1) % len(q.buf)
}
func (q *ringBufferQueue[T]) Dequeue() T {
	if q.count == 0 {
		panic(errors.New("Dequeue: no elements."))
	}

	defer func() {
		q.count--
		q.tl = (q.tl + 1) % len(q.buf)
	}()

	return q.buf[q.tl]
}
func (q *ringBufferQueue[T]) Count() int {
	return q.count
}
func (q *ringBufferQueue[T]) GetBuffer() *[]T {
	return &q.buf
}
