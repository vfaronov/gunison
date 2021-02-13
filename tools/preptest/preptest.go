package main

import (
	"bytes"
	"crypto/rand"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	flag.Usage = func() {
		fmt.Fprint(flag.CommandLine.Output(), "Usage: preptest [flags] profile left right\n"+
			"Create unison profile rooted in the left and right directories,\n"+
			"and fill left and right with example replicas/changes for testing.\n",
		)
		flag.PrintDefaults()
	}
	var (
		tplName   string
		extra     string
		rootLeft  string
		rootRight string
		err       error
	)
	flag.StringVar(&tplName, "template", "assorted",
		"`name` of the template according to which left and right will be filled")
	flag.StringVar(&extra, "extra", "",
		"extra config text to put into the profile.prf file")
	flag.StringVar(&rootLeft, "root-left", "",
		"use `root` instead of left as the Unison root")
	flag.StringVar(&rootRight, "root-right", "",
		"use `root` instead of right as the Unison root")
	flag.Parse()
	tpl, ok := templates[tplName]
	if !ok {
		panic("unknown template: " + tplName)
	}
	profile := flag.Arg(0)
	left := flag.Arg(1)
	right := flag.Arg(2)
	if profile == "" || left == "" || right == "" {
		flag.Usage()
		os.Exit(1)
	}
	left, err = filepath.Abs(left)
	must(err)
	right, err = filepath.Abs(right)
	must(err)
	if rootLeft == "" {
		rootLeft = left
	}
	if rootRight == "" {
		rootRight = right
	}

	log.Println("preparing .unison directory")
	prepareUnison(profile, rootLeft, rootRight, extra)

	log.Println("filling initial tree in", left)
	rm(left)
	tpl.create(left)

	log.Println("batch-syncing initial tree to", rootRight)
	rm(right)
	mkdir(right)
	batchSync(profile)

	if tpl.changeLeft != nil {
		log.Println("applying changes to", left)
		tpl.changeLeft(left)
	}

	if tpl.changeRight != nil {
		log.Println("applying changes to", right)
		tpl.changeRight(right)
	}
}

func prepareUnison(profile, root1, root2, extra string) {
	dir := os.Getenv("UNISON")
	if dir == "" {
		panic("please set the UNISON environment variable to contain path to the .unison directory " +
			"(which will be erased and recreated)")
	}
	rm(dir)
	put(p(dir, profile+".prf"),
		[]byte(fmt.Sprintf("root = %s\nroot = %s\nlogfile = %s\nwatch = false\n%s\n",
			root1, root2, p(dir, "log"), extra)))
}

func batchSync(profile string) {
	_, err := exec.Command("unison", profile, "-batch").Output()
	must(err)
}

type template struct {
	create      func(root string)
	changeLeft  func(root string)
	changeRight func(root string)
}

var templates = map[string]template{
	"assorted": {
		create: func(root string) {
			put(p(root, "one"), empty)
			put(p(root, "two"), smallText)
			put(p(root, "three"), largeText)
			put(p(root, "four"), smallBinary)
			put(p(root, "five"), largeBinary)
			put(p(root, "six", "seven"), empty)
			put(p(root, "six", "eight"), smallText)
			put(p(root, "six", "nine"), largeText)
			put(p(root, "six", "ten"), smallBinary)
			put(p(root, "six", "eleven"), largeBinary)
			mkdir(p(root, "twelve"))
			mkdir(p(root, "six", "thirteen"))
			put(p(root, "six", "fourteen", "fifteen"), smallText)
			put(p(root, "six", "fourteen", "sixteen"), smallText)
			put(p(root, funnyName, funnyName), smallText)
			symlink(p(root, "six", "funny name!"), p("..", funnyName))
			put(p(root, "one hundred", "one hundred one"), smallText)
			put(p(root, "one hundred", "one hundred two", "one hundred three"), smallText)
			put(p(root, "one hundred", "one hundred two", "one hundred four"), smallText)
			put(p(root, "twenty one"), smallText)
			put(p(root, "deeply", "nested", "sub", "directory", "with", "file"), smallText)
		},
		changeLeft: func(root string) {
			put(p(root, "seventeen"), empty)
			tweak(p(root, "two"), 1)
			tweak(p(root, "six", "nine"), 1000)
			chmod(p(root, "six", "ten"))
			symlink(p(root, "eighteen"), p("six", "eleven"))
			mkdir(p(root, "nineteen"))
			mv(p(root, "one hundred", "one hundred one"), p(root, "one hundred", "one hundred two"))
			put(p(root, "one hundred", "one hundred one"), smallBinary)
			for i := 0; i < 30; i++ {
				put(p(root, "twelve", fmt.Sprintf("small file %02d", i)), smallText)
			}
			tweak(p(root, "twenty one"), 100)
		},
		changeRight: func(root string) {
			rm(p(root, "three"))
			rm(p(root, "six", "fourteen"))
			put(p(root, "six", "seven"), smallText)
			put(p(root, "six", "eight"), largeText)
			tweak(p(root, "six", "nine"), 2000)
			symlink(p(root, "six", "funny name!"), p("..", funnyName, funnyName))
			mkdir(p(root, "nineteen"))
			mkdir(p(root, "twenty"))
			chmod(p(root, "twelve"))
			tweak(p(root, "six", "eleven"), 2000)
			mv(p(root, "one hundred", "one hundred two"), p(root, "one hundred", "one hundred one"))
			tweak(p(root, funnyName, funnyName), 200)
			tweak(p(root, "twenty one"), 100)
			tweak(p(root, "deeply", "nested", "sub", "directory", "with", "file"), 200)
		},
	},

	"minimal": {
		create: func(root string) {
			put(p(root, "one"), smallText)
		},
		changeLeft: func(root string) {
			tweak(p(root, "one"), 100)
		},
	},

	"several": {
		create: func(root string) {
			for i := 1; i <= 3; i++ {
				put(p(root, fmt.Sprintf("file%d", i)), smallText)
			}
		},
		changeLeft: func(root string) {
			for i := 1; i <= 3; i++ {
				tweak(p(root, fmt.Sprintf("file%d", i)), 50*i)
			}
		},
	},

	"unchanged": {
		create: func(root string) {
			put(p(root, "one"), smallText)
		},
	},

	"empty": {
		create: func(root string) {},
	},

	"missing-right": {
		create: func(root string) {
			put(p(root, "one"), smallText)
		},
		changeRight: func(root string) {
			rm(root)
		},
	},
}

