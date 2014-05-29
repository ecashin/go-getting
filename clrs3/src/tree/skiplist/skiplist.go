// I didn't read about this in CLRS but just heard it covered in an
// MIT intro to algorithms course I got from YouTube.  It's been a
// while since I did Skip Lists in grad school.

package skiplist

type CompareFn func(a, b interface{}) int

type node struct {
	links []*node
	val   interface{}
}

type SkipList struct {
	cmp  CompareFn
	head *node
}

func (s *SkipList) Insert(val interface{}) {
	n := s.head
	if n == nil {
		s.head = &node{nil, val}
		s.head.links = append(s.head.links, nil)
	}
}
