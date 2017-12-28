package lru

import (
	"bytes"
	"reflect"
	"testing"
)

func TestCacheValueList(t *testing.T) {
	var l cacheValueList

	check := func(want ...string) {
		t.Helper()
		var got []string
		var prev *cacheValue
		for v := l.front; v != nil; v = v.next {
			if v.prev != prev {
				t.Fatalf("bad prev pointer: got %p; want %p", v.prev, prev)
			}
			if string(v.val) != v.key+v.key {
				t.Fatalf("mismatched value: key=%s; bytes=%s", v.key, v.val)
			}
			got = append(got, v.key)
			prev = v
		}
		if l.back != prev {
			t.Fatalf("bad back pointer: got %p; want %p", l.back, prev)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("wrong list contents: got %q; want %q", got, want)
		}
	}

	a := &cacheValue{key: "a", val: []byte("aa")}
	b := &cacheValue{key: "b", val: []byte("bb")}
	c := &cacheValue{key: "c", val: []byte("cc")}

	check()
	l.pushFront(a)
	check("a")
	l.moveToFront(a)
	check("a")
	l.pushFront(b)
	check("b", "a")
	l.moveToFront(b)
	check("b", "a")
	l.moveToFront(a)
	check("a", "b")
	l.pushFront(c)
	check("c", "a", "b")
	l.moveToFront(c)
	check("c", "a", "b")
	l.moveToFront(b)
	check("b", "c", "a")
	l.moveToFront(c)
	check("c", "b", "a")

	l.delete(c)
	check("b", "a")
	l.pushFront(c)
	check("c", "b", "a")
	l.delete(b)
	check("c", "a")
	l.pushFront(b)
	check("b", "c", "a")
	l.delete(a)
	check("b", "c")
	l.delete(b)
	check("c")
	l.pushFront(b)
	check("b", "c")
	l.delete(c)
	check("b")
	l.delete(b)
	check()
}

func TestCache(t *testing.T) {
	c := New(10)

	exists := func(key string, val []byte) {
		t.Helper()
		got, ok := c.Get(key)
		if !ok || !bytes.Equal(got, val) {
			t.Errorf("Get(%q) gave %q, %t; want %q, true", key, got, ok, val)
		}
	}
	notExists := func(key string) {
		t.Helper()
		if _, ok := c.Get(key); ok {
			t.Errorf("Get(%q) gave ok=true; want ok=false", key)
		}
	}

	key4, val4 := "4", []byte("4__")
	key5, val5 := "5", []byte("5___")
	key6, val6 := "6", []byte("6____")

	notExists(key4)

	c.Insert(key4, val4)
	exists(key4, val4)

	c.Insert(key4, nil)
	exists(key4, val4)

	c.Insert(key5, make([]byte, 10))
	notExists(key5)

	c.Insert(key5, val5)
	exists(key4, val4)
	exists(key5, val5)

	c.Insert(key6, val6)
	exists(key6, val6)
	notExists(key4)
	notExists(key5)

	c.Insert(key4, val4)
	exists(key6, val6)
	exists(key4, val4)

	c.Insert(key5, val5)
	notExists(key6)
	exists(key4, val4)
	exists(key5, val5)

	c.Delete(key4)
	notExists(key4)
	c.Delete(key4)
}
