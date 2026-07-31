package ringbuf

type RingBuffer[T any] struct {
	buffer []T
	pos    int
	cap    int
	len    int
}

// New returns a new RingBuffer instance
func New[T any](size int) *RingBuffer[T] {
	return &RingBuffer[T]{
		buffer: make([]T, size),
		cap:    size,
	}
}

// Push adds a value into the ring buffer
func (rb *RingBuffer[T]) Push(value T) {
	rb.buffer[rb.pos] = value
	rb.pos++
	if rb.pos >= rb.cap {
		rb.pos = 0
	}
	if rb.len < rb.cap {
		rb.len++
	}
}

// Length returns the ringbuffer length.  This length will start from zero if no items have been pushed
// and cap out at the ringbuffer capacity
func (rb *RingBuffer[T]) Length() int {
	return rb.len
}

// Item returns an item from a zero based offset from the current head.  In order to make this conducive
// to windowing functions, the offset is a negative number from the current head.  This allows the user
// to easily generate a window of past activity from the current "Push" point.
func (rb *RingBuffer[T]) Item(offset int) T {
	// subtract one to get the previous insertion point
	itemNum := rb.pos - 1 + offset
	if itemNum < 0 {
		itemNum = rb.cap + itemNum
	}

	return rb.buffer[itemNum]
}
