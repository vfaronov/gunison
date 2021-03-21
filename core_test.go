// +build !coremock

package main

import (
	"errors"
	"flag"
	"io"
	"log"
	"os"
	"testing"
	"time"

	"github.com/gotk3/gotk3/gtk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var needGTK = false // depending on -tags

func TestMain(m *testing.M) {
	// Silence debug logging unless running under -v.
	// TODO: Instead, inject t.Log as logger into code under test,
	// so that it gets enabled magically for failing tests?
	flag.Parse()
	if !testing.Verbose() {
		log.SetOutput(io.Discard)
	}

	if needGTK {
		gtk.Init(nil)
	}

	os.Exit(m.Run())
}

func TestMinimal(t *testing.T) {
	c := NewCore()
	assertEqual(t, c.Status, "Starting Unison")
	assert.False(t, c.Running)
	assert.True(t, c.Busy)
	assert.Empty(t, c.Progress)
	assert.Empty(t, c.ProgressFraction)
	assert.Empty(t, c.Left)
	assert.Empty(t, c.Right)
	assert.Nil(t, c.Sync)
	assert.Nil(t, c.Quit)
	assert.Nil(t, c.Abort)
	assert.Nil(t, c.Interrupt)
	assert.Nil(t, c.Kill)

	assert.Zero(t, c.ProcStart())
	assert.True(t, c.Running)
	assert.Zero(t, c.ProcOutput([]byte("Unison 2.51.3 (ocaml 4.11.1): Contacting server...\n")))
	assertEqual(t, c.Status, "Contacting server")
	assert.Zero(t, c.ProcOutput([]byte("Looking for changes\n")))
	assertEqual(t, c.Status, "Looking for changes")
	assert.Zero(t, c.ProcOutput([]byte("Reconciling changes\n")))
	assertEqual(t, c.Status, "Reconciling changes")
	assert.Zero(t, c.ProcOutput([]byte("\nleft           right              \n")))
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            one  [f] ")),
		Update{Input: []byte("l\n")})
	assertEqual(t, c.Status, "Assembling plan")
	assertEqual(t, c.Left, "left")
	assertEqual(t, c.Right, "right")
	assert.Zero(t, c.ProcOutput([]byte("  ")))
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            one  \n")))
	assert.Zero(t, c.ProcOutput([]byte("left         : changed file       modified on 2021-02-08 at 18:30:50  size 1146      rw-r--r--\nright        : unchanged file     modified on 2021-02-08 at 18:30:50  size 1146      rw-r--r--\n")))
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            one  [f] ")))
	assertEqual(t, c.Status, "Ready to synchronize")
	assert.False(t, c.Busy)
	assertEqual(t, c.Items, []Item{
		{
			Path: "one",
			Left: Content{File, Modified, "modified on 2021-02-08 at 18:30:50  size 1146      rw-r--r--",
				time.Date(2021, 2, 8, 18, 30, 50, 0, time.Local), 1146},
			Right: Content{File, Unchanged, "modified on 2021-02-08 at 18:30:50  size 1146      rw-r--r--",
				time.Date(2021, 2, 8, 18, 30, 50, 0, time.Local), 1146},
			Action: LeftToRight,
		},
	})
	require.NotNil(t, c.Sync)
	assert.NotNil(t, c.Quit)

	assertEqual(t, c.Sync(),
		Update{Input: []byte("0\n")})
	assertEqual(t, c.Status, "Starting synchronization")
	assert.True(t, c.Busy)
	assert.Nil(t, c.Sync)
	assert.Nil(t, c.Quit)
	assert.NotNil(t, c.Abort)
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            one  [f] ")),
		Update{Input: []byte(">\n")})
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            one  \n")))
	assertEqual(t, c.ProcOutput([]byte("\nProceed with propagating updates? [] ")),
		Update{Input: []byte("y\n")})

	assert.Zero(t, c.ProcOutput([]byte("Propagating updates\n")))
	assertEqual(t, c.Status, "Propagating updates")
	assert.Zero(t, c.ProcOutput([]byte("\n\nUNISON 2.51.3 (OCAML 4.11.1) started propagating changes at 18:31:20.92 on 08 Feb 2021\n")))
	assert.Zero(t, c.ProcOutput([]byte("[BGN] Updating file one from /home/vasiliy/tmp/gunison/left to /home/vasiliy/tmp/gunison/right\n")))
	assert.Zero(t, c.ProcOutput([]byte("100%  00:00 ETA")))
	assertEqual(t, c.Progress, "100%  00:00 ETA")
	assertEqual(t, c.ProgressFraction, 1.00)
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))
	assert.Zero(t, c.ProcOutput([]byte("[END] Updating file one\n")))
	assert.Zero(t, c.ProcOutput([]byte("100%  00:00 ETA")))
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))
	assert.Zero(t, c.ProcOutput([]byte("UNISON 2.51.3 (OCAML 4.11.1) finished propagating changes at 18:31:20.92 on 08 Feb 2021\n\n\n")))
	assert.Zero(t, c.ProcOutput([]byte("100%  00:00 ETA")))
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))
	assert.Zero(t, c.ProcOutput([]byte("Saving synchronizer state\n")))
	assertEqual(t, c.Status, "Saving synchronizer state")
	assert.Empty(t, c.Progress)
	assert.Empty(t, c.ProgressFraction)

	assertEqual(t, c.ProcOutput([]byte("Synchronization complete at 18:31:20  (1 item transferred, 0 skipped, 0 failed)\n")),
		Update{Messages: []Message{
			{"Synchronization complete at 18:31:20  (1 item transferred, 0 skipped, 0 failed)", Info},
		}})
	assertEqual(t, c.Status, "Saving synchronizer state")
	assert.True(t, c.Busy)
	assert.True(t, c.Running)

	assert.Zero(t, c.ProcExit(0, nil))
	assertEqual(t, c.Status, "Finished successfully")
	assert.False(t, c.Busy)
	assert.False(t, c.Running)
	assert.NotNil(t, c.Items)
	assert.NotNil(t, c.Plan)
}

func TestTerse(t *testing.T) { // unison -terse
	c := NewCore()

	assert.Zero(t, c.ProcStart())
	assert.Zero(t, c.ProcOutput([]byte("\nleft           right              \n")))
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            one  [f] ")),
		Update{Input: []byte("l\n")})
	assertEqual(t, c.Status, "Assembling plan")
	assertEqual(t, c.Left, "left")
	assertEqual(t, c.Right, "right")
	assert.Zero(t, c.ProcOutput([]byte("  ")))
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            one  \n")))
	assert.Zero(t, c.ProcOutput([]byte("left         : changed file       modified on 2021-02-07 at  1:50:31  size 1146      rw-r--r--\nright        : unchanged file     modified on 2021-02-07 at  1:50:31  size 1146      rw-r--r--\n")))
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            one  [f] ")))
	assertEqual(t, c.Status, "Ready to synchronize")

	assertEqual(t, c.Sync(),
		Update{Input: []byte("0\n")})
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            one  [f] ")),
		Update{Input: []byte(">\n")})
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            one  \n")))
	assertEqual(t, c.ProcOutput([]byte("\nProceed with propagating updates? [] ")),
		Update{Input: []byte("y\n")})
	assertEqual(t, c.Status, "Starting synchronization")

	assert.Zero(t, c.ProcOutput([]byte("[BGN] Updating file one from /home/vasiliy/tmp/gunison/left to /home/vasiliy/tmp/gunison/right\n")))
	assert.Zero(t, c.ProcOutput([]byte("[END] Updating file one\n")))
	assertEqual(t, c.ProcOutput([]byte("Synchronization complete at 01:50:54  (1 item transferred, 0 skipped, 0 failed)\n")),
		Update{Messages: []Message{
			{"Synchronization complete at 01:50:54  (1 item transferred, 0 skipped, 0 failed)", Info},
		}})
	assert.Zero(t, c.ProcExit(0, nil))
	assertEqual(t, c.Status, "Finished successfully")
}

func TestQuit(t *testing.T) {
	c := initCoreMinimalReady(t)
	assertEqual(t, c.Quit(),
		Update{Input: []byte("q\n")})
	assertEqual(t, c.Status, "Quitting Unison")
	assert.True(t, c.Running)
	assert.True(t, c.Busy)
	assert.Nil(t, c.Quit)
	assert.Zero(t, c.ProcOutput([]byte("Terminated!\n")))
	assertEqual(t, c.ProcExit(3, errors.New("exit status 3")),
		Update{Messages: []Message{
			{"Terminated!", Info},
			{"exit status 3", Error},
		}})
	assertEqual(t, c.Status, "Unison exited")
	assert.False(t, c.Running)
}

func TestKilledExternally(t *testing.T) {
	c := initCoreMinimalReady(t)
	assertEqual(t, c.ProcExit(-1, errors.New("signal: killed")),
		Update{Messages: []Message{
			{"signal: killed", Error},
		}})
	assertEqual(t, c.Status, "Unison exited")
}

func TestProgressLookingForChanges(t *testing.T) {
	c := NewCore()
	assert.Zero(t, c.ProcStart())
	assert.Zero(t, c.ProcOutput([]byte("Unison 2.51.3 (ocaml 4.11.1): Contacting server...\n")))
	assert.Zero(t, c.ProcOutput([]byte("Looking for changes\n")))

	assertEqual(t, c.ProcOutput([]byte("| some/file")),
		Update{Progressed: true})
	assertEqual(t, c.Progress, "some/file")
	assertEqual(t, c.ProgressFraction, float64(-1))

	assert.Zero(t, c.ProcOutput([]byte("\r           \r")))

	assertEqual(t, c.ProcOutput([]byte("| another/file")),
		Update{Progressed: true})
	assertEqual(t, c.Progress, "another/file")
	assert.Zero(t, c.ProcOutput([]byte("\r              \r")))

	assertEqual(t, c.ProcOutput([]byte("/ another/file")),
		Update{Progressed: true})
	assertEqual(t, c.Progress, "another/file")
	assert.Zero(t, c.ProcOutput([]byte("\r              \r")))

	assertEqual(t, c.ProcOutput([]byte("- another/file")),
		Update{Progressed: true})
	assertEqual(t, c.Progress, "another/file")
	assert.Zero(t, c.ProcOutput([]byte("\r              \r")))

	assertEqual(t, c.ProcOutput([]byte("\\ another/file")),
		Update{Progressed: true})
	assertEqual(t, c.Progress, "another/file")
	assert.Zero(t, c.ProcOutput([]byte("\r              \r")))

	assertEqual(t, c.ProcOutput([]byte("\\ some file name/so long/that it has...ellipsized to fit into the output")),
		Update{Progressed: true})
	assertEqual(t, c.Progress, "some file name/so long/that it has...ellipsized to fit into the output")
	assert.Zero(t, c.ProcOutput([]byte("\r                                                                      \r")))

	assertEqual(t, c.ProcOutput([]byte("- yet/anoth")),
		Update{Progressed: true})
	assertEqual(t, c.Progress, "yet/anoth") // report updates as soon as they arrive...
	assert.Zero(t, c.ProcOutput([]byte("er/file")))
	assertEqual(t, c.Progress, "yet/another/file") // ...but fix up buffering artifacts
	assert.Zero(t, c.ProcOutput([]byte("\r                \r")))

	assert.Zero(t, c.ProcOutput([]byte("  Waiting for changes from server\n")))
	assertEqual(t, c.Status, "Waiting for changes from server")
	assert.Empty(t, c.Progress)
	assert.Empty(t, c.ProgressFraction)
}

func TestProgressPropagatingUpdates(t *testing.T) {
	c := initCoreMinimalSyncing(t)

	assert.Zero(t, c.ProcOutput([]byte("[BGN] Updating file one from /home/vasiliy/tmp/gunison/left to /home/vasiliy/tmp/gunison/right\n")))
	assert.Empty(t, c.Progress)
	assert.Empty(t, c.ProgressFraction)

	assert.Zero(t, c.ProcOutput([]byte("  0%  73:28 ETA")))
	assertEqual(t, c.Progress, "0%  73:28 ETA")
	assertEqual(t, c.ProgressFraction, 0.00)
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))

	assert.Zero(t, c.ProcOutput([]byte("  8%  07:45 ETA")))
	assertEqual(t, c.Progress, "8%  07:45 ETA")
	assertEqual(t, c.ProgressFraction, 0.08)
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))

	assert.Zero(t, c.ProcOutput([]byte(" 13%  07:51 ETA")))
	assertEqual(t, c.Progress, "13%  07:51 ETA")
	assertEqual(t, c.ProgressFraction, 0.13)
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))

	assert.Zero(t, c.ProcOutput([]byte(" 14%  --:-- ETA")))
	assertEqual(t, c.Progress, "14%  --:-- ETA")
	assertEqual(t, c.ProgressFraction, 0.14)
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))

	assert.Zero(t, c.ProcOutput([]byte(" 94%  00:01 ETA")))
	assertEqual(t, c.Progress, "94%  00:01 ETA")
	assertEqual(t, c.ProgressFraction, 0.94)
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))

	assert.Zero(t, c.ProcOutput([]byte("[END] Updating file one\n")))
	assertEqual(t, c.Progress, "94%  00:01 ETA")
	assertEqual(t, c.ProgressFraction, 0.94)
}

func TestNonexistentProfile(t *testing.T) {
	c := NewCore()
	assert.Zero(t, c.ProcStart())
	assert.Zero(t, c.ProcOutput([]byte("Usage: unison [options]\n    or unison root1 root2 [options]\n    or unison profilename [options]\n\nFor a list of options, type \"unison -help\".\nFor a tutorial on basic usage, type \"unison -doc tutorial\".\nFor other documentation, type \"unison -doc topics\".\n\nProfile /home/vasiliy/tmp/gunison/.unison/nonexistent.prf does not exist\n")))
	assert.True(t, c.Busy)
	assertEqual(t, c.Status, "Starting Unison")
	assertEqual(t, c.ProcExit(1, nil),
		Update{Messages: []Message{
			{"Usage: unison [options]\n    or unison root1 root2 [options]\n    or unison profilename [options]\n\nFor a list of options, type \"unison -help\".\nFor a tutorial on basic usage, type \"unison -doc tutorial\".\nFor other documentation, type \"unison -doc topics\".\n\nProfile /home/vasiliy/tmp/gunison/.unison/nonexistent.prf does not exist", Info},
		}})
	assert.False(t, c.Busy)
	assertEqual(t, c.Status, "Unison exited")
	assert.Nil(t, c.Interrupt)
}

