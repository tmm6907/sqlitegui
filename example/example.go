package main

import (
	"fmt"
	"strings"
)

type Set struct {
	_set map[any]struct{}
}

func newSet(args ...any) Set {
	s := Set{make(map[any]struct{})}
	for _, v := range args {
		s._set[v] = struct{}{}
	}
	return s
}

func (s *Set) Add(v any) {
	s._set[v] = struct{}{}
}
func (s *Set) Contains(v any) bool {
	_, ok := s._set[v]
	return ok
}

func (s *Set) Remove(v any) {
	delete(s._set, v)
}

func (s Set) String() string {
	items := make([]string, 0, len(s._set))
	for item := range s._set {
		items = append(items, fmt.Sprint(item))
	}
	return strings.Join(items, ", ")
}

func Map[T, V any](s []T, transform func(t T) V) []V {
	res := make([]V, len(s))
	for i, v := range s {
		res[i] = transform(v)
	}
	return res
}

func Reduce[T, V any](s []T, initial V, reduce func(acc V, curr T) V) V {
	res := initial
	for _, v := range s {
		res = reduce(res, v)
	}
	return res
}
func main() {
	s := newSet(false, "apples", 32, 5, struct{ name string }{"thomas"})
	fmt.Println("Set: ", s)
	s.Add("angel")
	fmt.Println("Set: ", s)
	s.Add([3]string{"ham", "chicken", "turkey"})
	fmt.Println("Set: ", s)
	fmt.Println(s.Contains(32))
	fmt.Println(s.Contains("apple sauce"))
	s.Remove(5)
	s.Remove("carrots")
	fmt.Println("Set: ", s)

	data := []any{1, []string{"stuff"}, 6, 342, 6}
	doubled := Map(data, func(x any) string {
		return fmt.Sprintf("angler : %v", x)
	})
	fmt.Println(doubled)
}
