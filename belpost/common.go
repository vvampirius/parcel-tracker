package belpost

import (
	"log"
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().Unix())
}

func RequestInterval(lastRequestAt time.Time, debugLog *log.Logger) {
	difference := time.Now().Sub(lastRequestAt)
	if difference >= 28 * time.Second { return }
	sleepTime :=time.Duration(28 - int(difference.Seconds()) + rand.Intn(4)) * time.Second
	if debugLog != nil { debugLog.Println(`Wait`, sleepTime) }
	time.Sleep(sleepTime)
}