func TestDiff(t *testing.T) {
	c := NewCore()
	assert.Zero(t, c.ProcStart())
	assert.Zero(t, c.ProcOutput([]byte("\nleft           right              \n")))
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            file1  [f] ")),
		Update{Input: []byte("l\n")})
	assert.Zero(t, c.ProcOutput([]byte("  ")))
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            file1  \n")))
	assert.Zero(t, c.ProcOutput([]byte("left         : changed file       modified on 2021-02-13 at 14:29:12  size 1146      rw-r--r--\nright        : unchanged file     modified on 2021-02-13 at 14:29:12  size 1146      rw-r--r--\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            file2  \n")))
	assert.Zero(t, c.ProcOutput([]byte("left         : changed file       modified on 2021-02-13 at 14:29:12  size 1146      rw-r--r--\nright        : unchanged file     modified on 2021-02-13 at 14:29:12  size 1146      rw-r--r--\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            file3  \n")))
	assert.Zero(t, c.ProcOutput([]byte("left         : changed file       modified on 2021-02-13 at 14:29:12  size 1146      rw-r--r--\nright        : unchanged file     modified on 2021-02-13 at 14:29:12  size 1146      rw-r--r--\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            file1  [f] ")))

	assertEqual(t, c.Diff("file3"),
		Update{Input: []byte("0\n")})
	assertEqual(t, c.Status, "Requesting diff")
	assert.True(t, c.Busy)
	assert.Nil(t, c.Sync)
	assert.Nil(t, c.Diff)
	assert.Nil(t, c.Quit)
	assert.NotNil(t, c.Abort)
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            file1  [f] ")),
		Update{Input: []byte("n\n")})
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            file2  [f] ")),
		Update{Input: []byte("n\n")})
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            file3  [f] ")),
		Update{Input: []byte("d\n")})
	assertEqual(t, c.ProcOutput([]byte(`"\ndiff -u '/home/vasiliy/tmp/gunison/right/file3' '/home/vasiliy/tmp/gunison/left/file3'\n\n--- /home/vasiliy/tmp/gunison/right/file3\t2021-02-13 14:29:12.571303322 +0300\n+++ /home/vasiliy/tmp/gunison/left/file3\t2021-02-13 14:29:12.575303310 +0300\n@@ -1,9 +1,9 @@\n Quia est unde laboriosam. Eum ullam deleniti dolores. Magni quasi facere voluptas. Dolor doloribus aut ut sed officiis id. Et aut nostrum est quia corrupti maiores optio.\n \n-Consectetur fuga sed vitae et nihil quia. Eveniet rerum officia repudiandae tenetur molestiae. Magni ipsum et natus accusantium ut consequatur neque. Veniam in voluptate quia. Culpa labore distinctio laudantium maxime voluptate eaque.\n+Consectetur fuga sed vitae et nihil quia. Eveniet rerum officia repudiandae teonsectetur tempore quia id. Perspiciatis enim corrupti aliquam nam accusamus et molestiae rerum. Sint sit exercitationem corrupti omnis.\n \n-Deserunt dignissimos corrupti aut vel. Laboriosam at labore omnis eos et minus porro perspiciatis. Veniam in dignissimos voluptatem exercitationem excepturi reprehenderit sed optio.\n+Facere asperiores unde rerum dignissimos id. Nihil maiores sequi accusamus eum repudiandae et. Nesciunt ab inveniatis. Veniam in dignissimos voluptatem exercitationem excepturi reprehenderit sed optio.\n \n-Omnis repudiandae nobis autem qui autem possimus. Dolorem id a reprehenderit nihil laboriosam non. Dolor minima in soluta. Magni eveniet magnam velit officia consectetur tempore quia id. Perspiciatis enim corrupti aliquam nam accusamus et molestiae rerum. Sint sit exercitationem corrupti omnis.\n+Omnis repudiandae nobis autem qui autem possimus. Dolorem id a reprehenderit nihil laboriosam non. Dolor minima in soluta. Magni eveniet magnam velit officia cnetur molestiae. Magni ipsum et natus accusantium ut consequatur neque. Veniam in voluptate quia. Culpa labore distinctio laudantium maxime voluptate eaque.\n \n-Facere asperiores unde rerum dignissimos id. Nihil maiores sequi accusamus eum repudiandae et. Nesciunt ab inventore repellat enim illum ratione enim voluptatum. Tempore sint quos tempore fugit rerum sit omnis quae. Minus deserunt aut dolores excepturi qui.\n+Deserunt dignissimos corrupti aut vel. Laboriosam at labore omnis eos et minus porro perspictore repellat enim illum ratione enim voluptatum. Tempore sint quos tempore fugit rerum sit omnis quae. Minus deserunt aut dolores excepturi qui.\n\nchanged  ---->            file3  [f] "`)),
		Update{Diff: []byte("--- /home/vasiliy/tmp/gunison/right/file3\t2021-02-13 14:29:12.571303322 +0300\n+++ /home/vasiliy/tmp/gunison/left/file3\t2021-02-13 14:29:12.575303310 +0300\n@@ -1,9 +1,9 @@\n Quia est unde laboriosam. Eum ullam deleniti dolores. Magni quasi facere voluptas. Dolor doloribus aut ut sed officiis id. Et aut nostrum est quia corrupti maiores optio.\n \n-Consectetur fuga sed vitae et nihil quia. Eveniet rerum officia repudiandae tenetur molestiae. Magni ipsum et natus accusantium ut consequatur neque. Veniam in voluptate quia. Culpa labore distinctio laudantium maxime voluptate eaque.\n+Consectetur fuga sed vitae et nihil quia. Eveniet rerum officia repudiandae teonsectetur tempore quia id. Perspiciatis enim corrupti aliquam nam accusamus et molestiae rerum. Sint sit exercitationem corrupti omnis.\n \n-Deserunt dignissimos corrupti aut vel. Laboriosam at labore omnis eos et minus porro perspiciatis. Veniam in dignissimos voluptatem exercitationem excepturi reprehenderit sed optio.\n+Facere asperiores unde rerum dignissimos id. Nihil maiores sequi accusamus eum repudiandae et. Nesciunt ab inveniatis. Veniam in dignissimos voluptatem exercitationem excepturi reprehenderit sed optio.\n \n-Omnis repudiandae nobis autem qui autem possimus. Dolorem id a reprehenderit nihil laboriosam non. Dolor minima in soluta. Magni eveniet magnam velit officia consectetur tempore quia id. Perspiciatis enim corrupti aliquam nam accusamus et molestiae rerum. Sint sit exercitationem corrupti omnis.\n+Omnis repudiandae nobis autem qui autem possimus. Dolorem id a reprehenderit nihil laboriosam non. Dolor minima in soluta. Magni eveniet magnam velit officia cnetur molestiae. Magni ipsum et natus accusantium ut consequatur neque. Veniam in voluptate quia. Culpa labore distinctio laudantium maxime voluptate eaque.\n \n-Facere asperiores unde rerum dignissimos id. Nihil maiores sequi accusamus eum repudiandae et. Nesciunt ab inventore repellat enim illum ratione enim voluptatum. Tempore sint quos tempore fugit rerum sit omnis quae. Minus deserunt aut dolores excepturi qui.\n+Deserunt dignissimos corrupti aut vel. Laboriosam at labore omnis eos et minus porro perspictore repellat enim illum ratione enim voluptatum. Tempore sint quos tempore fugit rerum sit omnis quae. Minus deserunt aut dolores excepturi qui.\n")})
	assertEqual(t, c.Status, "Ready to synchronize")
	assert.False(t, c.Busy)
	assert.NotNil(t, c.Sync)
	assert.NotNil(t, c.Diff)
	assert.NotNil(t, c.Quit)
	assert.Nil(t, c.Abort)

	assertEqual(t, c.Diff("file2"),
		Update{Input: []byte("0\n")})
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            file1  [f] ")),
		Update{Input: []byte("n\n")})
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            file2  [f] ")),
		Update{Input: []byte("d\n")})
	assertEqual(t, c.ProcOutput([]byte("\ndiff -u '/home/vasiliy/tmp/gunison/right/file2' '/home/vasiliy/tmp/gunison/left/file2'\n\n--- /home/vasiliy/tmp/gunison/right/file2\t2021-02-13 14:29:12.571303322 +0300\n+++ /home/vasiliy/tmp/gunison/left/file2\t2021-02-13 14:29:12.571303322 +0300\n@@ -1,6 +1,6 @@\n-Quia est unde laboriosam. Eum ullam deleniti dolores. Magni quasi facere voluptas. Dolor doloribus aut ut sed officiis id. Et aut nostrum est quia corrupti maiores optio.\n+Quia est unde laboriosam. Eum ullam deleniti dolores. Magni quasi facere voluptas. Dolor doloribus aut consequatur neque. Veniam in voluptate quia. Culpa labore distinctio laudantium maxime voluptate t nihil quia. Eveniet rerum officia repudiandae tenetur molestiae. Magni ipsum et natus accusantium ut ut sed officiis id. Et aut nostrum est quia corrupti maiores optio.\n \n-Consectetur fuga sed vitae et nihil quia. Eveniet rerum officia repudiandae tenetur molestiae. Magni ipsum et natus accusantium ut consequatur neque. Veniam in voluptate quia. Culpa labore distinctio laudantium maxime voluptate eaque.\n+Consectetur fuga sed vitae eeaque.\n \n Deserunt dignissimos corrupti aut vel. Laboriosam at labore omnis eos et minus porro perspiciatis. Veniam in dignissimos voluptatem exercitationem excepturi reprehenderit sed optio.\n \n\nchanged  ---->            file2  [f] ")),
		Update{Diff: []byte("--- /home/vasiliy/tmp/gunison/right/file2\t2021-02-13 14:29:12.571303322 +0300\n+++ /home/vasiliy/tmp/gunison/left/file2\t2021-02-13 14:29:12.571303322 +0300\n@@ -1,6 +1,6 @@\n-Quia est unde laboriosam. Eum ullam deleniti dolores. Magni quasi facere voluptas. Dolor doloribus aut ut sed officiis id. Et aut nostrum est quia corrupti maiores optio.\n+Quia est unde laboriosam. Eum ullam deleniti dolores. Magni quasi facere voluptas. Dolor doloribus aut consequatur neque. Veniam in voluptate quia. Culpa labore distinctio laudantium maxime voluptate t nihil quia. Eveniet rerum officia repudiandae tenetur molestiae. Magni ipsum et natus accusantium ut ut sed officiis id. Et aut nostrum est quia corrupti maiores optio.\n \n-Consectetur fuga sed vitae et nihil quia. Eveniet rerum officia repudiandae tenetur molestiae. Magni ipsum et natus accusantium ut consequatur neque. Veniam in voluptate quia. Culpa labore distinctio laudantium maxime voluptate eaque.\n+Consectetur fuga sed vitae eeaque.\n \n Deserunt dignissimos corrupti aut vel. Laboriosam at labore omnis eos et minus porro perspiciatis. Veniam in dignissimos voluptatem exercitationem excepturi reprehenderit sed optio.\n \n")})

	assertEqual(t, c.Diff("file1"),
		Update{Input: []byte("0\n")})
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            file1  [f] ")),
		Update{Input: []byte("d\n")})
	assertEqual(t, c.ProcOutput([]byte("\ndiff -u '/home/vasiliy/tmp/gunison/right/file1' '/home/vasiliy/tmp/gunison/left/file1'\n\n--- /home/vasiliy/tmp/gunison/right/file1\t2021-02-13 14:29:12.571303322 +0300\n+++ /home/vasiliy/tmp/gunison/left/file1\t2021-02-13 14:29:12.571303322 +0300\n@@ -1,6 +1,6 @@\n-Quia est unde laboriosam. Eum ullam deleniti dolores. Magni quasi facere voluptas. Dolor doloribus aut ut sed officiis id. Et aut nostrum est quia corrupti maiores optio.\n+Quia est unde laboriosam. Eum ullam deleniti dolorrupti maiores optio.\n \n-Consectetur fuga sed vitae et nihil quia. Eveniet rerum officia repudiandae tenetur molestiae. Magni ipsum et natus accusantium ut consequatur neque. Veniam in voluptate quia. Culpa labore distinctio laudantium maxime voluptate eaque.\n+Consectetur fuga sed vitae eut ut sed officiis id. Et aut nostrum est quia cores. Magni quasi facere voluptas. Dolor doloribus at nihil quia. Eveniet rerum officia repudiandae tenetur molestiae. Magni ipsum et natus accusantium ut consequatur neque. Veniam in voluptate quia. Culpa labore distinctio laudantium maxime voluptate eaque.\n \n Deserunt dignissimos corrupti aut vel. Laboriosam at labore omnis eos et minus porro perspiciatis. Veniam in dignissimos voluptatem exercitationem excepturi reprehenderit sed optio.\n \n\nchanged  ---->            file1  [f] ")),
		Update{Diff: []byte("--- /home/vasiliy/tmp/gunison/right/file1\t2021-02-13 14:29:12.571303322 +0300\n+++ /home/vasiliy/tmp/gunison/left/file1\t2021-02-13 14:29:12.571303322 +0300\n@@ -1,6 +1,6 @@\n-Quia est unde laboriosam. Eum ullam deleniti dolores. Magni quasi facere voluptas. Dolor doloribus aut ut sed officiis id. Et aut nostrum est quia corrupti maiores optio.\n+Quia est unde laboriosam. Eum ullam deleniti dolorrupti maiores optio.\n \n-Consectetur fuga sed vitae et nihil quia. Eveniet rerum officia repudiandae tenetur molestiae. Magni ipsum et natus accusantium ut consequatur neque. Veniam in voluptate quia. Culpa labore distinctio laudantium maxime voluptate eaque.\n+Consectetur fuga sed vitae eut ut sed officiis id. Et aut nostrum est quia cores. Magni quasi facere voluptas. Dolor doloribus at nihil quia. Eveniet rerum officia repudiandae tenetur molestiae. Magni ipsum et natus accusantium ut consequatur neque. Veniam in voluptate quia. Culpa labore distinctio laudantium maxime voluptate eaque.\n \n Deserunt dignissimos corrupti aut vel. Laboriosam at labore omnis eos et minus porro perspiciatis. Veniam in dignissimos voluptatem exercitationem excepturi reprehenderit sed optio.\n \n")})
}

func TestDiffNoOutput(t *testing.T) {
	// When the diff command produces no output, it's probably a GUI one, so we silently ignore it.
	c := initCoreMinimalReady(t)
	assertEqual(t, c.Diff("one"),
		Update{Input: []byte("0\n")})
	assertEqual(t, c.Status, "Requesting diff")
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            one  [f] ")),
		Update{Input: []byte("d\n")})
	assert.Zero(t, c.ProcOutput([]byte("\ntrue '/home/vasiliy/tmp/gunison/left/one' '/home/vasiliy/tmp/gunison/right/one'\n\n\n\nchanged  ---->            one  [f] ")))
	assertEqual(t, c.Status, "Ready to synchronize")
}

