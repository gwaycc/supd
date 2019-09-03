package filebak

import "testing"

func TestStrToSize(t *testing.T) {
	tests := []struct {
		in  string
		out int64
	}{
		{"1", 1},
		{"1KB", SIZE_K},
		{"1MB", SIZE_M},
		{"1.5MB", SIZE_M + 512*SIZE_K},
	}

	for i, c := range tests {
		output := StrToSize(c.in)
		if output != c.out {
			t.Fatalf("expect:%d,but:%d,index:%d", c.out, output, i)
		}
	}
}

func TestSizeToStr(t *testing.T) {
	tests := []struct {
		in  int64
		out string
	}{
		{1, "1"},
		{SIZE_K, "1.00KB"},
		{SIZE_M, "1.00MB"},
		{10 * SIZE_K, "10.00KB"},
		{10 * SIZE_M, "10.00MB"},
		{10*SIZE_K + 512, "10.50KB"},
		{10*SIZE_M + 512*SIZE_K, "10.50MB"},
	}

	for i, c := range tests {
		output := SizeToStr(c.in)
		if output != c.out {
			t.Fatalf("expect:%s,but:%s,index:%d", c.out, output, i)
		}
	}
}
