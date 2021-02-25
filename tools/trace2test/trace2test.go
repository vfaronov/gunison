// TODO: Doesn't support `strace -f`, making it cumbersome to deal with output from diff/merge commands.
package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	fmt.Println("c := NewCore()")
	fmt.Println("assert.Zero(t, c.ProcStart())")
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
			fmt.Printf("assert.Zero(t, c.ProcOutput([]byte(%q)))\n", output)
			output = ""
		}

		switch {
		case out != "":
			if output != "" && len(out) >= 5 {
				fmt.Printf("assert.Zero(t, c.ProcOutput([]byte(%q)))\n", output)
				output = ""
			}
			output += out

		case in != "":
			fmt.Printf("assertEqual(t, ?,\n\tUpdate{Input: []byte(%q)})\n", in)

		case interrupt:
			fmt.Print("assertEqual(t, ?,\n\tUpdate{Interrupt: true})\n")

		case kill:
			fmt.Print("assertEqual(t, ?,\n\tUpdate{Kill: true})\n")

		case exit:
			fmt.Printf("assert.Zero(t, c.ProcExit(%d, nil))\n", code)
		}
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}

	if output != "" {
		fmt.Printf("assert.Zero(c.ProcOutput([]byte(%q)))\n", output)
	}
}