func TestDiffDirectory(t *testing.T) {
	c := NewCore()
	assert.Zero(t, c.ProcStart())
	assert.Zero(t, c.ProcOutput([]byte("\nleft           right              \n")))
	assertEqual(t, c.ProcOutput([]byte("changed  <-?-> new dir    one hundred/one hundred one  [] ")),
		Update{Input: []byte("l\n")})
	assert.Zero(t, c.ProcOutput([]byte("  ")))
	assert.Zero(t, c.ProcOutput([]byte("changed  <-?-> new dir    one hundred/one hundred one  \n")))
	assert.Zero(t, c.ProcOutput([]byte("left         : changed file       modified on 2021-02-13 at 14:56:44  size 1000      rw-r--r--\nright        : new dir            modified on 2021-02-13 at 14:56:44  size 2292      rwxr-xr-x\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("changed  <-?-> new dir    one hundred/one hundred one  [] ")))
	assertEqual(t, c.Diff("one hundred/one hundred one"),
		Update{Input: []byte("0\n")})
	assertEqual(t, c.ProcOutput([]byte("changed  <-?-> new dir    one hundred/one hundred one  [] ")),
		Update{Input: []byte("d\n")})
	assertEqual(t, c.ProcOutput([]byte("Can't diff: path doesn't refer to a file in both replicas\nchanged  <-?-> new dir    one hundred/one hundred one  [] ")),
		Update{Messages: []Message{
			{"Can't diff: path doesn't refer to a file in both replicas", Error},
		}})
	assertEqual(t, c.Status, "Ready to synchronize")
}

func TestDiffAbort(t *testing.T) {
	c := NewCore()
	assert.Zero(t, c.ProcStart())
	assert.Zero(t, c.ProcOutput([]byte("\nleft           right              \n")))
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            file1  [f] ")),
		Update{Input: []byte("l\n")})
	assert.Zero(t, c.ProcOutput([]byte("  ")))
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            file1  \n")))
	assert.Zero(t, c.ProcOutput([]byte("left         : changed file       modified on 2021-02-13 at 15:12:12  size 1146      rw-r--r--\nright        : unchanged file     modified on 2021-02-13 at 15:12:12  size 1146      rw-r--r--\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            file2  \n")))
	assert.Zero(t, c.ProcOutput([]byte("left         : changed file       modified on 2021-02-13 at 15:12:12  size 1146      rw-r--r--\nright        : unchanged file     modified on 2021-02-13 at 15:12:12  size 1146      rw-r--r--\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            file1  [f] ")))

	assertEqual(t, c.Diff("file2"),
		Update{Input: []byte("0\n")})
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            file1  [f] ")),
		Update{Input: []byte("n\n")})
	// While we're still seeking to the requested file in Unison prompt, we can abort gracefully.
	assert.Zero(t, c.Abort())
	assertEqual(t, c.Status, "Waiting for Unison")
	assert.True(t, c.Busy)
	assert.Nil(t, c.Sync)
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            file2  [f] ")))
	assertEqual(t, c.Status, "Ready to synchronize")
	assert.False(t, c.Busy)
	assert.NotNil(t, c.Sync)

	assertEqual(t, c.Diff("file2"),
		Update{Input: []byte("0\n")})
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            file1  [f] ")),
		Update{Input: []byte("n\n")})
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            file2  [f] ")),
		Update{Input: []byte("d\n")})
	// But if we have already requested the diff, the only way to abort is by interrupting.
	assertEqual(t, c.Abort(),
		Update{Interrupt: true})
}

func TestDiffBadCommand(t *testing.T) { // unison -diff 'diff --bad-option'
	c := initCoreMinimalReady(t)
	assertEqual(t, c.Diff("one"),
		Update{Input: []byte("0\n")})
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            one  [f] ")),
		Update{Input: []byte("d\n")})
	assert.Zero(t, c.ProcOutput([]byte("diff: unrecognized option '--bad-option'\n")))
	assert.Zero(t, c.ProcOutput([]byte("diff: ")))
	assert.Zero(t, c.ProcOutput([]byte("Try 'diff --help' for more information.\n")))
	assertEqual(t, c.ProcOutput([]byte("\ndiff --bad-option '/home/vasiliy/tmp/gunison/left/one' '/home/vasiliy/tmp/gunison/right/one'\n\n\n\nchanged  ---->            one  [f] ")),
		Update{Messages: []Message{
			// Unison prepends diff output with a blank line, the command line, and two more blank lines.
			// This means that anything printed after the prompt but before a blank line is definitely *not*
			// normal diff output.
			{"diff: unrecognized option '--bad-option'\ndiff: Try 'diff --help' for more information.", Warning},
		}})
	assertEqual(t, c.Status, "Ready to synchronize")
}

func TestMerge(t *testing.T) {
	c := initCoreMinimalReady(t)
	c.Plan["one"] = Merge
	assertEqual(t, c.Sync(),
		Update{Input: []byte("0\n")})
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            one  [f] ")),
		Update{Input: []byte("m\n")})
	assert.Zero(t, c.ProcOutput([]byte("changed  <=M=>            one  \n")))
	assertEqual(t, c.ProcOutput([]byte("\nProceed with propagating updates? [] ")),
		Update{Input: []byte("y\n")})
	assert.Zero(t, c.ProcOutput([]byte("Merge command: meld ''/home/vasiliy/tmp/gunison/left/.unison.merge1-one'' ''/home/vasiliy/tmp/gunison/left/.unison.merge2-one''\n")))
	assert.Zero(t, c.ProcOutput([]byte("Merge result (exited (0)):\n\n")))
	assert.Zero(t, c.ProcOutput([]byte("No outputs detected \n")))
	assert.Zero(t, c.ProcOutput([]byte("No output from merge cmd and both original files are still present\n")))
	assert.Zero(t, c.ProcOutput([]byte("Merge program made files equal\n")))
	assertEqual(t, c.ProcOutput([]byte("Warning: 'backupcurrent' is not set for path one\n")),
		Update{Messages: []Message{
			{"Warning: 'backupcurrent' is not set for path one", Warning},
		}})
	assertEqual(t, c.ProcOutput([]byte("Synchronization complete at 16:40:04  (1 item transferred, 0 skipped, 0 failed)\n")),
		Update{Messages: []Message{
			{"Synchronization complete at 16:40:04  (1 item transferred, 0 skipped, 0 failed)", Info},
		}})
	assert.Zero(t, c.ProcExit(0, nil))
	assertEqual(t, c.Status, "Finished successfully")
}

func TestMergeFailed(t *testing.T) {
	c := initCoreMinimalReady(t)
	c.Plan["one"] = Merge
	assertEqual(t, c.Sync(),
		Update{Input: []byte("0\n")})
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            one  [f] ")),
		Update{Input: []byte("m\n")})
	assert.Zero(t, c.ProcOutput([]byte("changed  <=M=>            one  \n")))
	assertEqual(t, c.ProcOutput([]byte("\nProceed with propagating updates? [] ")),
		Update{Input: []byte("y\n")})
	assert.Zero(t, c.ProcOutput([]byte("Merge command: meld ''/home/vasiliy/tmp/gunison/left/.unison.merge1-one'' ''/home/vasiliy/tmp/gunison/left/.unison.merge2-one''\n")))
	assert.Zero(t, c.ProcOutput([]byte("Merge result (exited (0)):\n\n")))
	assert.Zero(t, c.ProcOutput([]byte("No outputs detected \n")))
	assert.Zero(t, c.ProcOutput([]byte("No output from merge cmd and both original files are still present\n")))
	assertEqual(t, c.ProcOutput([]byte("\nFailed [one]: Merge program didn't change either temp file\n")),
		Update{Messages: []Message{
			{"Failed [one]: Merge program didn't change either temp file", Error},
		}})
	assertEqual(t, c.ProcOutput([]byte("Synchronization incomplete at 16:49:21  (0 items transferred, 0 skipped, 1 failed)\n")),
		Update{Messages: []Message{
			{"Synchronization incomplete at 16:49:21  (0 items transferred, 0 skipped, 1 failed)", Warning},
		}})
	assert.Zero(t, c.ProcExit(2, nil))
	assertEqual(t, c.Status, "Finished with errors")
}

func TestMergeDir(t *testing.T) {
	c := NewCore()
	assert.Zero(t, c.ProcStart())
	assert.Zero(t, c.ProcOutput([]byte("\nleft           right              \n")))
	assertEqual(t, c.ProcOutput([]byte("new dir  ---->            one  [f] ")),
		Update{Input: []byte("l\n")})
	assert.Zero(t, c.ProcOutput([]byte("  ")))
	assert.Zero(t, c.ProcOutput([]byte("new dir  ---->            one  \n")))
	assert.Zero(t, c.ProcOutput([]byte("left         : new dir            modified on 2021-02-25 at 17:06:22  size 0         rwxrwxr-x\nright        : absent\n")))
	assert.Zero(t, c.ProcOutput([]byte("new dir  ---->            one  [f] ")))
	c.Plan["one"] = Merge
	assertEqual(t, c.Sync(),
		Update{Input: []byte("0\n")})
	assertEqual(t, c.ProcOutput([]byte("new dir  ---->            one  [f] ")),
		Update{Input: []byte("m\n")})
	assert.Zero(t, c.ProcOutput([]byte("new dir  <=M=>            one  \n")))
	assertEqual(t, c.ProcOutput([]byte("\nProceed with propagating updates? [] ")),
		Update{Input: []byte("y\n")})
	assertEqual(t, c.ProcOutput([]byte("Failed [one]: Can only merge two existing files\n")),
		Update{Messages: []Message{
			{"Failed [one]: Can only merge two existing files", Error},
		}})
	assertEqual(t, c.ProcOutput([]byte("Synchronization incomplete at 17:06:28  (0 items transferred, 0 skipped, 1 failed)\n")),
		Update{Messages: []Message{
			{"Synchronization incomplete at 17:06:28  (0 items transferred, 0 skipped, 1 failed)", Warning},
		}})
	assertEqual(t, c.ProcOutput([]byte("  failed: one\n")),
		Update{Messages: []Message{
			{"failed: one", Error},
		}})
	assert.Zero(t, c.ProcExit(2, nil))
	assertEqual(t, c.Status, "Finished with errors")
}

func TestMergeBadProgram(t *testing.T) { // unison -merge 'Name * -> nonexistent-program'
	c := initCoreMinimalReady(t)
	c.Plan["one"] = Merge
	assertEqual(t, c.Sync(),
		Update{Input: []byte("0\n")})
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            one  [f] ")),
		Update{Input: []byte("m\n")})
	assert.Zero(t, c.ProcOutput([]byte("changed  <=M=>            one  \n")))
	assertEqual(t, c.ProcOutput([]byte("\nProceed with propagating updates? [] ")),
		Update{Input: []byte("y\n")})
	assert.Zero(t, c.ProcOutput([]byte("Merge command: nonexistent-program\n")))
	assertEqual(t, c.ProcOutput([]byte("Merge result (exited (127)):\n/bin/sh: 1: nonexistent-program: not found\n\nExited with status 127\n")),
		Update{Messages: []Message{
			{"Merge result (exited (127)):", Warning},
			{"/bin/sh: 1: nonexistent-program: not found", Info},
			{"Exited with status 127", Info},
		}})
	assert.Zero(t, c.ProcOutput([]byte("No outputs detected \n")))
	assert.Zero(t, c.ProcOutput([]byte("No output from merge cmd and both original files are still present\n")))
	assertEqual(t, c.ProcOutput([]byte("/bin/sh: 1: nonexistent-program: not found\n\nExited with status 127\nFailed [one]: Merge program didn't change either temp file\n")),
		Update{Messages: []Message{
			{"/bin/sh: 1: nonexistent-program: not found", Info},
			{"Exited with status 127", Info},
			{"Failed [one]: Merge program didn't change either temp file", Error},
		}})
}

// TODO: tests for merge cases other than "No output from merge cmd and both original files are still present"

func TestReplicaMissing(t *testing.T) {
	c := NewCore()
	assert.Zero(t, c.ProcStart())
	assert.Zero(t, c.ProcOutput([]byte("Unison 2.51.3 (ocaml 4.11.1): Contacting server...\n")))
	assert.Zero(t, c.ProcOutput([]byte("Looking for changes\n")))
	assert.Zero(t, c.ProcOutput([]byte("Reconciling changes\n")))
	assert.Zero(t, c.ProcOutput([]byte("The root of one of the replicas has been completely emptied.\nUnison may delete everything in the other replica.  (Set the \n'confirmbigdel' preference to false to disable this check.)\n\n")))
	upd := c.ProcOutput([]byte("Do you really want to proceed? [] "))
	assertEqual(t, upd.Alert.Text, "The root of one of the replicas has been completely emptied.\nUnison may delete everything in the other replica.  (Set the \n'confirmbigdel' preference to false to disable this check.)\n\nDo you really want to proceed?")
	assertEqual(t, upd.Alert.Importance, Warning)
	assertEqual(t, c.Status, "Reconciling changes")
	assert.True(t, c.Busy)
	assertEqual(t, upd.Alert.Proceed(),
		Update{Input: []byte("y\n")})
	assert.Zero(t, c.ProcOutput([]byte("\nleft           right              \n")))
	assertEqual(t, c.ProcOutput([]byte("         <---- deleted      [f] ")),
		Update{Input: []byte("l\n")})
	assert.Zero(t, c.ProcOutput([]byte("  ")))
	assert.Zero(t, c.ProcOutput([]byte("         <---- deleted      \n")))
	assert.Zero(t, c.ProcOutput([]byte("left         : unchanged dir      modified on 2021-02-06 at 18:31:42  size 1146      rwxr-xr-x\nright        : deleted\n")))
	assert.Zero(t, c.ProcOutput([]byte("         <---- deleted      [f] ")))
	assertEqual(t, c.Status, "Ready to synchronize")
	assertEqual(t, c.Items, []Item{
		{
			Path: "",
			Left: Content{Directory, Unchanged, "modified on 2021-02-06 at 18:31:42  size 1146      rwxr-xr-x",
				time.Date(2021, 2, 6, 18, 31, 42, 0, time.Local), 1146},
			Right:  Content{Absent, Deleted, "", time.Time{}, 0},
			Action: RightToLeft,
		},
	})
	assert.False(t, c.Busy)
	assert.NotNil(t, c.Sync)
}

func TestReplicaMissingAbort(t *testing.T) {
	c := NewCore()
	assert.Zero(t, c.ProcStart())
	assert.Zero(t, c.ProcOutput([]byte("The root of one of the replicas has been completely emptied.\nUnison may delete everything in the other replica.  (Set the \n'confirmbigdel' preference to false to disable this check.)\n\n")))
	upd := c.ProcOutput([]byte("Do you really want to proceed? [] "))
	assertEqual(t, upd.Alert.Abort(),
		Update{Input: []byte("q\n")})
	assertEqual(t, c.Status, "Quitting Unison")
}

func TestNewReplicas(t *testing.T) {
	c := NewCore()
	assert.Zero(t, c.ProcStart())
	assert.Zero(t, c.ProcOutput([]byte("Unison 2.51.3 (ocaml 4.11.1): Contacting server...\n")))
	assert.Zero(t, c.ProcOutput([]byte("Looking for changes\n")))
	assert.Zero(t, c.ProcOutput([]byte("Warning: ")))
	assert.Zero(t, c.ProcOutput([]byte("No archive files were found for these roots, whose canonical names are:\n\t/home/vasiliy/tmp/gunison/left\n\t/home/vasiliy/tmp/gunison/right\nThis can happen either\nbecause this is the first time you have synchronized these roots, \nor because you have upgraded Unison to a new version with a different\narchive format.  \n\nUpdate detection may take a while on this run if the replicas are \nlarge.\n\nUnison will assume that the 'last synchronized state' of both replicas\nwas completely empty.  This means that any files that are different\nwill be reported as conflicts, and any files that exist only on one\nreplica will be judged as new and propagated to the other replica.\nIf the two replicas are identical, then no changes will be reported.\n\nIf you see this message repeatedly, it may be because one of your machines\nis getting its address from DHCP, which is causing its host name to change\nbetween synchronizations.  See the documentation for the UNISONLOCALHOSTNAME\nenvironment variable for advice on how to correct this.\n\nDonations to the Unison project are gratefully accepted: \nhttp://www.cis.upenn.edu/~bcpierce/unison\n\n\n")))
	assert.Zero(t, c.ProcOutput([]byte("Press return to continue.[")))
	upd := c.ProcOutput([]byte("<spc>] "))
	assertEqual(t, upd.Alert.Text, "Warning: No archive files were found for these roots, whose canonical names are:\n\t/home/vasiliy/tmp/gunison/left\n\t/home/vasiliy/tmp/gunison/right\nThis can happen either\nbecause this is the first time you have synchronized these roots, \nor because you have upgraded Unison to a new version with a different\narchive format.  \n\nUpdate detection may take a while on this run if the replicas are \nlarge.\n\nUnison will assume that the 'last synchronized state' of both replicas\nwas completely empty.  This means that any files that are different\nwill be reported as conflicts, and any files that exist only on one\nreplica will be judged as new and propagated to the other replica.\nIf the two replicas are identical, then no changes will be reported.\n\nIf you see this message repeatedly, it may be because one of your machines\nis getting its address from DHCP, which is causing its host name to change\nbetween synchronizations.  See the documentation for the UNISONLOCALHOSTNAME\nenvironment variable for advice on how to correct this.\n\nDonations to the Unison project are gratefully accepted: \nhttp://www.cis.upenn.edu/~bcpierce/unison")
	assertEqual(t, upd.Alert.Importance, Warning)
	assertEqual(t, c.Status, "Looking for changes")
	assert.True(t, c.Busy)
	assertEqual(t, upd.Alert.Proceed(),
		Update{Input: []byte("\n")})
	assert.Zero(t, c.ProcOutput([]byte("Reconciling changes\n")))
	assertEqual(t, c.Status, "Reconciling changes")
}

