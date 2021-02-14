// +build !coremock

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	assert.NotNil(t, c.Interrupt)
	assert.NotNil(t, c.Kill)

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
		Update{Input: []byte("l")})
	assertEqual(t, c.Status, "Assembling plan")
	assertEqual(t, c.Left, "left")
	assertEqual(t, c.Right, "right")
	assert.Zero(t, c.ProcOutput([]byte("l\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            one  \n")))
	assert.Zero(t, c.ProcOutput([]byte("left         : changed file       modified on 2021-02-08 at 18:30:50  size 1146      rw-r--r--\nright        : unchanged file     modified on 2021-02-08 at 18:30:50  size 1146      rw-r--r--\n")))
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            one  [f] ")),
		Update{PlanReady: true})
	assertEqual(t, c.Status, "Ready to synchronize")
	assert.False(t, c.Busy)
	assertEqual(t, c.Items, []Item{
		{
			Path:   "one",
			Left:   Content{File, Modified, "modified on 2021-02-08 at 18:30:50  size 1146      rw-r--r--"},
			Right:  Content{File, Unchanged, "modified on 2021-02-08 at 18:30:50  size 1146      rw-r--r--"},
			Action: LeftToRight,
		},
	})
	require.NotNil(t, c.Sync)
	assert.NotNil(t, c.Quit)

	assertEqual(t, c.Sync(),
		Update{Input: []byte("0")})
	assertEqual(t, c.Status, "Starting synchronization")
	assert.True(t, c.Busy)
	assert.Nil(t, c.Sync)
	assert.Nil(t, c.Quit)
	assert.NotNil(t, c.Abort)
	assert.Zero(t, c.ProcOutput([]byte("0\n")))
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            one  [f] ")),
		Update{Input: []byte(">")})
	assert.Zero(t, c.ProcOutput([]byte(">\r")))
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            one  \n")))
	assertEqual(t, c.ProcOutput([]byte("\nProceed with propagating updates? [] ")),
		Update{Input: []byte("y")})
	assert.Zero(t, c.ProcOutput([]byte("y\n")))

	assert.Zero(t, c.ProcOutput([]byte("Propagating updates\n")))
	assertEqual(t, c.Status, "Propagating updates")
	assert.Zero(t, c.ProcOutput([]byte("\n\nUNISON 2.51.3 (OCAML 4.11.1) started propagating changes at 18:31:20.92 on 08 Feb 2021\n")))
	assert.Zero(t, c.ProcOutput([]byte("[BGN] Updating file one from /home/vasiliy/tmp/gunison/left to /home/vasiliy/tmp/gunison/right\n")))
	assertEqual(t, c.ProcOutput([]byte("100%  00:00 ETA")),
		Update{Progressed: true})
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

	assert.Zero(t, c.ProcOutput([]byte("Synchronization complete at 18:31:20  (1 item transferred, 0 skipped, 0 failed)\n")))
	assertEqual(t, c.Status, "Sync complete (1 item transferred)")
	assert.False(t, c.Busy)
	assert.True(t, c.Running)

	assert.Zero(t, c.ProcExit(0, nil))
	assert.False(t, c.Running)
}

func TestTerse(t *testing.T) { // unison -terse
	c := NewCore()

	assert.Zero(t, c.ProcStart())
	assert.Zero(t, c.ProcOutput([]byte("\nleft           right              \n")))
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            one  [f] ")),
		Update{Input: []byte("l")})
	assertEqual(t, c.Status, "Assembling plan")
	assertEqual(t, c.Left, "left")
	assertEqual(t, c.Right, "right")
	assert.Zero(t, c.ProcOutput([]byte("l\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            one  \n")))
	assert.Zero(t, c.ProcOutput([]byte("left         : changed file       modified on 2021-02-07 at  1:50:31  size 1146      rw-r--r--\nright        : unchanged file     modified on 2021-02-07 at  1:50:31  size 1146      rw-r--r--\n")))
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            one  [f] ")))
	assertEqual(t, c.Status, "Ready to synchronize")

	assertEqual(t, c.Sync(),
		Update{Input: []byte("0")})
	assert.Zero(t, c.ProcOutput([]byte("0\n")))
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            one  [f] ")),
		Update{Input: []byte(">")})
	assert.Zero(t, c.ProcOutput([]byte(">\r")))
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            one  \n")))
	assertEqual(t, c.ProcOutput([]byte("\nProceed with propagating updates? [] ")),
		Update{Input: []byte("y")})
	assert.Zero(t, c.ProcOutput([]byte("y")))
	assertEqual(t, c.Status, "Propagating updates")

	assert.Zero(t, c.ProcOutput([]byte("[BGN] Updating file one from /home/vasiliy/tmp/gunison/left to /home/vasiliy/tmp/gunison/right\n")))
	assert.Zero(t, c.ProcOutput([]byte("[END] Updating file one\n")))
	assert.Zero(t, c.ProcOutput([]byte("Synchronization complete at 01:50:54  (1 item transferred, 0 skipped, 0 failed)\n")))
	assert.Zero(t, c.ProcExit(0, nil))
}

