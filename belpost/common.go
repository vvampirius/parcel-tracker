package belpost

import (
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().Unix())
}

func RequestInterval(lastRequestAt time.Time) {
	time.Sleep(time.Duration(28 + rand.Intn(4)) * time.Second)
}