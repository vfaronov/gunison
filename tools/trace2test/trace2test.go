// TODO: Doesn't support `strace -f`, making it cumbersome to deal with output from diff/merge commands.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"math/rand"
	"os"
)

func main() {
	var random bool
	flag.BoolVar(&random, "random", false,
		"simulate chunks of output from Unison getting buffered randomly, "+
			"instead of stuffing each write() entirely into one ProcOutput()")
	flag.Parse()

	fmt.Println("c := NewCore()")
	fmt.Println("assert.Zero(t, c.ProcStart())")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(nil, 1024*1024)
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
			dumpOutput(output, random)
			output = ""
		}

		switch {
		case out != "":
			if output != "" && !random && len(out) >= 5 {
				dumpOutput(output, random)
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
		dumpOutput(output, random)
	}
}

func dumpOutput(output string, random bool) {
	for output != "" {
		chunk := output
		if random {
			if n := rand.Intn(300); n < len(output) {
				chunk = output[:n]
			}
		}
		fmt.Printf("assert.Zero(t, c.ProcOutput([]byte(%q)))\n", chunk)
		output = output[len(chunk):]
	}
}
