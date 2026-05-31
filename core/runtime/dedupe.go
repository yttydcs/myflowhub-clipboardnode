package runtime

type boundedStringSet struct {
	limit int
	set   map[string]struct{}
	order []string
}

func newBoundedStringSet(limit int) *boundedStringSet {
	if limit <= 0 {
		limit = 1
	}
	return &boundedStringSet{
		limit: limit,
		set:   make(map[string]struct{}),
	}
}

func (s *boundedStringSet) Add(value string) {
	if value == "" {
		return
	}
	if _, ok := s.set[value]; ok {
		return
	}
	s.set[value] = struct{}{}
	s.order = append(s.order, value)
	s.trim()
}

func (s *boundedStringSet) Contains(value string) bool {
	_, ok := s.set[value]
	return ok
}

func (s *boundedStringSet) Consume(value string) bool {
	if !s.Contains(value) {
		return false
	}
	delete(s.set, value)
	return true
}

func (s *boundedStringSet) trim() {
	for len(s.set) > s.limit && len(s.order) > 0 {
		oldest := s.order[0]
		s.order = s.order[1:]
		delete(s.set, oldest)
	}
	if len(s.order) > s.limit*2 {
		kept := s.order[:0]
		for _, value := range s.order {
			if _, ok := s.set[value]; ok {
				kept = append(kept, value)
			}
		}
		s.order = kept
	}
}
