package storage

import (
	"errors"
	"os"
	"testing"
)

func TestDiskvOperations(t *testing.T) {
	st, err := NewDiskvStroage("testdata")
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Put([]byte("key1"), []byte("value1")); err != nil {
		t.Fatal(err)
	}
	if err := st.Put([]byte("key2"), []byte("value2")); err != nil {
		t.Fatal(err)
	}
	if v, _, err := st.Get([]byte("key1")); err != nil {
		t.Fatal(err)
	} else {
		if string(v) != "value1" {
			t.Fatal("value not match")
		}
	}
}

func TestDiskvOperations2(t *testing.T) {
	st, err := NewDiskvStroage("testdata")
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Put([]byte("key1()"), []byte("value1")); err == nil {
		t.Fatal(errors.New("put error expected"))
	}
}

func TestDiskvOperations3(t *testing.T) {
	st, err := NewDiskvStroage("testdata")
	if err != nil {
		t.Fatal(err)
	}
	dat, _, err := st.Get([]byte("not_exist"))
	t.Log(dat)
	t.Log(os.IsNotExist(err))
}