func TestProgressLookingForChanges(t *testing.T) {
	c := NewCore()
	assert.Zero(t, c.ProcStart())
	assert.Zero(t, c.ProcOutput([]byte("Unison 2.51.3 (ocaml 4.11.1): Contacting server...\n")))
	assert.Zero(t, c.ProcOutput([]byte("Looking for changes\n")))

	assertEqual(t, c.ProcOutput([]byte("| some/file")),
		Update{Progressed: true})
	assertEqual(t, c.Progress, "some/file")
	assertEqual(t, c.ProgressFraction, -1)

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

	assert.Zero(t, c.ProcOutput([]byte("  Waiting for changes from server\n")))
	assertEqual(t, c.Status, "Waiting for changes from server")
	assert.Empty(t, c.Progress)
	assert.Empty(t, c.ProgressFraction)
}

func TestProgressPropagatingUpdates(t *testing.T) {
	c := NewCore()
	assert.Zero(t, c.ProcStart())
	assert.Zero(t, c.ProcOutput([]byte("\nleft           right              \n")))
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            one  [f] ")),
		Update{Input: []byte("l")})
	assert.Zero(t, c.ProcOutput([]byte("l\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            one  \n")))
	assert.Zero(t, c.ProcOutput([]byte("left         : changed file       modified on 2021-02-07 at  1:50:31  size 1146      rw-r--r--\nright        : unchanged file     modified on 2021-02-07 at  1:50:31  size 1146      rw-r--r--\n")))
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            one  [f] ")),
		Update{PlanReady: true})
	assertEqual(t, c.Sync(),
		Update{Input: []byte("0")})
	assert.Zero(t, c.ProcOutput([]byte("0\n")))
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            one  [f] ")),
		Update{Input: []byte(">")})
	assert.Zero(t, c.ProcOutput([]byte(">\r")))
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            one  \n")))
	assertEqual(t, c.ProcOutput([]byte("\nProceed with propagating updates? [] ")),
		Update{Input: []byte("y")})
	assert.Zero(t, c.ProcOutput([]byte("y")))

	assert.Zero(t, c.ProcOutput([]byte("[BGN] Updating file one from /home/vasiliy/tmp/gunison/left to /home/vasiliy/tmp/gunison/right\n")))
	assert.Empty(t, c.Progress)
	assert.Empty(t, c.ProgressFraction)

	assertEqual(t, c.ProcOutput([]byte("  0%  73:28 ETA")),
		Update{Progressed: true})
	assertEqual(t, c.Progress, "0%  73:28 ETA")
	assertEqual(t, c.ProgressFraction, 0.00)
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))
	assertEqual(t, c.ProcOutput([]byte("  8%  07:45 ETA")),
		Update{Progressed: true})
	assertEqual(t, c.Progress, "8%  07:45 ETA")
	assertEqual(t, c.ProgressFraction, 0.08)
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))

	assertEqual(t, c.ProcOutput([]byte(" 13%  07:51 ETA")),
		Update{Progressed: true})
	assertEqual(t, c.Progress, "13%  07:51 ETA")
	assertEqual(t, c.ProgressFraction, 0.13)
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))

	assertEqual(t, c.ProcOutput([]byte(" 94%  00:01 ETA")),
		Update{Progressed: true})
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
	assertEqual(t, c.Status, "Starting unison")
	assertEqual(t, c.ProcExit(1, nil),
		Update{Message: Message{
			Text:       "Unison exited, saying:\nUsage: unison [options]\n[...] Profile /home/vasiliy/tmp/gunison/.unison/nonexistent.prf does not exist",
			Importance: Error,
		}})
	assert.False(t, c.Busy)
	assertEqual(t, c.Status, "Unison exited unexpectedly")
	assert.Nil(t, c.Interrupt)
}

