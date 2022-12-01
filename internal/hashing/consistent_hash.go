package hashing

import (
	"errors"
	"fmt"
	"hash/maphash"
)

type ringNode struct {
	hash      uint64
	valueHash uint64
}

type ConsistentHash[T any] struct {
	ring []ringNode

	values map[uint64]T

	// A function that turns a value into a string representation
	stringifier func(T) string

	seed maphash.Seed
}

func NewConsistentHash[T any](stringifier func(T) string) ConsistentHash[T] {
	return ConsistentHash[T]{
		values:      make(map[uint64]T),
		seed:        maphash.MakeSeed(),
		stringifier: stringifier,
	}
}

// Returns an index into the ring of the node with the smallest hash greater than the given hash.
//
// Returns len(ring) if hash is greater than all current nodes.
func (c *ConsistentHash[T]) findRingIndexGreaterThan(hash uint64) int {

	if len(c.ring) == 0 {
		return 0
	}

	if hash < c.ring[0].hash {
		return 0
	}
	if hash > c.ring[len(c.ring)-1].hash {
		return len(c.ring)
	}

	// binary search
	low := 0
	high := len(c.ring) - 1

	for low <= high {
		mid := (low + high) / 2

		if c.ring[mid].hash < hash {
			low = mid + 1
		} else {
			high = mid - 1
		}
	}

	return low
}

// Creates a new node to hold the given value, and inserts replicas into the ring.
func (c *ConsistentHash[T]) Add(value T, replicas int) {
	// calculate string and hashes for value, insert to map
	valueString := c.stringifier(value)
	valueHash := maphash.String(c.seed, valueString)

	c.values[valueHash] = value

	// insert virtual nodes into ring
	for i := 0; i < replicas; i++ {
		node := ringNode{
			hash:      maphash.String(c.seed, valueString+fmt.Sprint(i)),
			valueHash: valueHash,
		}

		c.insertNode(node)
	}
}

// Inserts a node into the ring.
func (c *ConsistentHash[T]) insertNode(node ringNode) {
	indexToInsert := c.findRingIndexGreaterThan(node.hash)

	newRing := make([]ringNode, 0, len(c.ring)+1)

	if indexToInsert != 0 {
		newRing = append(newRing, c.ring[0:indexToInsert]...)
	}
	newRing = append(newRing, node)
	if indexToInsert != len(c.ring) {
		newRing = append(newRing, c.ring[indexToInsert:]...)
	}

	c.ring = newRing
}

func (c *ConsistentHash[T]) Remove(value T) {
	valueString := c.stringifier(value)
	valueHash := maphash.String(c.seed, valueString)

	// walk through ring, removing all vnodes
	toRemove := make([]int, 0, len(c.ring))
	for i := 0; i < len(c.ring); i++ {
		if c.ring[i].valueHash == valueHash {
			toRemove = append(toRemove, i)
		}
	}

	newRing := make([]ringNode, 0, len(c.ring)-len(toRemove))
	if toRemove[0] != 0 {
		newRing = append(newRing, c.ring[0:toRemove[0]]...)
	}
	for r := 1; r < len(toRemove); r++ {
		if toRemove[r-1] != toRemove[r]-1 {
			newRing = append(newRing, c.ring[toRemove[r-1]+1:toRemove[r]]...)
		}
	}
	if toRemove[len(toRemove)-1] != len(c.ring)-1 {
		newRing = append(newRing, c.ring[toRemove[len(toRemove)-1]+1:]...)
	}

	c.ring = newRing
}

// Lookup  a hash in the ring.
//
// Panics if the ring is empty
// TODO: find an alternative to panicking here
func (c *ConsistentHash[T]) RingLookup(i uint64) T {
	if len(c.ring) == 0 {
		panic(errors.New("Empty ring."))
	}

	ringIndex := c.findRingIndexGreaterThan(i)

	if ringIndex == len(c.ring) {
		ringIndex = 0
	}

	return c.values[c.ring[ringIndex].valueHash]
}

///// TODO: could make insertNode into bulk operation inserting lists of nodes at once
