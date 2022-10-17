package array

type RollingArray[V any] struct {
	Capacity int
	size     int
	zeroPos  int
	values   []V
}

func NewRolling[V any](cap int) *RollingArray[V] {
	return &RollingArray[V]{
		Capacity: cap,
		size:     0,
		values:   make([]V, cap),
		zeroPos:  0,
	}
}

func (o *RollingArray[V]) Get(index int) V {
	if index > (o.size - 1) {
		panic("[orderedArray] - index out of bounds")
	}
	if o.zeroPos+index >= o.Capacity {
		index = index - o.Capacity
	}
	return o.values[o.zeroPos+index]
}

func (o *RollingArray[V]) Size() int {
	return o.size
}

func (o *RollingArray[V]) Append(values ...V) {
	if len(values) > o.Capacity {
		panic("[rollingArray] - cannot append more values than capacity")
	}

	if o.size < o.Capacity {
		if len(values) <= (o.Capacity - o.size) {
			for i, v := range values {
				o.values[o.size+i] = v
			}
			o.size += len(values)
		} else {
			remaining := o.Capacity - o.size
			for i, v := range values {
				if i < remaining {
					o.values[o.size+i] = v
				} else {
					o.values[o.zeroPos+i-remaining] = v
				}
			}
			o.zeroPos = len(values) - remaining
			o.size += remaining
		}
	} else {
		if len(values) < (o.Capacity - o.zeroPos) {
			for i, v := range values {
				o.values[o.zeroPos+i] = v
				o.zeroPos += len(values)
			}
		} else {
			remaining := o.Capacity - o.zeroPos
			for i, v := range values {
				if i < remaining {
					o.values[o.zeroPos+i] = v
				} else {
					o.values[i-remaining] = v
				}
			}
			o.zeroPos = len(values) - remaining
		}
	}

}