func TestDiff(t *testing.T) {
	c := NewCore()
	assert.Zero(t, c.ProcStart())
	assert.Zero(t, c.ProcOutput([]byte("\nleft           right              \n")))
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            file1  [f] ")),
		Update{Input: []byte("l")})
	assert.Zero(t, c.ProcOutput([]byte("l\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            file1  \n")))
	assert.Zero(t, c.ProcOutput([]byte("left         : changed file       modified on 2021-02-13 at 14:29:12  size 1146      rw-r--r--\nright        : unchanged file     modified on 2021-02-13 at 14:29:12  size 1146      rw-r--r--\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            file2  \n")))
	assert.Zero(t, c.ProcOutput([]byte("left         : changed file       modified on 2021-02-13 at 14:29:12  size 1146      rw-r--r--\nright        : unchanged file     modified on 2021-02-13 at 14:29:12  size 1146      rw-r--r--\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            file3  \n")))
	assert.Zero(t, c.ProcOutput([]byte("left         : changed file       modified on 2021-02-13 at 14:29:12  size 1146      rw-r--r--\nright        : unchanged file     modified on 2021-02-13 at 14:29:12  size 1146      rw-r--r--\n  ")))
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            file1  [f] ")),
		Update{PlanReady: true})

	assertEqual(t, c.Diff("file3"),
		Update{Input: []byte("0")})
	assertEqual(t, c.Status, "Requesting diff")
	assert.True(t, c.Busy)
	assert.Nil(t, c.Sync)
	assert.Nil(t, c.Diff)
	assert.Nil(t, c.Quit)
	assert.NotNil(t, c.Abort)
	assert.Zero(t, c.ProcOutput([]byte("0\n")))
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            file1  [f] ")),
		Update{Input: []byte("n")})
	assert.Zero(t, c.ProcOutput([]byte("n\n")))
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            file2  [f] ")),
		Update{Input: []byte("n")})
	assert.Zero(t, c.ProcOutput([]byte("n\n")))
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            file3  [f] ")),
		Update{Input: []byte("d")})
	assert.Zero(t, c.ProcOutput([]byte("d\n")))
	assertEqual(t, c.ProcOutput([]byte(`"\ndiff -u '/home/vasiliy/tmp/gunison/right/file3' '/home/vasiliy/tmp/gunison/left/file3'\n\n--- /home/vasiliy/tmp/gunison/right/file3\t2021-02-13 14:29:12.571303322 +0300\n+++ /home/vasiliy/tmp/gunison/left/file3\t2021-02-13 14:29:12.575303310 +0300\n@@ -1,9 +1,9 @@\n Quia est unde laboriosam. Eum ullam deleniti dolores. Magni quasi facere voluptas. Dolor doloribus aut ut sed officiis id. Et aut nostrum est quia corrupti maiores optio.\n \n-Consectetur fuga sed vitae et nihil quia. Eveniet rerum officia repudiandae tenetur molestiae. Magni ipsum et natus accusantium ut consequatur neque. Veniam in voluptate quia. Culpa labore distinctio laudantium maxime voluptate eaque.\n+Consectetur fuga sed vitae et nihil quia. Eveniet rerum officia repudiandae teonsectetur tempore quia id. Perspiciatis enim corrupti aliquam nam accusamus et molestiae rerum. Sint sit exercitationem corrupti omnis.\n \n-Deserunt dignissimos corrupti aut vel. Laboriosam at labore omnis eos et minus porro perspiciatis. Veniam in dignissimos voluptatem exercitationem excepturi reprehenderit sed optio.\n+Facere asperiores unde rerum dignissimos id. Nihil maiores sequi accusamus eum repudiandae et. Nesciunt ab inveniatis. Veniam in dignissimos voluptatem exercitationem excepturi reprehenderit sed optio.\n \n-Omnis repudiandae nobis autem qui autem possimus. Dolorem id a reprehenderit nihil laboriosam non. Dolor minima in soluta. Magni eveniet magnam velit officia consectetur tempore quia id. Perspiciatis enim corrupti aliquam nam accusamus et molestiae rerum. Sint sit exercitationem corrupti omnis.\n+Omnis repudiandae nobis autem qui autem possimus. Dolorem id a reprehenderit nihil laboriosam non. Dolor minima in soluta. Magni eveniet magnam velit officia cnetur molestiae. Magni ipsum et natus accusantium ut consequatur neque. Veniam in voluptate quia. Culpa labore distinctio laudantium maxime voluptate eaque.\n \n-Facere asperiores unde rerum dignissimos id. Nihil maiores sequi accusamus eum repudiandae et. Nesciunt ab inventore repellat enim illum ratione enim voluptatum. Tempore sint quos tempore fugit rerum sit omnis quae. Minus deserunt aut dolores excepturi qui.\n+Deserunt dignissimos corrupti aut vel. Laboriosam at labore omnis eos et minus porro perspictore repellat enim illum ratione enim voluptatum. Tempore sint quos tempore fugit rerum sit omnis quae. Minus deserunt aut dolores excepturi qui.\n\nchanged  ---->            file3  [f] "`)),
		Update{Diff: []byte("--- /home/vasiliy/tmp/gunison/right/file3\t2021-02-13 14:29:12.571303322 +0300\n+++ /home/vasiliy/tmp/gunison/left/file3\t2021-02-13 14:29:12.575303310 +0300\n@@ -1,9 +1,9 @@\n Quia est unde laboriosam. Eum ullam deleniti dolores. Magni quasi facere voluptas. Dolor doloribus aut ut sed officiis id. Et aut nostrum est quia corrupti maiores optio.\n \n-Consectetur fuga sed vitae et nihil quia. Eveniet rerum officia repudiandae tenetur molestiae. Magni ipsum et natus accusantium ut consequatur neque. Veniam in voluptate quia. Culpa labore distinctio laudantium maxime voluptate eaque.\n+Consectetur fuga sed vitae et nihil quia. Eveniet rerum officia repudiandae teonsectetur tempore quia id. Perspiciatis enim corrupti aliquam nam accusamus et molestiae rerum. Sint sit exercitationem corrupti omnis.\n \n-Deserunt dignissimos corrupti aut vel. Laboriosam at labore omnis eos et minus porro perspiciatis. Veniam in dignissimos voluptatem exercitationem excepturi reprehenderit sed optio.\n+Facere asperiores unde rerum dignissimos id. Nihil maiores sequi accusamus eum repudiandae et. Nesciunt ab inveniatis. Veniam in dignissimos voluptatem exercitationem excepturi reprehenderit sed optio.\n \n-Omnis repudiandae nobis autem qui autem possimus. Dolorem id a reprehenderit nihil laboriosam non. Dolor minima in soluta. Magni eveniet magnam velit officia consectetur tempore quia id. Perspiciatis enim corrupti aliquam nam accusamus et molestiae rerum. Sint sit exercitationem corrupti omnis.\n+Omnis repudiandae nobis autem qui autem possimus. Dolorem id a reprehenderit nihil laboriosam non. Dolor minima in soluta. Magni eveniet magnam velit officia cnetur molestiae. Magni ipsum et natus accusantium ut consequatur neque. Veniam in voluptate quia. Culpa labore distinctio laudantium maxime voluptate eaque.\n \n-Facere asperiores unde rerum dignissimos id. Nihil maiores sequi accusamus eum repudiandae et. Nesciunt ab inventore repellat enim illum ratione enim voluptatum. Tempore sint quos tempore fugit rerum sit omnis quae. Minus deserunt aut dolores excepturi qui.\n+Deserunt dignissimos corrupti aut vel. Laboriosam at labore omnis eos et minus porro perspictore repellat enim illum ratione enim voluptatum. Tempore sint quos tempore fugit rerum sit omnis quae. Minus deserunt aut dolores excepturi qui.\n")})
	assertEqual(t, c.Status, "Ready to synchronize")
	assert.False(t, c.Busy)
	assert.NotNil(t, c.Sync)
	assert.NotNil(t, c.Diff)
	assert.NotNil(t, c.Quit)
	assert.Nil(t, c.Abort)

	assertEqual(t, c.Diff("file2"),
		Update{Input: []byte("0")})
	assert.Zero(t, c.ProcOutput([]byte("0\n")))
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            file1  [f] ")),
		Update{Input: []byte("n")})
	assert.Zero(t, c.ProcOutput([]byte("n\n")))
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            file2  [f] ")),
		Update{Input: []byte("d")})
	assert.Zero(t, c.ProcOutput([]byte("d\n")))
	assertEqual(t, c.ProcOutput([]byte("\ndiff -u '/home/vasiliy/tmp/gunison/right/file2' '/home/vasiliy/tmp/gunison/left/file2'\n\n--- /home/vasiliy/tmp/gunison/right/file2\t2021-02-13 14:29:12.571303322 +0300\n+++ /home/vasiliy/tmp/gunison/left/file2\t2021-02-13 14:29:12.571303322 +0300\n@@ -1,6 +1,6 @@\n-Quia est unde laboriosam. Eum ullam deleniti dolores. Magni quasi facere voluptas. Dolor doloribus aut ut sed officiis id. Et aut nostrum est quia corrupti maiores optio.\n+Quia est unde laboriosam. Eum ullam deleniti dolores. Magni quasi facere voluptas. Dolor doloribus aut consequatur neque. Veniam in voluptate quia. Culpa labore distinctio laudantium maxime voluptate t nihil quia. Eveniet rerum officia repudiandae tenetur molestiae. Magni ipsum et natus accusantium ut ut sed officiis id. Et aut nostrum est quia corrupti maiores optio.\n \n-Consectetur fuga sed vitae et nihil quia. Eveniet rerum officia repudiandae tenetur molestiae. Magni ipsum et natus accusantium ut consequatur neque. Veniam in voluptate quia. Culpa labore distinctio laudantium maxime voluptate eaque.\n+Consectetur fuga sed vitae eeaque.\n \n Deserunt dignissimos corrupti aut vel. Laboriosam at labore omnis eos et minus porro perspiciatis. Veniam in dignissimos voluptatem exercitationem excepturi reprehenderit sed optio.\n \n\nchanged  ---->            file2  [f] ")),
		Update{Diff: []byte("--- /home/vasiliy/tmp/gunison/right/file2\t2021-02-13 14:29:12.571303322 +0300\n+++ /home/vasiliy/tmp/gunison/left/file2\t2021-02-13 14:29:12.571303322 +0300\n@@ -1,6 +1,6 @@\n-Quia est unde laboriosam. Eum ullam deleniti dolores. Magni quasi facere voluptas. Dolor doloribus aut ut sed officiis id. Et aut nostrum est quia corrupti maiores optio.\n+Quia est unde laboriosam. Eum ullam deleniti dolores. Magni quasi facere voluptas. Dolor doloribus aut consequatur neque. Veniam in voluptate quia. Culpa labore distinctio laudantium maxime voluptate t nihil quia. Eveniet rerum officia repudiandae tenetur molestiae. Magni ipsum et natus accusantium ut ut sed officiis id. Et aut nostrum est quia corrupti maiores optio.\n \n-Consectetur fuga sed vitae et nihil quia. Eveniet rerum officia repudiandae tenetur molestiae. Magni ipsum et natus accusantium ut consequatur neque. Veniam in voluptate quia. Culpa labore distinctio laudantium maxime voluptate eaque.\n+Consectetur fuga sed vitae eeaque.\n \n Deserunt dignissimos corrupti aut vel. Laboriosam at labore omnis eos et minus porro perspiciatis. Veniam in dignissimos voluptatem exercitationem excepturi reprehenderit sed optio.\n \n")})

	assertEqual(t, c.Diff("file1"),
		Update{Input: []byte("0")})
	assert.Zero(t, c.ProcOutput([]byte("0\n")))
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            file1  [f] ")),
		Update{Input: []byte("d")})
	assert.Zero(t, c.ProcOutput([]byte("d\n")))
	assertEqual(t, c.ProcOutput([]byte("\ndiff -u '/home/vasiliy/tmp/gunison/right/file1' '/home/vasiliy/tmp/gunison/left/file1'\n\n--- /home/vasiliy/tmp/gunison/right/file1\t2021-02-13 14:29:12.571303322 +0300\n+++ /home/vasiliy/tmp/gunison/left/file1\t2021-02-13 14:29:12.571303322 +0300\n@@ -1,6 +1,6 @@\n-Quia est unde laboriosam. Eum ullam deleniti dolores. Magni quasi facere voluptas. Dolor doloribus aut ut sed officiis id. Et aut nostrum est quia corrupti maiores optio.\n+Quia est unde laboriosam. Eum ullam deleniti dolorrupti maiores optio.\n \n-Consectetur fuga sed vitae et nihil quia. Eveniet rerum officia repudiandae tenetur molestiae. Magni ipsum et natus accusantium ut consequatur neque. Veniam in voluptate quia. Culpa labore distinctio laudantium maxime voluptate eaque.\n+Consectetur fuga sed vitae eut ut sed officiis id. Et aut nostrum est quia cores. Magni quasi facere voluptas. Dolor doloribus at nihil quia. Eveniet rerum officia repudiandae tenetur molestiae. Magni ipsum et natus accusantium ut consequatur neque. Veniam in voluptate quia. Culpa labore distinctio laudantium maxime voluptate eaque.\n \n Deserunt dignissimos corrupti aut vel. Laboriosam at labore omnis eos et minus porro perspiciatis. Veniam in dignissimos voluptatem exercitationem excepturi reprehenderit sed optio.\n \n\nchanged  ---->            file1  [f] ")),
		Update{Diff: []byte("--- /home/vasiliy/tmp/gunison/right/file1\t2021-02-13 14:29:12.571303322 +0300\n+++ /home/vasiliy/tmp/gunison/left/file1\t2021-02-13 14:29:12.571303322 +0300\n@@ -1,6 +1,6 @@\n-Quia est unde laboriosam. Eum ullam deleniti dolores. Magni quasi facere voluptas. Dolor doloribus aut ut sed officiis id. Et aut nostrum est quia corrupti maiores optio.\n+Quia est unde laboriosam. Eum ullam deleniti dolorrupti maiores optio.\n \n-Consectetur fuga sed vitae et nihil quia. Eveniet rerum officia repudiandae tenetur molestiae. Magni ipsum et natus accusantium ut consequatur neque. Veniam in voluptate quia. Culpa labore distinctio laudantium maxime voluptate eaque.\n+Consectetur fuga sed vitae eut ut sed officiis id. Et aut nostrum est quia cores. Magni quasi facere voluptas. Dolor doloribus at nihil quia. Eveniet rerum officia repudiandae tenetur molestiae. Magni ipsum et natus accusantium ut consequatur neque. Veniam in voluptate quia. Culpa labore distinctio laudantium maxime voluptate eaque.\n \n Deserunt dignissimos corrupti aut vel. Laboriosam at labore omnis eos et minus porro perspiciatis. Veniam in dignissimos voluptatem exercitationem excepturi reprehenderit sed optio.\n \n")})
}