func TestNewReplicasAbort(t *testing.T) {
	c := NewCore()
	assert.Zero(t, c.ProcStart())
	assert.Zero(t, c.ProcOutput([]byte("Warning: ")))
	assert.Zero(t, c.ProcOutput([]byte("No archive files were found for these roots, whose canonical names are:\n\t/home/vasiliy/tmp/gunison/left\n\t/home/vasiliy/tmp/gunison/right\nThis can happen either\nbecause this is the first time you have synchronized these roots, \nor because you have upgraded Unison to a new version with a different\narchive format.  \n\nUpdate detection may take a while on this run if the replicas are \nlarge.\n\nUnison will assume that the 'last synchronized state' of both replicas\nwas completely empty.  This means that any files that are different\nwill be reported as conflicts, and any files that exist only on one\nreplica will be judged as new and propagated to the other replica.\nIf the two replicas are identical, then no changes will be reported.\n\nIf you see this message repeatedly, it may be because one of your machines\nis getting its address from DHCP, which is causing its host name to change\nbetween synchronizations.  See the documentation for the UNISONLOCALHOSTNAME\nenvironment variable for advice on how to correct this.\n\nDonations to the Unison project are gratefully accepted: \nhttp://www.cis.upenn.edu/~bcpierce/unison\n\n\n")))
	assert.Zero(t, c.ProcOutput([]byte("Press return to continue.[")))
	upd := c.ProcOutput([]byte("<spc>] "))
	assertEqual(t, upd.Alert.Abort(),
		Update{Input: []byte("q\n")})
	assertEqual(t, c.Status, "Quitting Unison")
}

func TestEmpty(t *testing.T) {
	c := NewCore()
	assert.Zero(t, c.ProcStart())
	assert.Zero(t, c.ProcOutput([]byte("Nothing to do: replicas have not changed since last sync.\n")))
	assertEqual(t, c.ProcExit(0, nil),
		Update{Messages: []Message{
			{"Nothing to do: replicas have not changed since last sync.", Info},
		}})
	assertEqual(t, c.Status, "Finished successfully")
	assert.False(t, c.Running)
}

func TestIdenticalChanges(t *testing.T) {
	c := NewCore()
	assert.Zero(t, c.ProcStart())
	assert.Zero(t, c.ProcOutput([]byte("Unison 2.51.3 (ocaml 4.11.1): Contacting server...\n")))
	assert.Zero(t, c.ProcOutput([]byte("Looking for changes\n")))
	assert.Zero(t, c.ProcOutput([]byte("Reconciling changes\n")))
	assert.Zero(t, c.ProcOutput([]byte("Nothing to do: replicas have been changed only in identical ways since last sync.\n")))
	assertEqual(t, c.Status, "Reconciling changes")
	assertEqual(t, c.ProcExit(0, nil),
		Update{Messages: []Message{
			{"Nothing to do: replicas have been changed only in identical ways since last sync.", Info},
		}})
	assertEqual(t, c.Status, "Finished successfully")
	assert.False(t, c.Running)
}

