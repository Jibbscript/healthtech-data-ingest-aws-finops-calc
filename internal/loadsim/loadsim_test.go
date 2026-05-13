package loadsim

import "testing"

func TestPayloadDistributionIsPositive(t *testing.T) {
	for i := int64(0); i < 100; i++ {
		if PayloadSize(i, 5*1024*1024) <= 0 {
			t.Fatal("non-positive size")
		}
	}
}