func TestDiffNoOutput(t *testing.T) {
	// When the diff command produces no output, it's probably a GUI one, so we silently ignore it.
	c := NewCore()
	assert.Zero(t, c.ProcStart())
	assert.Zero(t, c.ProcOutput([]byte("\nleft           right              \n")))
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            one  [f] ")),
		Update{Input: []byte("l")})
	assert.Zero(t, c.ProcOutput([]byte("l\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            one  \n")))
	assert.Zero(t, c.ProcOutput([]byte("left         : changed file       modified on 2021-02-07 at  1:40:31  size 1146      rw-r--r--\nright        : unchanged file     modified on 2021-02-07 at  1:40:31  size 1146      rw-r--r--\n")))
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            one  [f] ")),
		Update{PlanReady: true})
	assertEqual(t, c.Diff("one"),
		Update{Input: []byte("0")})
	assert.Equal(t, c.Status, "Requesting diff")
	assert.Zero(t, c.ProcOutput([]byte("0\n")))
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            one  [f] ")),
		Update{Input: []byte("d")})
	assert.Zero(t, c.ProcOutput([]byte("d\n")))
	assert.Zero(t, c.ProcOutput([]byte("\ntrue '/home/vasiliy/tmp/gunison/left/one' '/home/vasiliy/tmp/gunison/right/one'\n\n\n\nchanged  ---->            one  [f] ")))
	assert.Equal(t, c.Status, "Ready to synchronize")
}

