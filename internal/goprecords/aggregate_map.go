package goprecords

func getOrNewAggregate(m map[string]*Aggregate, name string) *Aggregate {
	if a, ok := m[name]; ok {
		return a
	}
	a := NewAggregate(name)
	m[name] = a
	return a
}
