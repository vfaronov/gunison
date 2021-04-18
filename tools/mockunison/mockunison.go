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
)

func main() {
	fmt.Println("\nleft           right              ")
	printPrompt()
	scanner := bufio.NewScanner(os.Stdin)
loop:
	for scanner.Scan() {
		switch scanner.Text() {
		case "l":
			printPlan()
		case "q":
			break loop
		default:
			fmt.Println("not implemented")
		}
		printPrompt()
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
}

func printPrompt() {
	fmt.Print("changed  ---->            one  [f] ")
}

func printPlan() {
	// XXX: This algorithm for generating items is copied from genItems (which see for comments).
	seen := make(map[string]bool)
	var prevpath string
	for i := 0; i < 10000; i++ {
		newpath := prevpath
		if rand.Intn(100) > 0 {
			maxchop := strings.Count(newpath, "/") + 1
			for nchop := 1 + rand.Intn(maxchop); nchop > 0; nchop-- {
				newpath = path.Dir(newpath)
			}
			if newpath == "." {
				newpath = ""
			}
		}
		if rand.Intn(100) > 0 {
			for ngrow := 1 + rand.Intn(5); ngrow > 0; ngrow-- {
				segment := string([]rune{rune('a' + rand.Intn(26))})
				if rand.Intn(2) == 1 {
					segment += string([]rune{rune('a' + rand.Intn(26))})
				}
				newpath = path.Join(newpath, segment)
			}
		}
		for seen[newpath] {
			newpath += string([]rune{rune('0' + rand.Intn(10))})
		}
		seen[newpath] = true
		prevpath = newpath

		fmt.Printf("changed  ---->            %s  \n", newpath)
		fmt.Println("left         : changed file       modified on 2021-02-07 at  1:50:31  size 1146      rw-r--r--")
		fmt.Println("right        : unchanged file     modified on 2021-02-07 at  1:50:31  size 1146      rw-r--r--")
	}
}