func TestDiffDirectory(t *testing.T) {
	c := NewCore()
	assert.Zero(t, c.ProcStart())
	assert.Zero(t, c.ProcOutput([]byte("\nleft           right              \n")))
	assertEqual(t, c.ProcOutput([]byte("changed  <-?-> new dir    one hundred/one hundred one  [] ")),
		Update{Input: []byte("l")})
	assert.Zero(t, c.ProcOutput([]byte("l\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("changed  <-?-> new dir    one hundred/one hundred one  \n")))
	assert.Zero(t, c.ProcOutput([]byte("left         : changed file       modified on 2021-02-13 at 14:56:44  size 1000      rw-r--r--\nright        : new dir            modified on 2021-02-13 at 14:56:44  size 2292      rwxr-xr-x\n  ")))
	assertEqual(t, c.ProcOutput([]byte("changed  <-?-> new dir    one hundred/one hundred one  [] ")),
		Update{PlanReady: true})
	assertEqual(t, c.Diff("one hundred/one hundred one"),
		Update{Input: []byte("0")})
	assert.Zero(t, c.ProcOutput([]byte("0\n")))
	assertEqual(t, c.ProcOutput([]byte("changed  <-?-> new dir    one hundred/one hundred one  [] ")),
		Update{Input: []byte("d")})
	assert.Zero(t, c.ProcOutput([]byte("d\n")))
	assertEqual(t, c.ProcOutput([]byte("Can't diff: path doesn't refer to a file in both replicas\nchanged  <-?-> new dir    one hundred/one hundred one  [] ")),
		Update{Message: Message{
			Text:       "Can't diff: path doesn't refer to a file in both replicas",
			Importance: Error,
		}})
	assertEqual(t, c.Status, "Ready to synchronize")
}

func TestDiffBadPath(t *testing.T) {
	c := NewCore()
	assert.Zero(t, c.ProcStart())
	assert.Zero(t, c.ProcOutput([]byte("\nleft           right              \n")))
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            one  [f] ")),
		Update{Input: []byte("l")})
	assert.Zero(t, c.ProcOutput([]byte("l\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            one  \n")))
	assert.Zero(t, c.ProcOutput([]byte("left         : changed file       modified on 2021-02-13 at 15:02:55  size 1146      rw-r--r--\nright        : unchanged file     modified on 2021-02-13 at 15:02:55  size 1146      rw-r--r--\n")))
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            one  [f] ")),
		Update{PlanReady: true})
	assertEqual(t, c.Diff("two"),
		Update{Input: []byte("0")})
	assert.Zero(t, c.ProcOutput([]byte("0\n")))
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            one  [f] ")),
		Update{Input: []byte("n")})
	assert.Zero(t, c.ProcOutput([]byte("n\n")))
	assertEqual(t, c.ProcOutput([]byte("\nProceed with propagating updates? [] ")),
		Update{
			Input: []byte("n"),
			Message: Message{
				Text:       "Failed to get diff for: two\nThere is no such path in Unison’s plan. This is probably a bug in Gunison.",
				Importance: Error,
			},
		})
	assert.Equal(t, c.Status, "Waiting for Unison")
	assert.True(t, c.Busy)
	assert.Nil(t, c.Sync)
	assert.Nil(t, c.Diff)
	assert.Zero(t, c.ProcOutput([]byte("n\n")))
	assert.Zero(t, c.ProcOutput([]byte("\nleft           right              \n")))
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            one  [f] ")))
	assert.Equal(t, c.Status, "Ready to synchronize")
	assert.NotNil(t, c.Sync)
	assert.NotNil(t, c.Diff)
}

