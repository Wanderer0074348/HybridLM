package utils

import "time"

const slmCooldown = 1800 * time.Millisecond

var slmGate = make(chan struct{}, 1)

func AcquireSLM() {
	slmGate <- struct{}{}
}

func ReleaseSLM() {
	time.Sleep(slmCooldown)
	<-slmGate
}
