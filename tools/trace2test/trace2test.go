package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	fmt.Println("c := NewCore()")
	fmt.Println("var upd Update")
	fmt.Println("upd = c.ProcStart()")
	fmt.Println("assert.Zero(t, upd)")
	scanner := bufio.NewScanner(os.Stdin)
	var output string
	for scanner.Scan() {
		var (
			s               string
			out, in         string
			interrupt, kill bool
			exit            bool
			code            int
		)
		line := scanner.Text()
		if _, err := fmt.Sscanf(line, "write(1, %q", &s); err == nil {
			out = s
		} else if _, err := fmt.Sscanf(line, "write(2, %q", &s); err == nil {
			out = s
		} else if _, err := fmt.Sscanf(line, "read(0, %q", &s); err == nil {
			in = s
		} else if _, err := fmt.Sscanf(line, "--- SIGINT"); err == nil {
			interrupt = true
		} else if _, err := fmt.Sscanf(line, "--- SIGKILL"); err == nil {
			kill = true
		} else if _, err := fmt.Sscanf(line, "exit_group(%d)", &code); err == nil {
			exit = true
		} else {
			continue
		}

		if out == "" && output != "" {
			fmt.Printf("upd = c.ProcOutput([]byte(%q))\n", output)
			output = ""
		}

		switch {
		case out != "":
			if output != "" && len(out) >= 5 {
				fmt.Printf("upd = c.ProcOutput([]byte(%q))\n", output)
				fmt.Print("assert.Zero(t, upd)\n")
				output = ""
			}
			output += out

		case in != "":
			fmt.Printf("assert.Equal(t, Update{Input: []byte(%q)}, upd)\n", in)

		case interrupt:
			fmt.Print("assert.Equal(t, Update{Interrupt: true}, upd)\n")

		case kill:
			fmt.Print("assert.Equal(t, Update{Kill: true}, upd)\n")

		case exit:
			fmt.Printf("upd = c.ProcExit(%d, nil)\n", code)
			fmt.Print("assert.Zero(t, upd)\n")
		}
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}

	if output != "" {
		fmt.Printf("upd = c.ProcOutput([]byte(%q))\n", output)
		fmt.Print("assert.Zero(t, upd)\n")
	}
}