func TestDiffAbort(t *testing.T) {
	c := NewCore()
	assert.Zero(t, c.ProcStart())
	assert.Zero(t, c.ProcOutput([]byte("\nleft           right              \n")))
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            file1  [f] ")),
		Update{Input: []byte("l")})
	assert.Zero(t, c.ProcOutput([]byte("l\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            file1  \n")))
	assert.Zero(t, c.ProcOutput([]byte("left         : changed file       modified on 2021-02-13 at 15:12:12  size 1146      rw-r--r--\nright        : unchanged file     modified on 2021-02-13 at 15:12:12  size 1146      rw-r--r--\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            file2  \n")))
	assert.Zero(t, c.ProcOutput([]byte("left         : changed file       modified on 2021-02-13 at 15:12:12  size 1146      rw-r--r--\nright        : unchanged file     modified on 2021-02-13 at 15:12:12  size 1146      rw-r--r--\n  ")))
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            file1  [f] ")),
		Update{PlanReady: true})

	assertEqual(t, c.Diff("file2"),
		Update{Input: []byte("0")})
	assert.Zero(t, c.ProcOutput([]byte("0\n")))
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            file1  [f] ")),
		Update{Input: []byte("n")})
	// While we're still seeking to the requested file in Unison prompt, we can abort gracefully.
	assert.Zero(t, c.Abort())
	assertEqual(t, c.Status, "Waiting for Unison")
	assert.True(t, c.Busy)
	assert.Nil(t, c.Sync)
	assert.Zero(t, c.ProcOutput([]byte("n\n")))
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            file2  [f] ")))
	assertEqual(t, c.Status, "Ready to synchronize")
	assert.False(t, c.Busy)
	assert.NotNil(t, c.Sync)

	assertEqual(t, c.Diff("file2"),
		Update{Input: []byte("0")})
	assert.Zero(t, c.ProcOutput([]byte("0\n")))
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            file1  [f] ")),
		Update{Input: []byte("n")})
	assert.Zero(t, c.ProcOutput([]byte("n\n")))
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            file2  [f] ")),
		Update{Input: []byte("d")})
	assert.Zero(t, c.ProcOutput([]byte("d\n")))
	// But if we have already requested the diff, the only way to abort is by interrupting.
	assertEqual(t, c.Abort(),
		Update{Interrupt: true})
}