var (
	empty     = []byte{}
	smallText = []byte(`Quia est unde laboriosam. Eum ullam deleniti dolores. Magni quasi facere voluptas. Dolor doloribus aut ut sed officiis id. Et aut nostrum est quia corrupti maiores optio.

Consectetur fuga sed vitae et nihil quia. Eveniet rerum officia repudiandae tenetur molestiae. Magni ipsum et natus accusantium ut consequatur neque. Veniam in voluptate quia. Culpa labore distinctio laudantium maxime voluptate eaque.

Deserunt dignissimos corrupti aut vel. Laboriosam at labore omnis eos et minus porro perspiciatis. Veniam in dignissimos voluptatem exercitationem excepturi reprehenderit sed optio.

Omnis repudiandae nobis autem qui autem possimus. Dolorem id a reprehenderit nihil laboriosam non. Dolor minima in soluta. Magni eveniet magnam velit officia consectetur tempore quia id. Perspiciatis enim corrupti aliquam nam accusamus et molestiae rerum. Sint sit exercitationem corrupti omnis.

Facere asperiores unde rerum dignissimos id. Nihil maiores sequi accusamus eum repudiandae et. Nesciunt ab inventore repellat enim illum ratione enim voluptatum. Tempore sint quos tempore fugit rerum sit omnis quae. Minus deserunt aut dolores excepturi qui.
`)
	largeText   = bytes.Repeat(append(smallText, '\n'), 1000)
	smallBinary = make([]byte, 1000)
	largeBinary = make([]byte, 10000000)
)

const funnyName = `here is a rather long and funny file name, 社會科學院語學研究所	              ​    　ﾟ･✿ヾ╲(｡◕‿◕｡)╱✿･ﾟ`

func init() {
	_, _ = rand.Read(smallBinary)
	_, _ = rand.Read(largeBinary)
}

func p(elem ...string) string {
	return filepath.Join(elem...)
}

func rm(path string) {
	must(os.RemoveAll(path))
}

func put(path string, data []byte) {
	rm(path)
	mkdir(p(path, ".."))
	must(ioutil.WriteFile(path, data, 0644))
}

func symlink(path, target string) {
	rm(path)
	mkdir(p(path, ".."))
	must(os.Symlink(target, path))
}

func mkdir(path string) {
	must(os.MkdirAll(path, 0755))
}

func chmod(path string) {
	must(os.Chmod(path, 0700))
}

func tweak(path string, pos int) {
	data, err := ioutil.ReadFile(path)
	must(err)
	b := make([]byte, pos)
	copy(b, data[pos:2*pos])
	copy(data[pos:2*pos], data[3*pos:4*pos])
	copy(data[3*pos:4*pos], b)
	put(path, data)
}

func mv(path, target string) {
	must(os.RemoveAll(target))
	must(os.Rename(path, target))
}

func must(err error) {
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			_, _ = os.Stderr.Write(exitErr.Stderr)
		}
		panic(err)
	}
}