func TestAssorted(t *testing.T) {
	c := NewCore()
	assert.Zero(t, c.ProcStart())
	assert.Zero(t, c.ProcOutput([]byte("\nlocal          tanais             \n")))
	assertEqual(t, c.ProcOutput([]byte("changed  <-?-> new dir    one hundred/one hundred one  [] ")),
		Update{Input: []byte("l\n")})
	assert.Zero(t, c.ProcOutput([]byte("  ")))
	assert.Zero(t, c.ProcOutput([]byte("changed  <-?-> new dir    one hundred/one hundred one  \n")))
	assert.Zero(t, c.ProcOutput([]byte("local        : changed file       modified on 2021-02-06 at 18:41:58  size 1000      rw-r--r--\ntanais       : new dir            modified on 2021-02-06 at 18:41:58  size 2292      rwxr-xr-x\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("new file <-?-> deleted    one hundred/one hundred two  \n")))
	assert.Zero(t, c.ProcOutput([]byte("local        : new file           modified on 2021-02-06 at 18:41:58  size 1146      rw-r--r--\ntanais       : deleted\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("changed  <-?-> changed    six/nine  \n")))
	assert.Zero(t, c.ProcOutput([]byte("local        : changed file       modified on 2021-02-06 at 18:41:58  size 1147000   rw-r--r--\ntanais       : changed file       modified on 2021-02-06 at 18:41:58  size 1147000   rw-rw-r--\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("props    <-?-> props      twenty one  \n")))
	assert.Zero(t, c.ProcOutput([]byte("local        : changed props      modified on 2021-02-06 at 18:41:58  size 1146      rw-r--r--\ntanais       : changed props      modified on 2021-02-06 at 18:41:58  size 1146      rw-rw-r--\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("         <---- changed    deeply/nested/sub/directory/with/file  \n")))
	assert.Zero(t, c.ProcOutput([]byte("local        : unchanged file     modified on 2021-02-06 at 18:41:58  size 1146      rw-r--r--\ntanais       : changed file       modified on 2021-02-06 at 18:41:58  size 1146      rw-rw-r--\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("new link ---->            eighteen  \n")))
	assert.Zero(t, c.ProcOutput([]byte("local        : new symlink        modified on 1970-01-01 at  3:00:00  size 0         unknown permissions\ntanais       : absent\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("         <---- changed    here is a rather long and funny file name, 社會科學院語學研究所\t\v\f \u0085 \u1680\u2002\u2003\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u200b\u2028\u2029\u202f\u205f\u3000ﾟ･✿ヾ╲(｡◕‿◕｡)╱✿･ﾟ/here is a rather long and funny file name, 社會科學院語學研究所\t\v\f \u0085 \u1680\u2002\u2003\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u200b\u2028\u2029\u202f\u205f\u3000ﾟ･✿ヾ╲(｡◕‿◕｡)╱✿･ﾟ  \n")))
	assert.Zero(t, c.ProcOutput([]byte("local        : unchanged file     modified on 2021-02-06 at 18:41:58  size 1146      rw-r--r--\ntanais       : changed file       modified on 2021-02-06 at 18:41:58  size 1146      rw-rw-r--\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("new file ---->            seventeen  \n")))
	assert.Zero(t, c.ProcOutput([]byte("local        : new file           modified on 2021-02-06 at 18:41:58  size 0         rw-r--r--\ntanais       : absent\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("         <---- changed    six/eight  \n")))
	assert.Zero(t, c.ProcOutput([]byte("local        : unchanged file     modified on 2021-02-06 at 18:41:58  size 1146      rw-r--r--\ntanais       : changed file       modified on 2021-02-06 at 18:41:58  size 1147000   rw-rw-r--\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("         <---- changed    six/eleven  \n")))
	assert.Zero(t, c.ProcOutput([]byte("local        : unchanged file     modified on 2021-02-06 at 18:41:58  size 10000000  rw-r--r--\ntanais       : changed file       modified on 2021-02-06 at 18:41:58  size 10000000  rw-rw-r--\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("         <---- deleted    six/fourteen  \n")))
	assert.Zero(t, c.ProcOutput([]byte("local        : unchanged dir      modified on 2021-02-06 at 18:41:58  size 2292      rwxr-xr-x\ntanais       : deleted\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("         <---- changed    six/seven  \n")))
	assert.Zero(t, c.ProcOutput([]byte("local        : unchanged file     modified on 2021-02-06 at 18:41:58  size 0         rw-r--r--\ntanais       : changed file       modified on 2021-02-06 at 18:41:58  size 1146      rw-rw-r--\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("props    ---->            six/ten  \n")))
	assert.Zero(t, c.ProcOutput([]byte("local        : changed props      modified on 2021-02-06 at 18:41:58  size 1000      rwx------\ntanais       : unchanged file     modified on 2021-02-06 at 18:41:58  size 1000      rw-r--r--\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("         <---- deleted    three  \n")))
	assert.Zero(t, c.ProcOutput([]byte("local        : unchanged file     modified on 2021-02-06 at 18:41:58  size 1147000   rw-r--r--\ntanais       : deleted\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("         <---- props      twelve  \n")))
	assert.Zero(t, c.ProcOutput([]byte("local        : unchanged dir      modified on 2021-02-06 at 18:41:58  size 0         rwxr-xr-x\ntanais       : dir props changed  modified on 2021-02-06 at 18:41:58  size 0         rwx------\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("         <---- new dir    twenty  \n")))
	assert.Zero(t, c.ProcOutput([]byte("local        : absent\ntanais       : new dir            modified on 2021-02-06 at 18:41:58  size 0         rwxr-xr-x\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            two  \n")))
	assert.Zero(t, c.ProcOutput([]byte("local        : changed file       modified on 2021-02-06 at 18:41:58  size 1146      rw-r--r--\ntanais       : unchanged file     modified on 2021-02-06 at 18:41:58  size 1146      rw-r--r--\n")))
	assert.Zero(t, c.ProcOutput([]byte("changed  <-?-> new dir    one hundred/one hundred one  [] ")))

	assertEqual(t, c.Items, []Item{
		{
			Path: "one hundred/one hundred one",
			Left: Content{File, Modified, "modified on 2021-02-06 at 18:41:58  size 1000      rw-r--r--",
				time.Date(2021, 2, 6, 18, 41, 58, 0, time.Local), 1000},
			Right: Content{Directory, Created, "modified on 2021-02-06 at 18:41:58  size 2292      rwxr-xr-x",
				time.Date(2021, 2, 6, 18, 41, 58, 0, time.Local), 2292},
			Action: Skip,
		},
		{
			Path: "one hundred/one hundred two",
			Left: Content{File, Created, "modified on 2021-02-06 at 18:41:58  size 1146      rw-r--r--",
				time.Date(2021, 2, 6, 18, 41, 58, 0, time.Local), 1146},
			Right:  Content{Absent, Deleted, "", time.Time{}, 0},
			Action: Skip,
		},
		{
			Path: "six/nine",
			Left: Content{File, Modified, "modified on 2021-02-06 at 18:41:58  size 1147000   rw-r--r--",
				time.Date(2021, 2, 6, 18, 41, 58, 0, time.Local), 1147000},
			Right: Content{File, Modified, "modified on 2021-02-06 at 18:41:58  size 1147000   rw-rw-r--",
				time.Date(2021, 2, 6, 18, 41, 58, 0, time.Local), 1147000},
			Action: Skip,
		},
		{
			Path: "twenty one",
			Left: Content{File, PropsChanged, "modified on 2021-02-06 at 18:41:58  size 1146      rw-r--r--",
				time.Date(2021, 2, 6, 18, 41, 58, 0, time.Local), 1146},
			Right: Content{File, PropsChanged, "modified on 2021-02-06 at 18:41:58  size 1146      rw-rw-r--",
				time.Date(2021, 2, 6, 18, 41, 58, 0, time.Local), 1146},
			Action: Skip,
		},
		{
			Path: "deeply/nested/sub/directory/with/file",
			Left: Content{File, Unchanged, "modified on 2021-02-06 at 18:41:58  size 1146      rw-r--r--",
				time.Date(2021, 2, 6, 18, 41, 58, 0, time.Local), 1146},
			Right: Content{File, Modified, "modified on 2021-02-06 at 18:41:58  size 1146      rw-rw-r--",
				time.Date(2021, 2, 6, 18, 41, 58, 0, time.Local), 1146},
			Action: RightToLeft,
		},
		{
			Path: "eighteen",
			Left: Content{Symlink, Created, "modified on 1970-01-01 at  3:00:00  size 0         unknown permissions",
				time.Date(1970, 1, 1, 3, 0, 0, 0, time.Local), 0},
			Right:  Content{Type: Absent},
			Action: LeftToRight,
		},
		{
			Path: "here is a rather long and funny file name, 社會科學院語學研究所\t\v\f \u0085 \u1680\u2002\u2003\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u200b\u2028\u2029\u202f\u205f\u3000ﾟ･✿ヾ╲(｡◕‿◕｡)╱✿･ﾟ/here is a rather long and funny file name, 社會科學院語學研究所\t\v\f \u0085 \u1680\u2002\u2003\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u200b\u2028\u2029\u202f\u205f\u3000ﾟ･✿ヾ╲(｡◕‿◕｡)╱✿･ﾟ",
			Left: Content{File, Unchanged, "modified on 2021-02-06 at 18:41:58  size 1146      rw-r--r--",
				time.Date(2021, 2, 6, 18, 41, 58, 0, time.Local), 1146},
			Right: Content{File, Modified, "modified on 2021-02-06 at 18:41:58  size 1146      rw-rw-r--",
				time.Date(2021, 2, 6, 18, 41, 58, 0, time.Local), 1146},
			Action: RightToLeft,
		},
		{
			Path: "seventeen",
			Left: Content{File, Created, "modified on 2021-02-06 at 18:41:58  size 0         rw-r--r--",
				time.Date(2021, 2, 6, 18, 41, 58, 0, time.Local), 0},
			Right:  Content{Type: Absent},
			Action: LeftToRight,
		},
		{
			Path: "six/eight",
			Left: Content{File, Unchanged, "modified on 2021-02-06 at 18:41:58  size 1146      rw-r--r--",
				time.Date(2021, 2, 6, 18, 41, 58, 0, time.Local), 1146},
			Right: Content{File, Modified, "modified on 2021-02-06 at 18:41:58  size 1147000   rw-rw-r--",
				time.Date(2021, 2, 6, 18, 41, 58, 0, time.Local), 1147000},
			Action: RightToLeft,
		},
		{
			Path: "six/eleven",
			Left: Content{File, Unchanged, "modified on 2021-02-06 at 18:41:58  size 10000000  rw-r--r--",
				time.Date(2021, 2, 6, 18, 41, 58, 0, time.Local), 10000000},
			Right: Content{File, Modified, "modified on 2021-02-06 at 18:41:58  size 10000000  rw-rw-r--",
				time.Date(2021, 2, 6, 18, 41, 58, 0, time.Local), 10000000},
			Action: RightToLeft,
		},
		{
			Path: "six/fourteen",
			Left: Content{Directory, Unchanged, "modified on 2021-02-06 at 18:41:58  size 2292      rwxr-xr-x",
				time.Date(2021, 2, 6, 18, 41, 58, 0, time.Local), 2292},
			Right:  Content{Type: Absent, Status: Deleted},
			Action: RightToLeft,
		},
		{
			Path: "six/seven",
			Left: Content{File, Unchanged, "modified on 2021-02-06 at 18:41:58  size 0         rw-r--r--",
				time.Date(2021, 2, 6, 18, 41, 58, 0, time.Local), 0},
			Right: Content{File, Modified, "modified on 2021-02-06 at 18:41:58  size 1146      rw-rw-r--",
				time.Date(2021, 2, 6, 18, 41, 58, 0, time.Local), 1146},
			Action: RightToLeft,
		},
		{
			Path: "six/ten",
			Left: Content{File, PropsChanged, "modified on 2021-02-06 at 18:41:58  size 1000      rwx------",
				time.Date(2021, 2, 6, 18, 41, 58, 0, time.Local), 1000},
			Right: Content{File, Unchanged, "modified on 2021-02-06 at 18:41:58  size 1000      rw-r--r--",
				time.Date(2021, 2, 6, 18, 41, 58, 0, time.Local), 1000},
			Action: LeftToRight,
		},
		{
			Path: "three",
			Left: Content{File, Unchanged, "modified on 2021-02-06 at 18:41:58  size 1147000   rw-r--r--",
				time.Date(2021, 2, 6, 18, 41, 58, 0, time.Local), 1147000},
			Right:  Content{Type: Absent, Status: Deleted},
			Action: RightToLeft,
		},
		{
			Path: "twelve",
			Left: Content{Directory, Unchanged, "modified on 2021-02-06 at 18:41:58  size 0         rwxr-xr-x",
				time.Date(2021, 2, 6, 18, 41, 58, 0, time.Local), 0},
			Right: Content{Directory, PropsChanged, "modified on 2021-02-06 at 18:41:58  size 0         rwx------",
				time.Date(2021, 2, 6, 18, 41, 58, 0, time.Local), 0},
			Action: RightToLeft,
		},
		{
			Path: "twenty",
			Left: Content{Type: Absent},
			Right: Content{Directory, Created, "modified on 2021-02-06 at 18:41:58  size 0         rwxr-xr-x",
				time.Date(2021, 2, 6, 18, 41, 58, 0, time.Local), 0},
			Action: RightToLeft,
		},
		{
			Path: "two",
			Left: Content{File, Modified, "modified on 2021-02-06 at 18:41:58  size 1146      rw-r--r--",
				time.Date(2021, 2, 6, 18, 41, 58, 0, time.Local), 1146},
			Right: Content{File, Unchanged, "modified on 2021-02-06 at 18:41:58  size 1146      rw-r--r--",
				time.Date(2021, 2, 6, 18, 41, 58, 0, time.Local), 1146},
			Action: LeftToRight,
		},
	})
	assertEqual(t, c.Plan, map[string]Action{
		"one hundred/one hundred one":           Skip,
		"one hundred/one hundred two":           Skip,
		"six/nine":                              Skip,
		"twenty one":                            Skip,
		"deeply/nested/sub/directory/with/file": RightToLeft,
		"eighteen":                              LeftToRight,
		"here is a rather long and funny file name, 社會科學院語學研究所\t\v\f \u0085 \u1680\u2002\u2003\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u200b\u2028\u2029\u202f\u205f\u3000ﾟ･✿ヾ╲(｡◕‿◕｡)╱✿･ﾟ/here is a rather long and funny file name, 社會科學院語學研究所\t\v\f \u0085 \u1680\u2002\u2003\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u200b\u2028\u2029\u202f\u205f\u3000ﾟ･✿ヾ╲(｡◕‿◕｡)╱✿･ﾟ": RightToLeft,
		"seventeen":    LeftToRight,
		"six/eight":    RightToLeft,
		"six/eleven":   RightToLeft,
		"six/fourteen": RightToLeft,
		"six/seven":    RightToLeft,
		"six/ten":      LeftToRight,
		"three":        RightToLeft,
		"twelve":       RightToLeft,
		"twenty":       RightToLeft,
		"two":          LeftToRight,
	})

	c.Plan["one hundred/one hundred one"] = LeftToRight
	c.Plan["six/nine"] = RightToLeft
	c.Plan["twenty one"] = RightToLeft
	c.Plan["here is a rather long and funny file name, 社會科學院語學研究所\t\v\f \u0085 \u1680\u2002\u2003\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u200b\u2028\u2029\u202f\u205f\u3000ﾟ･✿ヾ╲(｡◕‿◕｡)╱✿･ﾟ/here is a rather long and funny file name, 社會科學院語學研究所\t\v\f \u0085 \u1680\u2002\u2003\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u200b\u2028\u2029\u202f\u205f\u3000ﾟ･✿ヾ╲(｡◕‿◕｡)╱✿･ﾟ"] = LeftToRight
	c.Plan["twelve"] = Skip
	c.Plan["two"] = Merge

	assertEqual(t, c.Sync(),
		Update{Input: []byte("0\n")})
	assertEqual(t, c.Status, "Starting synchronization")
	assertEqual(t, c.ProcOutput([]byte("changed  <-?-> new dir    one hundred/one hundred one  [] ")),
		Update{Input: []byte(">\n")})
	assert.Zero(t, c.ProcOutput([]byte("changed  ====> new dir    one hundred/one hundred one  \n")))
	assertEqual(t, c.ProcOutput([]byte("new file <-?-> deleted    one hundred/one hundred two  [] ")),
		Update{Input: []byte("/\n")})
	assert.Zero(t, c.ProcOutput([]byte("new file <-?-> deleted    one hundred/one hundred two  \n")))
	assertEqual(t, c.ProcOutput([]byte("changed  <-?-> changed    six/nine  [] ")),
		Update{Input: []byte("<\n")})
	assert.Zero(t, c.ProcOutput([]byte("changed  <==== changed    six/nine  \n")))
	assertEqual(t, c.ProcOutput([]byte("props    <-?-> props      twenty one  [] ")),
		Update{Input: []byte("<\n")})
	assert.Zero(t, c.ProcOutput([]byte("props    <==== props      twenty one  \n")))
	assertEqual(t, c.ProcOutput([]byte("         <---- changed    deeply/nested/sub/directory/with/file  [f] ")),
		Update{Input: []byte("<\n")})
	assert.Zero(t, c.ProcOutput([]byte("         <---- changed    deeply/nested/sub/directory/with/file  \n")))
	assertEqual(t, c.ProcOutput([]byte("new link ---->            eighteen  [f] ")),
		Update{Input: []byte(">\n")})
	assert.Zero(t, c.ProcOutput([]byte("new link ---->            eighteen  \n")))
	assertEqual(t, c.ProcOutput([]byte("         <---- changed    here is a rather long and funny file name, 社會科學院語學研究所\t\v\f \u0085 \u1680\u2002\u2003\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u200b\u2028\u2029\u202f\u205f\u3000ﾟ･✿ヾ╲(｡◕‿◕｡)╱✿･ﾟ/here is a rather long and funny file name, 社會科學院語學研究所\t\v\f \u0085 \u1680\u2002\u2003\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u200b\u2028\u2029\u202f\u205f\u3000ﾟ･✿ヾ╲(｡◕‿◕｡)╱✿･ﾟ  [f] ")),
		Update{Input: []byte(">\n")})
	assert.Zero(t, c.ProcOutput([]byte("         ====> changed    here is a rather long and funny file name, 社會科學院語學研究所\t\v\f \u0085 \u1680\u2002\u2003\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u200b\u2028\u2029\u202f\u205f\u3000ﾟ･✿ヾ╲(｡◕‿◕｡)╱✿･ﾟ/here is a rather long and funny file name, 社會科學院語學研究所\t\v\f \u0085 \u1680\u2002\u2003\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u200b\u2028\u2029\u202f\u205f\u3000ﾟ･✿ヾ╲(｡◕‿◕｡)╱✿･ﾟ  \n")))
	assertEqual(t, c.ProcOutput([]byte("new file ---->            seventeen  [f] ")),
		Update{Input: []byte(">\n")})
	assert.Zero(t, c.ProcOutput([]byte("new file ---->            seventeen  \n")))
	assertEqual(t, c.ProcOutput([]byte("         <---- changed    six/eight  [f] ")),
		Update{Input: []byte("<\n")})
	assert.Zero(t, c.ProcOutput([]byte("         <---- changed    six/eight  \n")))
	assertEqual(t, c.ProcOutput([]byte("         <---- changed    six/eleven  [f] ")),
		Update{Input: []byte("<\n")})
	assert.Zero(t, c.ProcOutput([]byte("         <---- changed    six/eleven  \n")))
	assertEqual(t, c.ProcOutput([]byte("         <---- deleted    six/fourteen  [f] ")),
		Update{Input: []byte("<\n")})
	assert.Zero(t, c.ProcOutput([]byte("         <---- deleted    six/fourteen  \n")))
	assertEqual(t, c.ProcOutput([]byte("         <---- changed    six/seven  [f] ")),
		Update{Input: []byte("<\n")})
	assert.Zero(t, c.ProcOutput([]byte("         <---- changed    six/seven  \n")))
	assertEqual(t, c.ProcOutput([]byte("props    ---->            six/ten  [f] ")),
		Update{Input: []byte(">\n")})
	assert.Zero(t, c.ProcOutput([]byte("props    ---->            six/ten  \n")))
	assertEqual(t, c.ProcOutput([]byte("         <---- deleted    three  [f] ")),
		Update{Input: []byte("<\n")})
	assert.Zero(t, c.ProcOutput([]byte("         <---- deleted    three  \n")))
	assertEqual(t, c.ProcOutput([]byte("         <---- props      twelve  [f] ")),
		Update{Input: []byte("/\n")})
	assert.Zero(t, c.ProcOutput([]byte("         <=?=> props      twelve  \n")))
	assertEqual(t, c.ProcOutput([]byte("         <---- new dir    twenty  [f] ")),
		Update{Input: []byte("<\n")})
	assert.Zero(t, c.ProcOutput([]byte("         <---- new dir    twenty  \n")))
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            two  [f] ")),
		Update{Input: []byte("m\n")})
	assert.Zero(t, c.ProcOutput([]byte("changed  <=M=>            two  \n")))
	assertEqual(t, c.ProcOutput([]byte("\nProceed with propagating updates? [] ")),
		Update{Input: []byte("y\n")})

	assert.Zero(t, c.ProcOutput([]byte("Propagating updates\n")))
	assert.Zero(t, c.ProcOutput([]byte("\n\nUNISON 2.51.3 (OCAML 4.11.1) started propagating changes at 16:24:22.00 on 17 Feb 2021\n")))
	assert.Zero(t, c.ProcOutput([]byte("[BGN] Copying one hundred/one hundred one from /home/vasiliy/tmp/gunison/left to //tanais//home/vasiliy/tmp/gunison/right\n")))
	assert.Zero(t, c.ProcOutput([]byte("[CONFLICT] Skipping one hundred/one hundred two\n  conflicting updates\n")))
	assert.Zero(t, c.ProcOutput([]byte("[BGN] Updating file six/nine from //tanais//home/vasiliy/tmp/gunison/right to /home/vasiliy/tmp/gunison/left\n")))
	assert.Zero(t, c.ProcOutput([]byte("[BGN] Copying properties for twenty one from //tanais//home/vasiliy/tmp/gunison/right to /home/vasiliy/tmp/gunison/left\n")))
	assert.Zero(t, c.ProcOutput([]byte("[BGN] Updating file deeply/nested/sub/directory/with/file from //tanais//home/vasiliy/tmp/gunison/right to /home/vasiliy/tmp/gunison/left\n")))
	assert.Zero(t, c.ProcOutput([]byte("[BGN] Copying eighteen from /home/vasiliy/tmp/gunison/left to //tanais//home/vasiliy/tmp/gunison/right\n")))
	assert.Zero(t, c.ProcOutput([]byte("[BGN] Updating file here is a rather long and funny file name, 社會科學院語學研究所\t\v\f \u0085 \u1680\u2002\u2003\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u200b\u2028\u2029\u202f\u205f\u3000ﾟ･✿ヾ╲(｡◕‿◕｡)╱✿･ﾟ/here is a rather long and funny file name, 社會科學院語學研究所\t\v\f \u0085 \u1680\u2002\u2003\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u200b\u2028\u2029\u202f\u205f\u3000ﾟ･✿ヾ╲(｡◕‿◕｡)╱✿･ﾟ from /home/vasiliy/tmp/gunison/left to //tanais//home/vasiliy/tmp/gunison/right\n")))
	assert.Zero(t, c.ProcOutput([]byte("[BGN] Copying seventeen from /home/vasiliy/tmp/gunison/left to //tanais//home/vasiliy/tmp/gunison/right\n")))
	assert.Zero(t, c.ProcOutput([]byte("[BGN] Updating file six/eight from //tanais//home/vasiliy/tmp/gunison/right to /home/vasiliy/tmp/gunison/left\n")))
	assert.Zero(t, c.ProcOutput([]byte("[BGN] Updating file six/eleven from //tanais//home/vasiliy/tmp/gunison/right to /home/vasiliy/tmp/gunison/left\n")))
	assert.Zero(t, c.ProcOutput([]byte("[BGN] Updating file six/seven from //tanais//home/vasiliy/tmp/gunison/right to /home/vasiliy/tmp/gunison/left\n")))
	assert.Zero(t, c.ProcOutput([]byte("[BGN] Copying properties for six/ten from /home/vasiliy/tmp/gunison/left to //tanais//home/vasiliy/tmp/gunison/right\n")))
	assert.Zero(t, c.ProcOutput([]byte("[CONFLICT] Skipping twelve\n  skip requested\n")))
	assert.Zero(t, c.ProcOutput([]byte("[BGN] Copying twenty from //tanais//home/vasiliy/tmp/gunison/right to /home/vasiliy/tmp/gunison/left\n")))
	assert.Zero(t, c.ProcOutput([]byte("  0%  00:55 ETA")))
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))
	assert.Zero(t, c.ProcOutput([]byte("[END] Copying properties for twenty one\n")))
	assert.Zero(t, c.ProcOutput([]byte("  0%  00:55 ETA")))
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))
	assert.Zero(t, c.ProcOutput([]byte("  0%  00:04 ETA")))
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))
	assert.Zero(t, c.ProcOutput([]byte("  1%  00:02 ETA")))
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))
	assert.Zero(t, c.ProcOutput([]byte("  2%  00:01 ETA")))
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))
	assert.Zero(t, c.ProcOutput([]byte("  3%  00:01 ETA")))
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))
	assert.Zero(t, c.ProcOutput([]byte("  4%  00:01 ETA")))
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))
	assert.Zero(t, c.ProcOutput([]byte("  5%  00:01 ETA")))
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))
	assert.Zero(t, c.ProcOutput([]byte("  6%  00:01 ETA")))
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))
	assert.Zero(t, c.ProcOutput([]byte("  7%  00:01 ETA")))
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))
	assert.Zero(t, c.ProcOutput([]byte("  8%  00:00 ETA")))
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))
	assert.Zero(t, c.ProcOutput([]byte("  9%  00:00 ETA")))
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))
	assert.Zero(t, c.ProcOutput([]byte("Shortcut: copied /home/vasiliy/tmp/gunison/left/six/eight from local file /home/vasiliy/tmp/gunison/left/three\n")))
	assert.Zero(t, c.ProcOutput([]byte("  9%  00:00 ETA")))
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))
	assert.Zero(t, c.ProcOutput([]byte("  9%  00:02 ETA")))
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))
	assert.Zero(t, c.ProcOutput([]byte("Shortcut: copied /home/vasiliy/tmp/gunison/left/six/seven from local file /home/vasiliy/tmp/gunison/left/six/fourteen/sixteen\n")))
	assert.Zero(t, c.ProcOutput([]byte("  9%  00:02 ETA")))
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))
	assert.Zero(t, c.ProcOutput([]byte("[END] Copying properties for six/ten\n")))
	assert.Zero(t, c.ProcOutput([]byte("  9%  00:02 ETA")))
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))
	assert.Zero(t, c.ProcOutput([]byte("Shortcut: copied /home/vasiliy/tmp/gunison/left/two from local file /home/vasiliy/tmp/gunison/left/six/.unison.seven.1e1fb20baa490c92a38dae56142181e1.unison.tmp\n")))
	assert.Zero(t, c.ProcOutput([]byte("  9%  00:02 ETA")))
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))
	assert.Zero(t, c.ProcOutput([]byte(" 18%  00:01 ETA")))
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))
	assert.Zero(t, c.ProcOutput([]byte("Shortcut: copied /home/vasiliy/tmp/gunison/right/one hundred/one hundred one from local file /home/vasiliy/tmp/gunison/right/six/ten\n")))
	assert.Zero(t, c.ProcOutput([]byte(" 18%  00:01 ETA")))
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))
	assert.Zero(t, c.ProcOutput([]byte("Shortcut: copied /home/vasiliy/tmp/gunison/right/here is a rather long and funny file name, 社會科學院語學研究所\t\v\f \u0085 \u1680\u2002\u2003\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u200b\u2028\u2029\u202f\u205f\u3000ﾟ･✿ヾ╲(｡◕‿◕｡)╱✿･ﾟ/here is a rather long and funny file name, 社會科學院語學研究所\t\v\f \u0085 \u1680\u2002\u2003\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u200b\u2028\u2029\u202f\u205f\u3000ﾟ･✿ヾ╲(｡◕‿◕｡)╱✿･ﾟ from local file /home/vasiliy/tmp/gunison/right/two\n")))
	assert.Zero(t, c.ProcOutput([]byte(" 18%  00:01 ETA")))
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))
	assert.Zero(t, c.ProcOutput([]byte("Shortcut: copied /home/vasiliy/tmp/gunison/right/seventeen from local file /home/vasiliy/tmp/gunison/right/one\n")))
	assert.Zero(t, c.ProcOutput([]byte(" 18%  00:01 ETA")))
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))
	assert.Zero(t, c.ProcOutput([]byte("100%  00:00 ETA")))
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))
	assert.Zero(t, c.ProcOutput([]byte("[END] Copying twenty\n")))
	assert.Zero(t, c.ProcOutput([]byte("100%  00:00 ETA")))
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))
	assert.Zero(t, c.ProcOutput([]byte("[END] Copying one hundred/one hundred one\n")))
	assert.Zero(t, c.ProcOutput([]byte("100%  00:00 ETA")))
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))
	assert.Zero(t, c.ProcOutput([]byte("[END] Copying eighteen\n")))
	assert.Zero(t, c.ProcOutput([]byte("100%  00:00 ETA")))
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))
	assert.Zero(t, c.ProcOutput([]byte("[END] Updating file here is a rather long and funny file name, 社會科學院語學研究所\t\v\f \u0085 \u1680\u2002\u2003\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u200b\u2028\u2029\u202f\u205f\u3000ﾟ･✿ヾ╲(｡◕‿◕｡)╱✿･ﾟ/here is a rather long and funny file name, 社會科學院語學研究所\t\v\f \u0085 \u1680\u2002\u2003\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u200b\u2028\u2029\u202f\u205f\u3000ﾟ･✿ヾ╲(｡◕‿◕｡)╱✿･ﾟ\n")))
	assert.Zero(t, c.ProcOutput([]byte("100%  00:00 ETA")))
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))
	assert.Zero(t, c.ProcOutput([]byte("[END] Copying seventeen\n")))
	assert.Zero(t, c.ProcOutput([]byte("100%  00:00 ETA")))
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))
	assert.Zero(t, c.ProcOutput([]byte("[END] Updating file six/eight\n")))
	assert.Zero(t, c.ProcOutput([]byte("100%  00:00 ETA")))
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))
	assertEqual(t, c.ProcOutput([]byte("Failed [two]: 'merge' preference not set for two\n")),
		Update{Messages: []Message{
			{"Failed [two]: 'merge' preference not set for two", Error},
		}})
	assert.Zero(t, c.ProcOutput([]byte("[END] Updating file six/seven\n")))
	assert.Zero(t, c.ProcOutput([]byte("[END] Updating file six/nine\n")))
	assert.Zero(t, c.ProcOutput([]byte("[END] Updating file deeply/nested/sub/directory/with/file\n")))
	assert.Zero(t, c.ProcOutput([]byte("[END] Updating file six/eleven\n")))
	assert.Zero(t, c.ProcOutput([]byte("[BGN] Deleting six/fourteen from /home/vasiliy/tmp/gunison/left\n")))
	assert.Zero(t, c.ProcOutput([]byte("[BGN] Deleting three from /home/vasiliy/tmp/gunison/left\n")))
	assert.Zero(t, c.ProcOutput([]byte("[END] Deleting six/fourteen\n")))
	assert.Zero(t, c.ProcOutput([]byte("[END] Deleting three\n")))
	assert.Zero(t, c.ProcOutput([]byte("UNISON 2.51.3 (OCAML 4.11.1) finished propagating changes at 16:24:22.84 on 17 Feb 2021\n\n\n")))
	assert.Zero(t, c.ProcOutput([]byte("Saving synchronizer state\n")))
	assertEqual(t, c.ProcOutput([]byte("Synchronization incomplete at 16:24:22  (14 items transferred, 2 skipped, 1 failed)\n")),
		Update{Messages: []Message{
			{"Synchronization incomplete at 16:24:22  (14 items transferred, 2 skipped, 1 failed)", Warning},
		}})
	assertEqual(t, c.ProcOutput([]byte("  skipped: one hundred/one hundred two (conflicting updates)\n")),
		Update{Messages: []Message{
			{"skipped: one hundred/one hundred two (conflicting updates)", Info},
		}})
	assertEqual(t, c.ProcOutput([]byte("  skipped: twelve (skip requested)\n")),
		Update{Messages: []Message{
			{"skipped: twelve (skip requested)", Info},
		}})
	assertEqual(t, c.ProcOutput([]byte("  failed: two\n")),
		Update{Messages: []Message{
			{"failed: two", Error},
		}})
	assert.Zero(t, c.ProcExit(2, nil))
	assertEqual(t, c.Status, "Finished with errors")
}

