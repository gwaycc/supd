package filebak

import (
	"testing"
)

//
func TestFile(t *testing.T) {
	f, err := OpenFile("./test.log", 1*SIZE_K, 10)
	if err != nil {
		t.Fatal(err)
	}
	for i := 10000; i > 0; i-- {
		if _, err := f.WriteString("Hello"); err != nil {
			t.Fatal(err)
		}
	}
	f.Close()
}
