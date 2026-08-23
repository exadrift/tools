package tui

import "math"

type Orientation int

const (
	OrientationHorizontal Orientation = iota
	OrientationVertical
)

type Segment struct {
	portion  int
	child    Widget
	minChars int
	maxChars int
}

func WithSegmentOptionMinChars(minChars int) *Option {
	return &Option{
		optionType: SegmentOptionMinChars,
		data:       minChars,
	}
}

// NewSegment returns a new flexbox segment.  Portion follows the following general rule:
// A positive number indicates the size of the portion in terms of screen real estate within the parent box.  the
// sum of all portions creates a whole number representing the totality of the space in virtual units and each
// individual portion represents that fraction of the whole number.  Thus 3 portions of 1, 1, and 1, will mean
// 3 equally sized portions.  2, 1, and 1 will mean that the flex area will have 3 segments, the first will occupy
// twice the size of each of the subsequent two, and the subsequent two will be equal sized.
func NewSegment(portion int, child Widget, options ...*Option) *Segment {
	segment := &Segment{
		portion: portion,
		child:   child,
	}
	for _, option := range options {
		switch option.optionType {
		case SegmentOptionMinChars:
			segment.minChars = option.data.(int)
		case SegmentOptionMaxChars:
			segment.maxChars = option.data.(int)
		default:
			panic("unrecognized option")
		}
	}

	return segment
}

type FlexLayout struct {
	*Box
	children    []Widget
	orientation Orientation
	segments    []*Segment
	size        int
}

// NewFlexLayout returns a FlexLayout
func NewFlexLayout(orientation Orientation, segments ...*Segment) *FlexLayout {
	flexLayout := &FlexLayout{
		Box:         NewBox(),
		orientation: orientation,
	}

	flexLayout.segments = make([]*Segment, len(segments))
	copy(flexLayout.segments, segments)

	flexLayout.children = make([]Widget, len(segments))
	for i, seg := range flexLayout.segments {
		flexLayout.children[i] = seg.child
		flexLayout.size += seg.portion
	}

	return flexLayout
}

func (f *FlexLayout) CanHaveFocus() bool {
	return false
}

func (f *FlexLayout) GetChildren() []Widget {
	return f.children
}

func adjustSize(size int, minSize int, maxSize int) int {
	if minSize > 0 && size < minSize {
		size = minSize
	}

	if maxSize > 0 && size > maxSize {
		size = maxSize
	}

	return size
}

func (f *FlexLayout) SetDimensions(left int, top int, width int, height int) {
	f.Box.SetDimensions(left, top, width, height)
	initialTop := top
	initialLeft := left
	childLeft := left
	childTop := top
	for _, segment := range f.segments {
		portion := float64(segment.portion) / float64(f.size)
		var size int
		switch f.orientation {
		case OrientationHorizontal:
			size = int(math.Round(portion * float64(width)))
			size = adjustSize(size, segment.minChars, segment.maxChars)
			if childLeft+size-initialLeft > width {
				size = width - (childLeft - initialLeft)
			}
			segment.child.SetDimensions(childLeft, top, size, height)
			childLeft += size

		case OrientationVertical:
			size = int(math.Round(portion * float64(height)))
			size = adjustSize(size, segment.minChars, segment.maxChars)
			if childTop+size+initialTop > height {
				size = height - (childTop - initialTop)
			}
			segment.child.SetDimensions(left, childTop, width, size)
			childTop += size
		}
	}
}