func TestAssortedRandom(t *testing.T) {
	// Like TestAssorted, but chunks of Unison output get buffered randomly before arriving to Gunison.
	c := NewCore()
	assert.Zero(t, c.ProcStart())
	assertEqual(t, c.ProcOutput([]byte("Unison 2.51.3 (ocaml 4.11.1): Contacting server...\nLooking for changes\n\\ four\r      \r\\ six/fourteen\r              \rReconciling changes\n\nleft           right              \nchanged  <-?-> new dir    one hundred/one hundred one  [] ")),
		Update{
			Progressed: true,
			Input:      []byte("l\n"),
		})
	assertEqual(t, c.Status, "Assembling plan")
	assert.Zero(t, c.ProcOutput([]byte("  changed  <-?-> new dir    one hundred/one hundred one  \nleft         : changed file  ")))
	assert.Zero(t, c.ProcOutput([]byte("     modified on 2021-02-26 at 15:42:40  size 1")))
	assert.Zero(t, c.ProcOutput([]byte("000      rw-r--r--\nright        : new dir            modifi")))
	assert.Zero(t, c.ProcOutput([]byte("ed on 2021-02-26 at 15:42:40  size 2292      rwxr-xr-x\n  new file <-?-> deleted    one hundred/one hundred two  \nleft         : new file           modified on 2021-02-26 at 15:42:40")))
	assert.Zero(t, c.ProcOutput([]byte("  size 1146      r")))
	assert.Zero(t, c.ProcOutput([]byte("w-r--r--\nright        : d")))
	assert.Zero(t, c.ProcOutput([]byte("eleted\n  changed  <-?-> changed    six/nine  \nleft         : changed file       modified on 2021-02-26 at 15:42:40  size 1147000   rw-r--r--")))
	assert.Zero(t, c.ProcOutput([]byte("\nright        : changed file       modified on 2021-02-26 at 15:42:40  size 1147000   rw-r--r--\n           <---- changed    deeply/nested/sub/directory/with/file  \nleft         : unchanged file     modified on 2021-02-26 at 15:42:40  size 1146      rw-r--r")))
	assert.Zero(t, c.ProcOutput([]byte("")))
	assert.Zero(t, c.ProcOutput([]byte("--\nright        : changed file       modified on 2021-02-26 at 15:42:40  size 1146      rw-r--r--\n  new link ---->            eighteen  \nleft         : new symlink        modified on 1970-01-01 ")))
	assert.Zero(t, c.ProcOutput([]byte("at  3:00:00  size 0         unknown permissions\nright        : absent\n           <---- changed    here is a rather long and funny file name, 社會科學院語學研究所\t\v\f \u0085 \u1680\u2002\u2003\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009")))
	assert.Zero(t, c.ProcOutput([]byte("\u200a\u200b\u2028\u2029\u202f\u205f\u3000ﾟ･✿ヾ╲(｡◕‿◕｡)╱✿･ﾟ/here is a rather long and funny file name, 社會科學院語學研究所\t\v\f \u0085 \u1680\u2002\u2003\u2002\u2003\xe2")))
	assert.Zero(t, c.ProcOutput([]byte("\x80\x84\u2005\u2006\u2007\u2008\u2009\u200a\u200b\u2028\u2029\u202f\u205f\u3000ﾟ･✿ヾ╲(｡◕‿◕｡)╱✿･ﾟ  \nleft")))
	assert.Zero(t, c.ProcOutput([]byte("         : unchanged file   ")))
	assert.Zero(t, c.ProcOutput([]byte("  modified on 2021-02-26 at 15:42:40  size 1146      rw-r--r--\nright      ")))
	assert.Zero(t, c.ProcOutput([]byte("  : changed file       modified on 2021-02-26 at 15:42:40  size 1146      rw-r--r--\n  new file ---->           ")))
	assert.Zero(t, c.ProcOutput([]byte(" seventeen  \nleft         : new file           modified on 2021-02-26 at 15:42:40  size 0         rw-r--r--\nright        : absent\n           <---- changed    six/eight  \nleft         : unchanged file     modified on 2021-02-26 at 15:42:40  size ")))
	assert.Zero(t, c.ProcOutput([]byte("1146      rw-r--r--\nright        : changed file       modified on 2021-02-26 at 15:42:40  size 1147000   rw-r--r--\n           <---- chang")))
	assert.Zero(t, c.ProcOutput([]byte("ed    six/eleven  \nleft         : unchanged file     modified on 2021-02-26 at 15:42:40  size 10000000  rw-r--r--\nright        : changed file       modified on 2021-02-26 at 15:42:40  size 10000000  rw-r--r")))
	assert.Zero(t, c.ProcOutput([]byte("--\n           <---- deleted    six/fourteen  \nleft         : unchanged dir      modified on 202")))
	assert.Zero(t, c.ProcOutput([]byte("1-02-26 at 15:42:40  size 2292      rwxr-xr-x\nright        : delet")))
	assert.Zero(t, c.ProcOutput([]byte("ed\n           <---- chgd lnk   six/funny name!  \nleft         : unchanged symlink  modified on 1970-01-01 at  3:00:00  size 0   ")))
	assert.Zero(t, c.ProcOutput([]byte("      unknown permissions\nright        : changed symlink  ")))
	assert.Zero(t, c.ProcOutput([]byte("  modified on 1970-01-01 at  3:00:00  size 0         unknown permissions\n           <---- changed    six/seven  \nleft         : unchanged file     ")))
	assert.Zero(t, c.ProcOutput([]byte("modified on 2021-02-26 at 15:42:40  size 0         rw-r--r--\nright        : changed file       modified on 2021-02-26 at 15:42:40  size 1146      rw-r--r--\n  props    ---->            six/ten  \nleft         : changed props      modified on 2021-02")))
	assert.Zero(t, c.ProcOutput([]byte("-26 at 15:42:40  size 1000      rwx------\nright        : unchanged file     modified on 2021-02-26 at 15:42:40  size 1000      rw-r--r--\n           <---- deleted    three  \nleft         :")))
	assert.Zero(t, c.ProcOutput([]byte(" unchanged file     modified on 2021-02-26 at 15:42:40  size 1147000   rw-r--r--\nright  ")))
	assert.Zero(t, c.ProcOutput([]byte("      : deleted\n           <---- props      twelve  \nleft         : unchanged dir      mod")))
	assert.Zero(t, c.ProcOutput([]byte("ified on 2021-0")))
	assert.Zero(t, c.ProcOutput([]byte("2-26 at 15:42:40  size 0         rwxr-xr-x\nright        : dir props changed  modified on 2021-02-26 at 15:42:40  size 0         rwx------\n           <---- new dir    twenty  \nleft         : absent\nright        : new dir            modified o")))
	assert.Zero(t, c.ProcOutput([]byte("n 2021-02-26 at 15:42:40  size 0         rwxr-xr-x\n  changed  ---->            two  \nleft         : changed ")))
	assert.Zero(t, c.ProcOutput([]byte("file       modified on 2021-02-26 at 15:42:40  size 1146      rw-r--r--\nright        : unchanged file     modified on 2021-02-26 at 15:42:40  size 1146      rw-r--r--\nchanged  <-?-> new dir    one hundred/one hundred one  [] ")))

	assertEqual(t, c.Plan, map[string]Action{
		"one hundred/one hundred one":           Skip,
		"one hundred/one hundred two":           Skip,
		"six/nine":                              Skip,
		"deeply/nested/sub/directory/with/file": RightToLeft,
		"eighteen":                              LeftToRight,
		"here is a rather long and funny file name, 社會科學院語學研究所\t\v\f \u0085 \u1680\u2002\u2003\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u200b\u2028\u2029\u202f\u205f\u3000ﾟ･✿ヾ╲(｡◕‿◕｡)╱✿･ﾟ/here is a rather long and funny file name, 社會科學院語學研究所\t\v\f \u0085 \u1680\u2002\u2003\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u200b\u2028\u2029\u202f\u205f\u3000ﾟ･✿ヾ╲(｡◕‿◕｡)╱✿･ﾟ": RightToLeft,
		"seventeen":       LeftToRight,
		"six/eight":       RightToLeft,
		"six/eleven":      RightToLeft,
		"six/fourteen":    RightToLeft,
		"six/funny name!": RightToLeft,
		"six/seven":       RightToLeft,
		"six/ten":         LeftToRight,
		"three":           RightToLeft,
		"twelve":          RightToLeft,
		"twenty":          RightToLeft,
		"two":             LeftToRight,
	})

	c.Plan["one hundred/one hundred one"] = LeftToRight
	c.Plan["six/nine"] = RightToLeft
	c.Plan["here is a rather long and funny file name, 社會科學院語學研究所\t\v\f \u0085 \u1680\u2002\u2003\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u200b\u2028\u2029\u202f\u205f\u3000ﾟ･✿ヾ╲(｡◕‿◕｡)╱✿･ﾟ/here is a rather long and funny file name, 社會科學院語學研究所\t\v\f \u0085 \u1680\u2002\u2003\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u200b\u2028\u2029\u202f\u205f\u3000ﾟ･✿ヾ╲(｡◕‿◕｡)╱✿･ﾟ"] = LeftToRight
	c.Plan["twelve"] = Skip
	c.Plan["two"] = Merge

	assertEqual(t, c.Sync(),
		Update{Input: []byte("0\n")})
	assertEqual(t, c.ProcOutput([]byte("changed  <-?-> new dir    one hundred/one hundred one  [] ")),
		Update{Input: []byte(">\n")})
	assertEqual(t, c.ProcOutput([]byte("changed  ====> new dir    one hundred/one hundred one  \nnew file <-?-> deleted    one hundred/one hundred two  [] ")),
		Update{Input: []byte("/\n")})
	assert.Zero(t, c.ProcOutput([]byte("new file <-?-> deleted    one hundred/one hundred two  \n")))
	assertEqual(t, c.ProcOutput([]byte("changed  <-?-> changed    six/nine  [] ")),
		Update{Input: []byte("<\n")})
	assert.Zero(t, c.ProcOutput([]byte("changed  <==== changed    six/n")))
	assertEqual(t, c.ProcOutput([]byte("ine  \n         <---- changed    deeply/nested/sub/directory/with/file  [f] ")),
		Update{Input: []byte("<\n")})
	assertEqual(t, c.ProcOutput([]byte("         <---- changed    deeply/nested/sub/directory/with/file  \nnew link ---->            eighteen  [f] ")),
		Update{Input: []byte(">\n")})
	assert.Zero(t, c.ProcOutput([]byte("new link ----")))
	assert.Zero(t, c.ProcOutput([]byte(">            eighteen  \n         <---- changed    here is a rather long and funny file name, 社會科學院語學研究所\t\v\f \u0085 \u1680\u2002\u2003\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u200b\u2028\u2029\u202f\u205f\u3000ﾟ･")))
	assert.Zero(t, c.ProcOutput([]byte("✿ヾ╲(｡◕‿◕｡)╱✿･ﾟ/here is a rather long and funny file name, 社會科學")))
	assert.Zero(t, c.ProcOutput([]byte("院語學研究所\t\v\f \u0085 \u1680\u2002\u2003\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\xe2\x80")))
	assertEqual(t, c.ProcOutput([]byte("\x8b\u2028\u2029\u202f\u205f\u3000ﾟ･✿ヾ╲(｡◕‿◕｡)╱✿･ﾟ  [f] ")),
		Update{Input: []byte(">\n")})
	assert.Zero(t, c.ProcOutput([]byte("         ====> changed    here is a rather long and funny file name, 社會科學院語學研究所\t\v\f \u0085 \u1680\u2002\u2003\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u200b\xe2\x80")))
	assert.Zero(t, c.ProcOutput([]byte("\xa8\u2029\u202f\u205f\u3000ﾟ･✿ヾ╲(｡◕‿◕｡)╱✿･ﾟ/here is a rather long and funny file name, 社會科學院語學研究所\t\v\f \u0085 \u1680\u2002\u2003\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u200b\xe2")))
	assertEqual(t, c.ProcOutput([]byte("\x80\xa8\u2029\u202f\u205f\u3000ﾟ･✿ヾ╲(｡◕‿◕｡)╱✿･ﾟ  \nnew file ---->            seventeen  [f] ")),
		Update{Input: []byte(">\n")})
	assertEqual(t, c.ProcOutput([]byte("new file ---->            seventeen  \n         <---- changed    six/eight  [f] ")),
		Update{Input: []byte("<\n")})
	assertEqual(t, c.ProcOutput([]byte("         <---- changed    six/eight  \n         <---- changed    six/eleven  [f] ")),
		Update{Input: []byte("<\n")})
	assert.Zero(t, c.ProcOutput([]byte("         <---- changed    six/eleven  \n         <---- del")))
	assertEqual(t, c.ProcOutput([]byte("eted    six/fourteen  [f] ")),
		Update{Input: []byte("<\n")})
	assertEqual(t, c.ProcOutput([]byte("         <---- deleted    six/fourteen  \n         <---- chgd lnk   six/funny name!  [f] ")),
		Update{Input: []byte("<\n")})
	assertEqual(t, c.ProcOutput([]byte("         <---- chgd lnk   six/funny name!  \n         <---- changed    six/seven  [f] ")),
		Update{Input: []byte("<\n")})
	assertEqual(t, c.ProcOutput([]byte("         <---- changed    six/seven  \nprops    ---->            six/ten  [f] ")),
		Update{Input: []byte(">\n")})
	assertEqual(t, c.ProcOutput([]byte("props    ---->            six/ten  \n         <---- deleted    three  [f] ")),
		Update{Input: []byte("<\n")})
	assertEqual(t, c.ProcOutput([]byte("         <---- deleted    three  \n         <---- props      twelve  [f] ")),
		Update{Input: []byte("/\n")})
	assertEqual(t, c.ProcOutput([]byte("         <=?=> props      twelve  \n         <---- new dir    twenty  [f] ")),
		Update{Input: []byte("<\n")})
	assertEqual(t, c.ProcOutput([]byte("         <---- new dir    twenty  \nchanged  ---->            two  [f] ")),
		Update{Input: []byte("m\n")})
	assertEqual(t, c.ProcOutput([]byte("changed  <=M=>            two  \n\nProceed with propagating updates? [] ")),
		Update{Input: []byte("y\n")})

	assert.Zero(t, c.ProcOutput([]byte("Propagating updates\n\n\nUNISON 2.51.3 (OCAML 4.11.1) started propagating changes at 15:44:20.69 on 26 Feb 2021\n[BGN] Copying one hundred/one hundred one from /home/vasiliy/tmp/gunison/left to /home/vasiliy/tmp/gunison/right\n  0%  00:52 ETA\r             ")))
	assert.Zero(t, c.ProcOutput([]byte("  \r[END] Copying one hundred/one hundred one\n  0%  00:52 ETA\r               \r[CONFLICT] Skipping one hundred/one hundred two\n  conflicting updates\n  0%  00:52 ETA\r               \r[BGN] Updating file six/nine fr")))
	assert.Zero(t, c.ProcOutput([]byte("om /home/vasiliy/tmp/gunison/right to /home/vasiliy/tmp/gunison/left\n  0%  00:52 ETA\r               \r  0%")))
	assert.Zero(t, c.ProcOutput([]byte("  00:02 ETA\r               \r  1%  00:01 ETA\r               \r  2%  00:01 ETA\r               \r  3%  00:01 ETA\r               \r  4%  00:00 ETA\r               \r")))
	assert.Zero(t, c.ProcOutput([]byte("  5%  00:00 ETA\r               \r  6%  00:00 ETA\r               \r  7%  00:00 ETA\r               \r  8%  00:00 ETA\r               \r  9%  00:00 ETA\r               \r[END] ")))
	assert.Zero(t, c.ProcOutput([]byte("Updating file six/nine\n  9%  00:00 ETA\r               \r[BGN] Updating file deeply/nested/sub/directory/with/file from /home/vasi")))
	assert.Zero(t, c.ProcOutput([]byte("liy/tmp/gunison/right to /home/vasiliy/tmp/gunison/left\n  9%  00:00 ETA\r               \r[END] Updating file deeply/nested/sub/directory/with/file\n  9%  00:00 ETA")))
	assert.Zero(t, c.ProcOutput([]byte("\r ")))
	assert.Zero(t, c.ProcOutput([]byte("              \r[BGN] Copying eighteen from /home/vasiliy/tmp/gunison/left to /home/")))
	assert.Zero(t, c.ProcOutput([]byte("vasiliy/tmp/gunison/right\n  9%  00:00 ETA\r               \r[END] Copying eighteen\n  9%  00:00 ETA\r               \r[BGN] Updating file here is a rat")))
	assert.Zero(t, c.ProcOutput([]byte("her long and funny file name, 社會科學院語學研究所\t\v\f \u0085 \u1680\u2002\u2003\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u200b\u2028\u2029\u202f\u205f\u3000ﾟ･✿ヾ╲(｡◕‿◕｡)╱✿･\xef")))
	assert.Zero(t, c.ProcOutput([]byte("\xbe\x9f/here is a rather long and funny file name, 社會科學院語學研究所\t\v\f \u0085 \u1680\u2002\u2003\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u200b\u2028\u2029\u202f\u205f\u3000ﾟ･✿ヾ╲(｡◕‿◕｡)╱✿･ﾟ from /home/vasiliy/tmp/gunison/left to /home/vasiliy/tmp/gunison/right\n  9%  00:00 ETA\r       ")))
	assert.Zero(t, c.ProcOutput([]byte("  ")))
	assert.Zero(t, c.ProcOutput([]byte("      \r[END] Updating file here is a rather long and funny file name, 社會科學院語學研究所\t\v\f \u0085 \u1680\u2002\u2003\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u200b\u2028\u2029\u202f\u205f\u3000ﾟ･✿ヾ╲(｡◕‿◕｡)╱✿･ﾟ/here is a ra")))
	assert.Zero(t, c.ProcOutput([]byte("ther long and funny file name, 社會科學院語學研究所\t\v\f \u0085 \u1680\u2002\u2003\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u200b\u2028\u2029\u202f\u205f\u3000ﾟ･✿ヾ╲(｡◕‿"))) //nolint:misspell
	assert.Zero(t, c.ProcOutput([]byte("◕｡)╱✿･ﾟ\n  9%  00:00 ETA\r               \r[BGN] Copying seventeen from /home/vasiliy/tmp/gunison/left to /home/vasiliy/tmp/gunison/right\n  9%  00:00 ETA\r               \r[END] Copying seventeen\n  9%  00:00 ETA\r               \r[BGN] Updating file six/eight from /home/vasiliy/tmp/guniso")))
	assert.Zero(t, c.ProcOutput([]byte("n/right to /home/vasiliy/tmp/gunison/left\n  9%  00:00 ETA\r               \r 10%  00:00 ETA\r               \r 11%  00:00 ETA\r               \r 12%  00:00 ETA\r               \r 13%  00:00 ETA\r               \r 14%  00:00 ETA\r               \r 15%  00:00 ETA\r               \r 16%  00:00")))
	assert.Zero(t, c.ProcOutput([]byte(" ETA\r               \r 17%  00:00 ETA\r               \r 18%  00:00 ETA\r               \r[END] Updating file six/eight\n 18%  00:00 ETA\r               \r[BGN] Updating f")))
	assert.Zero(t, c.ProcOutput([]byte("ile six/eleven from /home/vasiliy/tmp/gunison/right to /home/vasiliy/tmp/gunison/left\n 18%  00:00 ETA\r               \r 19%  00:00 ETA\r               \r 20%  00:00 ETA\r               \r 21%  00:00 ET")))
	assert.Zero(t, c.ProcOutput([]byte("A\r               \r 2")))
	assert.Zero(t, c.ProcOutput([]byte("2%  00:00 ETA\r               \r 23%  00:00 ETA\r               \r 24%  00:00 ETA\r               \r 25%  00:00 ETA\r             ")))
	assert.Zero(t, c.ProcOutput([]byte("  \r 26%  00:00 ETA\r               \r 27%  00:00 ETA\r               \r 28%  00:00 ETA\r               \r 29%  00:00 ETA\r               \r 30%  00:00 ETA\r      ")))
	assert.Zero(t, c.ProcOutput([]byte("         \r 31%  00:00 ETA\r           ")))
	assert.Zero(t, c.ProcOutput([]byte("    \r 32%  00:00 ETA\r               \r 33%  00:00 ETA\r               \r 34%  00:00 ETA\r               \r 35%  00:00 ETA\r               \r")))
	assert.Zero(t, c.ProcOutput([]byte(" 36%  00:00 ETA\r               \r 37%  00:")))
	assert.Zero(t, c.ProcOutput([]byte("00 ETA\r               \r 38%  00:00 ETA\r               \r 39%  00:00 ETA\r               \r 40%  00:00 ETA\r               \r 41%  00:00 ETA\r               \r 42%  00:00 ETA\r               \r 43%  00:00 ETA\r               \r 44%  00:00 ETA\r               \r 45%  00:00 ")))
	assert.Zero(t, c.ProcOutput([]byte("ETA\r               \r 46%  00:00 ETA\r               \r 47%  00:00 ETA\r               \r 48%  00:00 ETA\r               \r 49%  00:00 ETA\r               \r 50%  00:00 ETA\r               \r 51%  00:00 ETA\r               \r 52%  00:00 ETA\r     ")))
	assert.Zero(t, c.ProcOutput([]byte("          \r 53%  00:00 ETA\r               \r 54%  00:00 ETA\r               \r 55%  00:00 ETA\r               \r 56%  00:00 ETA\r               \r 57%")))
	assert.Zero(t, c.ProcOutput([]byte("  00:00 ETA\r               \r 58%  00:00 ETA\r               \r 59%  00:00 ETA\r               \r 60%  00:00 ETA\r               \r 61%  00:00 ETA\r               \r 62%  00:00 ETA\r               \r 63")))
	assert.Zero(t, c.ProcOutput([]byte("%  00:00 ETA\r               \r 64%  00:00 ETA\r               \r 65%  00:00 ETA\r               \r 66%  00:00 ETA\r               \r 67%  00:00 ETA\r               \r 68%  00:00 ETA\r               \r 69%  00:00 E")))
	assert.Zero(t, c.ProcOutput([]byte("TA\r               \r 70%  00:00 ETA\r               \r 71%  00:00 ETA\r               \r 72%  00:00 ETA\r               \r 73%  00:00 ETA\r               \r 74%  00:00 ETA\r               \r 75%  00:00 ETA\r               \r 76%  00:00 ETA\r               \r 77%  00:00 ETA\r               \r 78")))
	assert.Zero(t, c.ProcOutput([]byte("%  00:00 ETA\r               \r 79%  00:00 ETA\r               \r 80%  00:00 ETA\r               \r 81%  00:00 ETA\r               \r 82%  00:00")))
	assert.Zero(t, c.ProcOutput([]byte(" ETA\r               \r 83%  00:00 ETA\r               \r 84%  00:00 ETA\r               \r 85%  00:00 ETA\r               \r 86%  00:00 ETA\r               \r 87%  00:00 ETA\r               \r 88%  00:00 ETA\r               \r 89%  00:00 ETA\r               \r ")))
	assert.Zero(t, c.ProcOutput([]byte("90%  00")))
	assert.Zero(t, c.ProcOutput([]byte(":00 ETA\r               \r 91%  00:00 ETA\r               \r 92%  00:00 ETA\r               \r 93%  00:00 ETA\r               \r 94%  00:00 ETA\r               \r 95%  00:00 ETA\r               \r 96%  00:00 ETA\r               \r 97%  00:00 ETA\r        ")))
	assert.Zero(t, c.ProcOutput([]byte("       \r 98%  00:00 ETA\r               \r 99%  00:00 ETA\r               \r[END] Updating file six/eleven\n")))
	assert.Zero(t, c.ProcOutput([]byte(" 99%  00:00 ETA\r               \r[BGN] Copying six/funny name! from /home/vasiliy/tmp/gunison/right to /home/vasiliy/tmp/gunison/left\n 99%  00:00 ETA\r               \r[END] Copying six/funny name!\n 99%  00:00 ETA\r               \r[BGN] Updating file six/s")))
	assert.Zero(t, c.ProcOutput([]byte("even from /home/vasiliy/tmp/gunison/right to /home/vasiliy/tmp/gunison/left\n 99%  00:00 ETA\r               \r100%  00:00 ETA\r               \r[END] Updating file six/seven\n100%  00:00 ETA\r               \r[BGN] Copying properties for six/ten from")))
	assert.Zero(t, c.ProcOutput([]byte(" /home/vasiliy/tmp/gunison/left to /home/vasiliy/tmp/gunison/right\n100%  00:00 ETA\r               \r[END] Copying properties for six/ten\n100%  00:00 ETA\r               \r[CONFLICT] Skipping twelve\n  skip req")))
	assert.Zero(t, c.ProcOutput([]byte("uested\n100%  00:00 ETA\r               \r[BGN] Copying twenty from /home/vasiliy/tmp/gunison/right t")))
	assertEqual(t, c.ProcOutput([]byte("o /home/vasiliy/tmp/gunison/left\n100%  00:00 ETA\r               \r[END] Copying twenty\n100%  00:00 ETA\r               \rFailed [two]: 'merge' preference not set for two\n[BGN] Deleting six/fourteen from /home/vasiliy/tmp/gunison")),
		Update{Messages: []Message{
			{"Failed [two]: 'merge' preference not set for two", Error},
		}})
	assert.Zero(t, c.ProcOutput([]byte("/left\n[END] Deleting six/fourteen\n[BGN] Deleting three from /home/vasiliy/tmp/gunison/left\n[END] Deleting three\nUNISON 2.51.3 (OCAML 4.11.1) finished p")))
	assert.Zero(t, c.ProcOutput([]byte("ropagating chan")))
	assertEqual(t, c.ProcOutput([]byte("ges at 15:44:21.21 on 26 Feb 2021\n\n\nSaving synchronizer state\nSynchronization incomplete at 15:44:21  (14 items transferred, 2 skipped, 1 failed)\n  skipped: one hundred/one hundred two (conflicting updates)\n  skipped: twelve (skip requested)\n  failed: two\n")),
		Update{Messages: []Message{
			{"Synchronization incomplete at 15:44:21  (14 items transferred, 2 skipped, 1 failed)", Warning},
			{"skipped: one hundred/one hundred two (conflicting updates)", Info},
			{"skipped: twelve (skip requested)", Info},
			{"failed: two", Error},
		}})
	assertEqual(t, c.Status, "Saving synchronizer state")
	assert.Zero(t, c.ProcExit(2, nil))
	assertEqual(t, c.Status, "Finished with errors")
}

