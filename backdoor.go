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
	update(core.ProcBackdoor(line))
}

func (c *Core) ProcBackdoor(line string) Update {
	var upd Update
	data := []byte(line)
	shouldf(json.Unmarshal(data, c), "unmarshal backdoor command into Core")
	shouldf(json.Unmarshal(data, &upd), "unmarshal backdoor command into Update")
	if c.Status == "Ready to sync" {
		c.Sync = c.doSync
	}
	return upd
}