func TestReplicaMissing(t *testing.T) {
	c := NewCore()
	assert.Zero(t, c.ProcStart())
	assert.Zero(t, c.ProcOutput([]byte("Unison 2.51.3 (ocaml 4.11.1): Contacting server...\n")))
	assert.Zero(t, c.ProcOutput([]byte("Looking for changes\n")))
	assert.Zero(t, c.ProcOutput([]byte("Reconciling changes\n")))
	assert.Zero(t, c.ProcOutput([]byte("The root of one of the replicas has been completely emptied.\nUnison may delete everything in the other replica.  (Set the \n'confirmbigdel' preference to false to disable this check.)\n\n")))
	upd := c.ProcOutput([]byte("Do you really want to proceed? [] "))
	assertEqual(t, upd.Message.Text, "The root of one of the replicas has been completely emptied.\nUnison may delete everything in the other replica.  (Set the \n'confirmbigdel' preference to false to disable this check.)\n\nDo you really want to proceed?")
	assertEqual(t, upd.Message.Importance, Warning)
	assertEqual(t, c.Status, "Reconciling changes")
	assert.True(t, c.Busy)
	assertEqual(t, upd.Message.Proceed(),
		Update{Input: []byte("y")})
	assert.Zero(t, c.ProcOutput([]byte("y")))
	assert.Zero(t, c.ProcOutput([]byte("\nleft           right              \n")))
	assertEqual(t, c.ProcOutput([]byte("         <---- deleted      [f] ")),
		Update{Input: []byte("l")})
	assert.Zero(t, c.ProcOutput([]byte("l\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("         <---- deleted      \n")))
	assert.Zero(t, c.ProcOutput([]byte("left         : unchanged dir      modified on 2021-02-06 at 18:31:42  size 1146      rwxr-xr-x\nright        : deleted\n")))
	assertEqual(t, c.ProcOutput([]byte("         <---- deleted      [f] ")),
		Update{PlanReady: true})
	assertEqual(t, c.Status, "Ready to synchronize")
	assert.False(t, c.Busy)
	assert.NotNil(t, c.Sync)
}

func TestReplicaMissingAbort(t *testing.T) {
	c := NewCore()
	assert.Zero(t, c.ProcStart())
	assert.Zero(t, c.ProcOutput([]byte("The root of one of the replicas has been completely emptied.\nUnison may delete everything in the other replica.  (Set the \n'confirmbigdel' preference to false to disable this check.)\n\n")))
	upd := c.ProcOutput([]byte("Do you really want to proceed? [] "))
	assertEqual(t, upd.Message.Abort(),
		Update{Input: []byte("q")})
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
	assertEqual(t, upd.Message.Text, "No archive files were found for these roots, whose canonical names are:\n\t/home/vasiliy/tmp/gunison/left\n\t/home/vasiliy/tmp/gunison/right\nThis can happen either\nbecause this is the first time you have synchronized these roots, \nor because you have upgraded Unison to a new version with a different\narchive format.  \n\nUpdate detection may take a while on this run if the replicas are \nlarge.\n\nUnison will assume that the 'last synchronized state' of both replicas\nwas completely empty.  This means that any files that are different\nwill be reported as conflicts, and any files that exist only on one\nreplica will be judged as new and propagated to the other replica.\nIf the two replicas are identical, then no changes will be reported.\n\nIf you see this message repeatedly, it may be because one of your machines\nis getting its address from DHCP, which is causing its host name to change\nbetween synchronizations.  See the documentation for the UNISONLOCALHOSTNAME\nenvironment variable for advice on how to correct this.\n\nDonations to the Unison project are gratefully accepted: \nhttp://www.cis.upenn.edu/~bcpierce/unison")
	assertEqual(t, upd.Message.Importance, Warning)
	assertEqual(t, c.Status, "Looking for changes")
	assert.True(t, c.Busy)
	assertEqual(t, upd.Message.Proceed(),
		Update{Input: []byte("\n")})
	assert.Zero(t, c.ProcOutput([]byte("\n")))
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
	assertEqual(t, upd.Message.Abort(),
		Update{Input: []byte("q")})
	assertEqual(t, c.Status, "Quitting Unison")
}

// assertEqual is just assert.Equal with arguments swapped,
// which makes for more readable code in places.
func assertEqual(t *testing.T, actual, expected interface{}) bool { //nolint:unparam
	t.Helper()
	return assert.Equal(t, expected, actual)
}