func TestInterruptLookingForChanges(t *testing.T) {
	c := NewCore()
	assert.Zero(t, c.ProcStart())
	assert.Zero(t, c.ProcOutput([]byte("Unison 2.51.3 (ocaml 4.11.1): Contacting server...\n")))
	assert.Zero(t, c.ProcOutput([]byte("Connected [//aqtau//home/vasiliy/tmp/gunison/left -> //aqtau//home/vasiliy/tmp/gunison/right]\n")))
	assert.Zero(t, c.ProcOutput([]byte("Looking for changes\n")))
	assertEqual(t, c.Interrupt(),
		Update{Interrupt: true})
	assertEqual(t, c.Status, "Interrupting Unison")
	assert.True(t, c.Running)
	assert.True(t, c.Busy)
	assert.Nil(t, c.Interrupt)
	assert.NotNil(t, c.Kill)
	assert.Zero(t, c.ProcOutput([]byte("Terminated!\n")))
	assertEqual(t, c.Status, "Interrupting Unison")
	assert.True(t, c.Running)
	assertEqual(t, c.ProcExit(3, nil),
		Update{Messages: []Message{
			{"Terminated!", Info},
		}})
	assertEqual(t, c.Status, "Unison exited")
	assert.False(t, c.Running)
}

func TestSSHFailure(t *testing.T) {
	c := NewCore()
	assert.Zero(t, c.ProcStart())
	assert.Zero(t, c.ProcOutput([]byte("Unison 2.51.3 (ocaml 4.11.1): Contacting server...\n")))
	assert.Zero(t, c.ProcOutput([]byte("Fatal error: Lost connection with the server\n")))
	assertEqual(t, c.Status, "Contacting server")
	assert.True(t, c.Busy)
	assertEqual(t, c.ProcExit(3, nil),
		Update{Messages: []Message{
			{"Fatal error: Lost connection with the server", Error},
		}})
	assertEqual(t, c.Status, "Unison exited")
	assert.False(t, c.Busy)
}

