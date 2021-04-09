// Program trace2test converts traces of Unison runs into unit tests for Core.
// Unison's writes become calls to ProcOutput, Unison's reads become assertions on Update.Input,
// and so on. It is used in the following workflow:
//
// 1. Run Unison under strace:
//	strace -o /path/to/trace -s 1000000 unison ...
//
// 2. Type into Unison the same input that you expect Gunison to send.
//
// 3. Convert into code:
//	go run ./tools/trace2test </path/to/trace >>core_test.go
//
// 4. Edit the resulting code. It is far from ready, because trace2test doesn't know
// which events trigger which, what should Updates contain other than Input, etc.
//
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
		"simulate chunks of output from Unison getting buffered randomly,\n"+
			"instead of stuffing each write() entirely into one ProcOutput()")
	flag.Parse()

	fmt.Println("\nfunc Test???(t *testing.T) {")
	fmt.Println("\tc := NewCore()")
	fmt.Println("\tassert.Zero(t, c.ProcStart())")
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
			fmt.Printf("\tassertEqual(t, ?,\n\t\tUpdate{Input: []byte(%q)})\n", in)

		case interrupt:
			fmt.Print("\tassertEqual(t, ?,\n\t\tUpdate{Interrupt: true})\n")

		case kill:
			fmt.Print("\tassertEqual(t, ?,\n\t\tUpdate{Kill: true})\n")

		case exit:
			fmt.Printf("\tassert.Zero(t, c.ProcExit(%d, nil))\n", code)
		}
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}

	if output != "" {
		dumpOutput(output, random)
	}
	fmt.Println("}")
}

func dumpOutput(output string, random bool) {
	for output != "" {
		chunk := output
		if random {
			if n := rand.Intn(300); n < len(output) {
				chunk = output[:n]
			}
		}
		fmt.Printf("\tassert.Zero(t, c.ProcOutput([]byte(%q)))\n", chunk)
		output = output[len(chunk):]
	}
}
