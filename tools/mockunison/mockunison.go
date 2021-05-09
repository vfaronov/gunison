// Program mockunison can be used to check Gunison on arbitrary items/plans
// without actually placing them on the filesystem (as with preptest).
//
//	go build -o unison mockunison.go
//	PATH=$PWD:$PATH gunison
//
package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"path"
	"strings"
	"time"
)

func main() {
	paths := genPaths(10000)

	fmt.Print("Looking for changes\n")
	spinner := []string{`|`, `/`, `-`, `\`}
	for i := 0; i < 300; i++ {
		if paths[i] == "" {
			continue
		}
		if i > 0 {
			fmt.Print("\r", strings.Repeat(" ", 2+len(paths[i-1])), "\r")
		}
		fmt.Print(spinner[i%len(spinner)], " ", paths[i])
		time.Sleep(30 * time.Millisecond)
	}

	i := 0
	fmt.Print("\nalpha          beta               ")
	printPrompt(paths, i)
	scanner := bufio.NewScanner(os.Stdin)
loop:
	for scanner.Scan() {
		switch scanner.Text() {
		case "l":
			printPlan(paths)
		case "0":
			i = 0
		case "n", "/", "<", ">", "m":
			i++
		case "y":
			runUpdates()
			break loop
		case "q":
			break loop
		default:
			fmt.Print("not implemented")
		}
		printPrompt(paths, i)
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
}

var actions = []string{
	"changed  ---->        ",
	"         <---- changed",
	"changed  <-M-> changed",
	"changed  <-?-> changed",
}

func printPrompt(paths []string, i int) {
	if i == len(paths) {
		fmt.Print("\nProceed with propagating updates? [] ")
		return
	}
	action := actions[rand.Intn(len(actions))]
	fmt.Print("\n", action, "    ", paths[i], "  [f] ")
}

func printPlan(paths []string) {
	for _, path := range paths {
		action := actions[rand.Intn(len(actions))]
		fmt.Print(action, "    ", path, "  \n")
		fmt.Print("alpha        : changed file       modified on 2021-02-07 at  1:50:31  size 1146      rw-r--r--\n")
		fmt.Print("beta         : unchanged file     modified on 2021-02-07 at  1:50:31  size 1146      rw-r--r--\n")
	}
}

func runUpdates() {
	fmt.Print("\nPropagating updates\n")
	for p := 1; p <= 100; p++ {
		if p > 1 {
			fmt.Print("\r               \r")
		}
		fmt.Printf("%3d%%  01:00 ETA", p)
		time.Sleep(100 * time.Millisecond)
	}
	fmt.Print("\nSynchronization complete at 00:00:00  (...)\n")
}

func genPaths(n int) []string {
	// XXX: This algorithm is copied from genItems (which see for comments).
	paths := make([]string, n)
	seen := make(map[string]bool)
	for i := 0; i < len(paths); i++ {
		if i > 0 {
			paths[i] = paths[i-1]
		}
		if rand.Intn(100) > 0 {
			maxchop := strings.Count(paths[i], "/") + 1
			for nchop := 1 + rand.Intn(maxchop); nchop > 0; nchop-- {
				paths[i] = path.Dir(paths[i])
			}
			if paths[i] == "." {
				paths[i] = ""
			}
		}
		if rand.Intn(100) > 0 {
			for ngrow := 1 + rand.Intn(5); ngrow > 0; ngrow-- {
				segment := string([]rune{rune('a' + rand.Intn(26))})
				if rand.Intn(2) == 1 {
					segment += string([]rune{rune('a' + rand.Intn(26))})
				}
				paths[i] = path.Join(paths[i], segment)
			}
		}
		for seen[paths[i]] {
			paths[i] += string([]rune{rune('0' + rand.Intn(10))})
		}
		seen[paths[i]] = true
	}
	return paths
}