func TestExtraneousOutput1(t *testing.T) {
	c := NewCore()
	assert.Zero(t, c.ProcStart())
	assert.Zero(t, c.ProcOutput([]byte("Unison 2.51.3 (ocaml 4.11.1): Contacting server...\n")))
	assert.Zero(t, c.ProcOutput([]byte("Looking for changes\n")))
	assert.Zero(t, c.ProcOutput([]byte("Something interesting happening here\n")))
	assertEqual(t, c.Status, "Looking for changes")
	assertEqual(t, c.ProcOutput([]byte("Reconciling changes\n")),
		Update{Messages: []Message{
			{"Something interesting happening here", Info},
		}})
	assertEqual(t, c.Status, "Reconciling changes")
}

func TestExtraneousOutput2(t *testing.T) {
	c := NewCore()
	assert.Zero(t, c.ProcStart())
	assert.Zero(t, c.ProcOutput([]byte("\nleft           right              \n")))
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            one  [f] ")),
		Update{Input: []byte("l\n")})
	assert.Zero(t, c.ProcOutput([]byte("  ")))
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            one  \n")))
	assert.Zero(t, c.ProcOutput([]byte("What is this line right here?\n")))
	assertEqual(t, c.ProcOutput([]byte("left         : changed file       modified on 2021-02-07 at  1:50:31  size 1146      rw-r--r--\nright        : unchanged file     modified on 2021-02-07 at  1:50:31  size 1146      rw-r--r--\n")),
		Update{
			Interrupt: true,
			Messages: []Message{
				{"Cannot parse the following output from Unison:\nWhat is this line right here?\nThis is a fatal error. Unison will be stopped now.", Error},
			},
		})
	assertEqual(t, c.Status, "Interrupting Unison")
}

func TestExtraneousOutput3(t *testing.T) {
	c := initCoreMinimalReady(t)
	c.Plan["one"] = RightToLeft
	assertEqual(t, c.Sync(),
		Update{Input: []byte("0\n")})
	assertEqual(t, c.ProcOutput([]byte("some unexpected line here\nchanged  ---->            one  [f] ")),
		Update{
			Interrupt: true,
			Messages: []Message{
				{"Cannot parse the following output from Unison:\nsome unexpected line here\nThis is a fatal error. Unison will be stopped now.", Error},
			},
		})
	assertEqual(t, c.Status, "Interrupting Unison")

	// The plan, once initialized, always remains available because the UI still needs it.
	assert.NotNil(t, c.Items)
	assert.NotNil(t, c.Plan)
}

func TestModifiedDuringSync(t *testing.T) {
	c := initCoreMinimalSyncing(t)
	assert.Zero(t, c.ProcOutput([]byte("Propagating updates\n")))
	assert.Zero(t, c.ProcOutput([]byte("\n\nUNISON 2.51.3 (OCAML 4.11.1) started propagating changes at 20:13:49.30 on 26 Feb 2021\n")))
	assert.Zero(t, c.ProcOutput([]byte("[BGN] Updating file one from /home/vasiliy/tmp/gunison/left to /home/vasiliy/tmp/gunison/right\n")))
	assert.Zero(t, c.ProcOutput([]byte("100%  00:00 ETA")))
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))
	assertEqual(t, c.ProcOutput([]byte("Failed: The source file /home/vasiliy/tmp/gunison/left/one\nhas been modified during synchronization.  Transfer aborted.\n")),
		Update{Messages: []Message{
			{"Failed: The source file /home/vasiliy/tmp/gunison/left/one", Error},
			{"has been modified during synchronization.  Transfer aborted.", Info},
		}})
	assert.Zero(t, c.ProcOutput([]byte("100%  00:00 ETA")))
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))
	assertEqual(t, c.ProcOutput([]byte("Failed [one]: The source file /home/vasiliy/tmp/gunison/left/one\nhas been modified during synchronization.  Transfer aborted.\n")),
		Update{Messages: []Message{
			{"Failed [one]: The source file /home/vasiliy/tmp/gunison/left/one", Error},
			{"has been modified during synchronization.  Transfer aborted.", Info},
		}})
	assert.Zero(t, c.ProcOutput([]byte("UNISON 2.51.3 (OCAML 4.11.1) finished propagating changes at 20:13:49.31 on 26 Feb 2021\n\n\n")))
	assert.Zero(t, c.ProcOutput([]byte("Saving synchronizer state\n")))
	assertEqual(t, c.ProcOutput([]byte("Synchronization incomplete at 20:13:49  (0 items transferred, 0 skipped, 1 failed)\n")),
		Update{Messages: []Message{
			{"Synchronization incomplete at 20:13:49  (0 items transferred, 0 skipped, 1 failed)", Warning},
		}})
	assertEqual(t, c.ProcOutput([]byte("  failed: one\n")),
		Update{Messages: []Message{
			{"failed: one", Error},
		}})
	assert.Zero(t, c.ProcExit(2, nil))
	assertEqual(t, c.Status, "Finished with errors")
}

func TestConnectionLostDuringSync(t *testing.T) {
	c := initCoreMinimalSyncing(t)
	assert.Zero(t, c.ProcOutput([]byte("Propagating updates\n")))
	assert.Zero(t, c.ProcOutput([]byte("\n\nUNISON 2.51.3 (OCAML 4.11.1) started propagating changes at 10:19:56.00 on 28 Feb 2021\n")))
	assert.Zero(t, c.ProcOutput([]byte("[BGN] Updating file one from /home/vasiliy/tmp/gunison/left to //aqtau//home/vasiliy/tmp/gunison/right\n")))
	assertEqual(t, c.ProcOutput([]byte("Fatal error: Lost connection with the server\n")),
		Update{Messages: []Message{
			{"Fatal error: Lost connection with the server", Error},
		}})
	assert.True(t, c.Busy)
	assert.Zero(t, c.ProcExit(3, nil))
	assertEqual(t, c.Status, "Unison exited")
	assert.False(t, c.Busy)
	assert.NotNil(t, c.Items)
	assert.NotNil(t, c.Plan)
}

func TestErrorDuringStart(t *testing.T) {
	c := NewCore()
	assertEqual(t, c.ProcError(errors.New(`exec: "unison": executable file not found in $PATH`)),
		Update{Messages: []Message{
			{`exec: "unison": executable file not found in $PATH`, Error},
		}})
	assert.False(t, c.Running)
	assert.False(t, c.Busy)
	assertEqual(t, c.Status, "Failed to start Unison")
	assert.Nil(t, c.Kill)
}

func TestErrorBeforeSync(t *testing.T) {
	c := initCoreMinimalReady(t)
	assertEqual(t, c.ProcError(errors.New("some unexpected error")),
		Update{
			Interrupt: true,
			Messages: []Message{
				{"some unexpected error\nThis is a fatal error. Unison will be stopped now.", Error},
			},
		})
	assertEqual(t, c.Status, "Interrupting Unison")
	assert.True(t, c.Busy)
	assert.Nil(t, c.Abort)
	assert.Nil(t, c.Interrupt)
	assert.NotNil(t, c.Kill)
	assert.NotNil(t, c.Items)
	assert.NotNil(t, c.Plan)
	assert.Zero(t, c.ProcOutput([]byte("Terminated!\n")))
	assertEqual(t, c.ProcExit(3, nil),
		Update{Messages: []Message{
			{"Terminated!", Info},
		}})
	assertEqual(t, c.Status, "Unison exited")
}

func TestErrorDuringSync(t *testing.T) {
	c := initCoreMinimalSyncing(t)
	assert.Zero(t, c.ProcOutput([]byte("Propagating updates\n")))
	// At this point we don't interrupt Unison just because some I/O error occurred.
	// Instead, we show the error to the user, and they can abort if necessary.
	assertEqual(t, c.ProcError(errors.New("some unexpected error")),
		Update{Messages: []Message{
			{"some unexpected error", Error},
		}})
	assertEqual(t, c.Status, "Propagating updates")
	assert.NotNil(t, c.Abort)
}

func TestPlanMissing(t *testing.T) {
	c := initCoreMinimalReady(t)
	delete(c.Plan, "one")
	assertEqual(t, c.Sync(),
		Update{Input: []byte("0\n")})
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            one  [f] ")),
		Update{
			Interrupt: true,
			Messages: []Message{
				{"Failed to start synchronization because this path is missing from Gunison's plan: one\nThis is probably a bug in Gunison.\nThis is a fatal error. Unison will be stopped now.", Error},
			},
		})
	assertEqual(t, c.Status, "Interrupting Unison")
}

// assertEqual is just assert.Equal with arguments swapped,
// which makes for more readable code in places.
func assertEqual(t *testing.T, actual, expected interface{}) bool { //nolint:unparam
	t.Helper()
	return assert.Equal(t, expected, actual)
}

func initCoreMinimalReady(t *testing.T) *Core { //nolint:thelper
	c := NewCore()
	c.ProcStart()
	c.ProcOutput([]byte("\nleft           right              \n"))
	c.ProcOutput([]byte("changed  ---->            one  [f] "))
	c.ProcOutput([]byte("  "))
	c.ProcOutput([]byte("changed  ---->            one  \n"))
	c.ProcOutput([]byte("left         : changed file       modified on 2021-02-07 at  1:50:31  size 1146      rw-r--r--\nright        : unchanged file     modified on 2021-02-07 at  1:50:31  size 1146      rw-r--r--\n"))
	c.ProcOutput([]byte("changed  ---->            one  [f] "))

	assert.True(t, c.Running)
	assert.False(t, c.Busy)
	assertEqual(t, c.Status, "Ready to synchronize")
	assert.Empty(t, c.Progress)
	assert.Empty(t, c.ProgressFraction)
	assertEqual(t, c.Left, "left")
	assertEqual(t, c.Right, "right")
	assertEqual(t, c.Items, []Item{
		{
			Path: "one",
			Left: Content{File, Modified, "modified on 2021-02-07 at  1:50:31  size 1146      rw-r--r--",
				time.Date(2021, 2, 7, 1, 50, 31, 0, time.Local), 1146},
			Right: Content{File, Unchanged, "modified on 2021-02-07 at  1:50:31  size 1146      rw-r--r--",
				time.Date(2021, 2, 7, 1, 50, 31, 0, time.Local), 1146},
			Action: LeftToRight,
		},
	})
	assertEqual(t, c.Plan, map[string]Action{
		"one": LeftToRight,
	})

	require.NotNil(t, c.Diff)
	require.NotNil(t, c.Sync)
	require.NotNil(t, c.Quit)
	require.Nil(t, c.Abort)
	require.NotNil(t, c.Interrupt)
	require.NotNil(t, c.Kill)

	return c
}

func initCoreMinimalSyncing(t *testing.T) *Core { //nolint:thelper
	c := initCoreMinimalReady(t)
	c.Sync()
	c.ProcOutput([]byte("changed  ---->            one  [f] "))
	c.ProcOutput([]byte("changed  ---->            one  \n"))
	c.ProcOutput([]byte("\nProceed with propagating updates? [] "))

	assert.True(t, c.Running)
	assert.True(t, c.Busy)
	assertEqual(t, c.Status, "Starting synchronization")
	assert.Empty(t, c.Progress)
	assert.Empty(t, c.ProgressFraction)
	assert.Nil(t, c.Diff)
	assert.Nil(t, c.Sync)
	assert.Nil(t, c.Quit)
	require.NotNil(t, c.Abort)
	require.NotNil(t, c.Interrupt)
	require.NotNil(t, c.Kill)

	return c
}
