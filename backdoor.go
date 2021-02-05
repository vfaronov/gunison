package main

import (
	"bufio"
	"encoding/json"
	"os"
)

func init() {
	go watchBackdoor()
}

func watchBackdoor() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		shouldIdleAdd(recvBackdoor, scanner.Text())
	}
	shouldf(scanner.Err(), "scan backdoor")
}

func recvBackdoor(line string) {
	update(engine.ProcBackdoor(line))
}

func (e *Engine) ProcBackdoor(line string) Update {
	var upd Update
	data := []byte(line)
	shouldf(json.Unmarshal(data, e), "unmarshal backdoor command into Engine")
	shouldf(json.Unmarshal(data, &upd), "unmarshal backdoor command into Update")
	if e.Status == "Ready to sync" {
		e.Sync = e.doSync
	}
	return upd
}
