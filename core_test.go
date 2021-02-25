// +build !coremock

package main

import (
	"testing"
	"time"

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
		Update{Input: []byte("l\n")})
	assertEqual(t, c.Status, "Assembling plan")
	assertEqual(t, c.Left, "left")
	assertEqual(t, c.Right, "right")
	assert.Zero(t, c.ProcOutput([]byte("  ")))
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            one  \n")))
	assert.Zero(t, c.ProcOutput([]byte("left         : changed file       modified on 2021-02-08 at 18:30:50  size 1146      rw-r--r--\nright        : unchanged file     modified on 2021-02-08 at 18:30:50  size 1146      rw-r--r--\n")))
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            one  [f] ")),
		Update{PlanReady: true})
	assertEqual(t, c.Status, "Ready to synchronize")
	assert.False(t, c.Busy)
	assertEqual(t, c.Items, []Item{
		{
			Path: "one",
			Left: Content{File, Modified, "modified on 2021-02-08 at 18:30:50  size 1146      rw-r--r--",
				time.Date(2021, 2, 8, 18, 30, 50, 0, time.Local)},
			Right: Content{File, Unchanged, "modified on 2021-02-08 at 18:30:50  size 1146      rw-r--r--",
				time.Date(2021, 2, 8, 18, 30, 50, 0, time.Local)},
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
	assertEqual(t, c.Status, "Sync complete (1 item transferred, 0 skipped, 0 failed)")
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
	c := initCoreMinimalReady(t)
	assertEqual(t, c.Sync(),
		Update{Input: []byte("0\n")})
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            one  [f] ")),
		Update{Input: []byte(">\n")})
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            one  \n")))
	assertEqual(t, c.ProcOutput([]byte("\nProceed with propagating updates? [] ")),
		Update{Input: []byte("y\n")})

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
	assertEqual(t, c.Status, "Starting unison")
	assertEqual(t, c.ProcExit(1, nil),
		Update{Messages: []Message{
			{"Unison exited, saying:\nUsage: unison [options]\n[...] Profile /home/vasiliy/tmp/gunison/.unison/nonexistent.prf does not exist", Error},
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
		Update{Input: []byte("l\n")})
	assert.Zero(t, c.ProcOutput([]byte("  ")))
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            file1  \n")))
	assert.Zero(t, c.ProcOutput([]byte("left         : changed file       modified on 2021-02-13 at 14:29:12  size 1146      rw-r--r--\nright        : unchanged file     modified on 2021-02-13 at 14:29:12  size 1146      rw-r--r--\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            file2  \n")))
	assert.Zero(t, c.ProcOutput([]byte("left         : changed file       modified on 2021-02-13 at 14:29:12  size 1146      rw-r--r--\nright        : unchanged file     modified on 2021-02-13 at 14:29:12  size 1146      rw-r--r--\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            file3  \n")))
	assert.Zero(t, c.ProcOutput([]byte("left         : changed file       modified on 2021-02-13 at 14:29:12  size 1146      rw-r--r--\nright        : unchanged file     modified on 2021-02-13 at 14:29:12  size 1146      rw-r--r--\n  ")))
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            file1  [f] ")),
		Update{PlanReady: true})

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
	assertEqual(t, c.ProcOutput([]byte("changed  <-?-> new dir    one hundred/one hundred one  [] ")),
		Update{PlanReady: true})
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

func TestDiffBadPath(t *testing.T) {
	c := initCoreMinimalReady(t)
	assertEqual(t, c.Diff("two"),
		Update{Input: []byte("0\n")})
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            one  [f] ")),
		Update{Input: []byte("n\n")})
	assertEqual(t, c.ProcOutput([]byte("\nProceed with propagating updates? [] ")),
		Update{
			Input: []byte("n\n"),
			Messages: []Message{
				{"Failed to get diff for: two\nThere is no such path in Unison’s plan. This is probably a bug in Gunison.", Error},
			},
		})
	assertEqual(t, c.Status, "Waiting for Unison")
	assert.True(t, c.Busy)
	assert.Nil(t, c.Sync)
	assert.Nil(t, c.Diff)
	assert.Zero(t, c.ProcOutput([]byte("\nleft           right              \n")))
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            one  [f] ")))
	assertEqual(t, c.Status, "Ready to synchronize")
	assert.NotNil(t, c.Sync)
	assert.NotNil(t, c.Diff)
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
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            file1  [f] ")),
		Update{PlanReady: true})

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
			{"diff: unrecognized option '--bad-option'\ndiff: Try 'diff --help' for more information.", Error},
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
	assert.Zero(t, c.ProcOutput([]byte("Synchronization complete at 16:40:04  (1 item transferred, 0 skipped, 0 failed)\n")))
	assert.Zero(t, c.ProcExit(0, nil))
	assertEqual(t, c.Status, "Sync complete (1 item transferred, 0 skipped, 0 failed)")
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
			{"Failed [one]: Merge program didn't change either temp file\n", Error},
		}})
	assert.Zero(t, c.ProcOutput([]byte("Synchronization incomplete at 16:49:21  (0 items transferred, 0 skipped, 1 failed)\n")))
	assert.Zero(t, c.ProcExit(2, nil))
	assertEqual(t, c.Status, "Sync incomplete (0 items transferred, 0 skipped, 1 failed)")
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
	assertEqual(t, c.ProcOutput([]byte("new dir  ---->            one  [f] ")),
		Update{PlanReady: true})
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
	assert.Zero(t, c.ProcOutput([]byte("Synchronization incomplete at 17:06:28  (0 items transferred, 0 skipped, 1 failed)\n")))
	assert.Zero(t, c.ProcOutput([]byte("  failed: one\n")))
	assert.Zero(t, c.ProcExit(2, nil))
	assertEqual(t, c.Status, "Sync incomplete (0 items transferred, 0 skipped, 1 failed)")
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
	assertEqual(t, c.ProcOutput([]byte("         <---- deleted      [f] ")),
		Update{PlanReady: true})
	assertEqual(t, c.Status, "Ready to synchronize")
	assertEqual(t, c.Items, []Item{
		{
			Path: "",
			Left: Content{Directory, Unchanged, "modified on 2021-02-06 at 18:31:42  size 1146      rwxr-xr-x",
				time.Date(2021, 2, 6, 18, 31, 42, 0, time.Local)},
			Right:  Content{Absent, Deleted, "", time.Time{}},
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
	assertEqual(t, upd.Alert.Text, "No archive files were found for these roots, whose canonical names are:\n\t/home/vasiliy/tmp/gunison/left\n\t/home/vasiliy/tmp/gunison/right\nThis can happen either\nbecause this is the first time you have synchronized these roots, \nor because you have upgraded Unison to a new version with a different\narchive format.  \n\nUpdate detection may take a while on this run if the replicas are \nlarge.\n\nUnison will assume that the 'last synchronized state' of both replicas\nwas completely empty.  This means that any files that are different\nwill be reported as conflicts, and any files that exist only on one\nreplica will be judged as new and propagated to the other replica.\nIf the two replicas are identical, then no changes will be reported.\n\nIf you see this message repeatedly, it may be because one of your machines\nis getting its address from DHCP, which is causing its host name to change\nbetween synchronizations.  See the documentation for the UNISONLOCALHOSTNAME\nenvironment variable for advice on how to correct this.\n\nDonations to the Unison project are gratefully accepted: \nhttp://www.cis.upenn.edu/~bcpierce/unison")
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

func TestAssorted(t *testing.T) {
	c := NewCore()
	assert.Zero(t, c.ProcStart())
	assert.Zero(t, c.ProcOutput([]byte("\nlocal          tanais             \n")))
	assertEqual(t, c.ProcOutput([]byte("changed  <-?-> new dir    one hundred/one hundred one  [] ")),
		Update{Input: []byte("l\n")})
	assert.Zero(t, c.ProcOutput([]byte("  ")))
	assert.Zero(t, c.ProcOutput([]byte("changed  <-?-> new dir    one hundred/one hundred one  \n")))
	assert.Zero(t, c.ProcOutput([]byte("local        : changed file       modified on 2021-02-17 at 16:21:41  size 1000      rw-r--r--\ntanais       : new dir            modified on 2021-02-17 at 16:21:40  size 2292      rwxr-xr-x\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("new file <-?-> deleted    one hundred/one hundred two  \n")))
	assert.Zero(t, c.ProcOutput([]byte("local        : new file           modified on 2021-02-17 at 16:21:33  size 1146      rw-r--r--\ntanais       : deleted\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("changed  <-?-> changed    six/nine  \n")))
	assert.Zero(t, c.ProcOutput([]byte("local        : changed file       modified on 2021-02-17 at 16:21:41  size 1147000   rw-r--r--\ntanais       : changed file       modified on 2021-02-17 at 16:21:43  size 1147000   rw-rw-r--\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("props    <-?-> props      twenty one  \n")))
	assert.Zero(t, c.ProcOutput([]byte("local        : changed props      modified on 2021-02-17 at 16:21:41  size 1146      rw-r--r--\ntanais       : changed props      modified on 2021-02-17 at 16:21:55  size 1146      rw-rw-r--\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("         <---- changed    deeply/nested/sub/directory/with/file  \n")))
	assert.Zero(t, c.ProcOutput([]byte("local        : unchanged file     modified on 2021-02-17 at 16:21:40  size 1146      rw-r--r--\ntanais       : changed file       modified on 2021-02-17 at 16:21:56  size 1146      rw-rw-r--\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("new link ---->            eighteen  \n")))
	assert.Zero(t, c.ProcOutput([]byte("local        : new symlink        modified on 1970-01-01 at  3:00:00  size 0         unknown permissions\ntanais       : absent\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("         <---- changed    here is a rather long and funny file name, 社會科學院語學研究所\t\v\f \u0085 \u1680\u2002\u2003\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u200b\u2028\u2029\u202f\u205f\u3000ﾟ･✿ヾ╲(｡◕‿◕｡)╱✿･ﾟ/here is a rather long and funny file name, 社會科學院語學研究所\t\v\f \u0085 \u1680\u2002\u2003\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u200b\u2028\u2029\u202f\u205f\u3000ﾟ･✿ヾ╲(｡◕‿◕｡)╱✿･ﾟ  \n")))
	assert.Zero(t, c.ProcOutput([]byte("local        : unchanged file     modified on 2021-02-17 at 16:21:40  size 1146      rw-r--r--\ntanais       : changed file       modified on 2021-02-17 at 16:21:55  size 1146      rw-rw-r--\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("new file ---->            seventeen  \n")))
	assert.Zero(t, c.ProcOutput([]byte("local        : new file           modified on 2021-02-17 at 16:21:41  size 0         rw-r--r--\ntanais       : absent\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("         <---- changed    six/eight  \n")))
	assert.Zero(t, c.ProcOutput([]byte("local        : unchanged file     modified on 2021-02-17 at 16:21:40  size 1146      rw-r--r--\ntanais       : changed file       modified on 2021-02-17 at 16:21:42  size 1147000   rw-rw-r--\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("         <---- changed    six/eleven  \n")))
	assert.Zero(t, c.ProcOutput([]byte("local        : unchanged file     modified on 2021-02-17 at 16:21:40  size 10000000  rw-r--r--\ntanais       : changed file       modified on 2021-02-17 at 16:21:55  size 10000000  rw-rw-r--\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("         <---- deleted    six/fourteen  \n")))
	assert.Zero(t, c.ProcOutput([]byte("local        : unchanged dir      modified on 2021-02-17 at 16:21:33  size 2292      rwxr-xr-x\ntanais       : deleted\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("         <---- changed    six/seven  \n")))
	assert.Zero(t, c.ProcOutput([]byte("local        : unchanged file     modified on 2021-02-17 at 16:21:40  size 0         rw-r--r--\ntanais       : changed file       modified on 2021-02-17 at 16:21:41  size 1146      rw-rw-r--\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("props    ---->            six/ten  \n")))
	assert.Zero(t, c.ProcOutput([]byte("local        : changed props      modified on 2021-02-17 at 16:21:33  size 1000      rwx------\ntanais       : unchanged file     modified on 2021-02-17 at 16:21:33  size 1000      rw-r--r--\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("         <---- deleted    three  \n")))
	assert.Zero(t, c.ProcOutput([]byte("local        : unchanged file     modified on 2021-02-17 at 16:21:40  size 1147000   rw-r--r--\ntanais       : deleted\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("         <---- props      twelve  \n")))
	assert.Zero(t, c.ProcOutput([]byte("local        : unchanged dir      modified on 2021-02-17 at 16:21:33  size 0         rwxr-xr-x\ntanais       : dir props changed  modified on 2021-02-17 at 16:21:39  size 0         rwx------\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("         <---- new dir    twenty  \n")))
	assert.Zero(t, c.ProcOutput([]byte("local        : absent\ntanais       : new dir            modified on 2021-02-17 at 16:21:44  size 0         rwxr-xr-x\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            two  \n")))
	assert.Zero(t, c.ProcOutput([]byte("local        : changed file       modified on 2021-02-17 at 16:21:41  size 1146      rw-r--r--\ntanais       : unchanged file     modified on 2021-02-17 at 16:21:33  size 1146      rw-r--r--\n")))
	assertEqual(t, c.ProcOutput([]byte("changed  <-?-> new dir    one hundred/one hundred one  [] ")),
		Update{PlanReady: true})

	assertEqual(t, c.Items, []Item{
		{
			Path: "one hundred/one hundred one",
			Left: Content{File, Modified, "modified on 2021-02-06 at 18:42:07  size 1000      rw-r--r--",
				time.Date(2021, 2, 6, 18, 42, 7, 0, time.Local)},
			Right: Content{Directory, Created, "modified on 2021-02-06 at 18:42:07  size 2292      rwxr-xr-x",
				time.Date(2021, 2, 6, 18, 42, 7, 0, time.Local)},
			Action: Skip,
		},
		{
			Path: "one hundred/one hundred two",
			Left: Content{File, Created, "modified on 2021-02-06 at 18:41:58  size 1146      rw-r--r--",
				time.Date(2021, 2, 6, 18, 41, 58, 0, time.Local)},
			Right:  Content{Absent, Deleted, "", time.Time{}},
			Action: Skip,
		},
		{
			Path: "six/nine",
			Left: Content{File, Modified, "modified on 2021-02-06 at 18:42:07  size 1147000   rw-r--r--",
				time.Date(2021, 2, 6, 18, 42, 7, 0, time.Local)},
			Right: Content{File, Modified, "modified on 2021-02-06 at 18:42:10  size 1147000   rw-rw-r--",
				time.Date(2021, 2, 6, 18, 42, 10, 0, time.Local)},
			Action: Skip,
		},
		{
			Path: "twenty one",
			Left: Content{File, PropsChanged, "modified on 2021-02-06 at 18:42:07  size 1146      rw-r--r--",
				time.Date(2021, 2, 6, 18, 42, 7, 0, time.Local)},
			Right: Content{File, PropsChanged, "modified on 2021-02-06 at 18:42:21  size 1146      rw-rw-r--",
				time.Date(2021, 2, 6, 18, 42, 21, 0, time.Local)},
			Action: Skip,
		},
		{
			Path: "deeply/nested/sub/directory/with/file",
			Left: Content{File, Unchanged, "modified on 2021-02-06 at 18:42:07  size 1146      rw-r--r--",
				time.Date(2021, 2, 6, 18, 42, 7, 0, time.Local)},
			Right: Content{File, Modified, "modified on 2021-02-06 at 18:42:21  size 1146      rw-rw-r--",
				time.Date(2021, 2, 6, 18, 42, 21, 0, time.Local)},
			Action: RightToLeft,
		},
		{
			Path: "eighteen",
			Left: Content{Symlink, Created, "modified on 1970-01-01 at  3:00:00  size 0         unknown permissions",
				time.Date(1970, 1, 1, 3, 0, 0, 0, time.Local)},
			Right:  Content{Absent, Unchanged, "", time.Time{}},
			Action: LeftToRight,
		},
		{
			Path: "here is a rather long and funny file name, 社會科學院語學研究所\t\v\f \u0085 \u1680\u2002\u2003\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u200b\u2028\u2029\u202f\u205f\u3000ﾟ･✿ヾ╲(｡◕‿◕｡)╱✿･ﾟ/here is a rather long and funny file name, 社會科學院語學研究所\t\v\f \u0085 \u1680\u2002\u2003\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u200b\u2028\u2029\u202f\u205f\u3000ﾟ･✿ヾ╲(｡◕‿◕｡)╱✿･ﾟ",
			Left: Content{File, Unchanged, "modified on 2021-02-06 at 18:42:07  size 1146      rw-r--r--",
				time.Date(2021, 2, 6, 18, 42, 7, 0, time.Local)},
			Right: Content{File, Modified, "modified on 2021-02-06 at 18:42:20  size 1146      rw-rw-r--",
				time.Date(2021, 2, 6, 18, 42, 20, 0, time.Local)},
			Action: RightToLeft,
		},
		{
			Path: "seventeen",
			Left: Content{File, Created, "modified on 2021-02-06 at 18:42:07  size 0         rw-r--r--",
				time.Date(2021, 2, 6, 18, 42, 7, 0, time.Local)},
			Right:  Content{Absent, Unchanged, "", time.Time{}},
			Action: LeftToRight,
		},
		{
			Path: "six/eight",
			Left: Content{File, Unchanged, "modified on 2021-02-06 at 18:42:07  size 1146      rw-r--r--",
				time.Date(2021, 2, 6, 18, 42, 7, 0, time.Local)},
			Right: Content{File, Modified, "modified on 2021-02-06 at 18:42:08  size 1147000   rw-rw-r--",
				time.Date(2021, 2, 6, 18, 42, 8, 0, time.Local)},
			Action: RightToLeft,
		},
		{
			Path: "six/eleven",
			Left: Content{File, Unchanged, "modified on 2021-02-06 at 18:42:07  size 10000000  rw-r--r--",
				time.Date(2021, 2, 6, 18, 42, 7, 0, time.Local)},
			Right: Content{File, Modified, "modified on 2021-02-06 at 18:42:20  size 10000000  rw-rw-r--",
				time.Date(2021, 2, 6, 18, 42, 20, 0, time.Local)},
			Action: RightToLeft,
		},
		{
			Path: "six/fourteen",
			Left: Content{Directory, Unchanged, "modified on 2021-02-06 at 18:41:58  size 2292      rwxr-xr-x",
				time.Date(2021, 2, 6, 18, 41, 58, 0, time.Local)},
			Right:  Content{Absent, Deleted, "", time.Time{}},
			Action: RightToLeft,
		},
		{
			Path: "six/seven",
			Left: Content{File, Unchanged, "modified on 2021-02-06 at 18:42:07  size 0         rw-r--r--",
				time.Date(2021, 2, 6, 18, 42, 7, 0, time.Local)},
			Right: Content{File, Modified, "modified on 2021-02-06 at 18:42:08  size 1146      rw-rw-r--",
				time.Date(2021, 2, 6, 18, 42, 8, 0, time.Local)},
			Action: RightToLeft,
		},
		{
			Path: "six/ten",
			Left: Content{File, PropsChanged, "modified on 2021-02-06 at 18:41:58  size 1000      rwx------",
				time.Date(2021, 2, 6, 18, 41, 58, 0, time.Local)},
			Right: Content{File, Unchanged, "modified on 2021-02-06 at 18:41:58  size 1000      rw-r--r--",
				time.Date(2021, 2, 6, 18, 41, 58, 0, time.Local)},
			Action: LeftToRight,
		},
		{
			Path: "three",
			Left: Content{File, Unchanged, "modified on 2021-02-06 at 18:42:07  size 1147000   rw-r--r--",
				time.Date(2021, 2, 6, 18, 42, 7, 0, time.Local)},
			Right:  Content{Absent, Deleted, "", time.Time{}},
			Action: RightToLeft,
		},
		{
			Path: "twelve",
			Left: Content{Directory, Unchanged, "modified on 2021-02-06 at 18:41:58  size 34380     rwxr-xr-x",
				time.Date(2021, 2, 6, 18, 41, 58, 0, time.Local)},
			Right: Content{Directory, PropsChanged, "modified on 2021-02-06 at 18:42:06  size 0         rwx------",
				time.Date(2021, 2, 6, 18, 42, 6, 0, time.Local)},
			Action: RightToLeft,
		},
		{
			Path: "twenty",
			Left: Content{Absent, Unchanged, "", time.Time{}},
			Right: Content{Directory, Created, "modified on 2021-02-06 at 18:42:10  size 0         rwxr-xr-x",
				time.Date(2021, 2, 6, 18, 42, 10, 0, time.Local)},
			Action: RightToLeft,
		},
		{
			Path: "two",
			Left: Content{File, Modified, "modified on 2021-02-06 at 18:42:07  size 1146      rw-r--r--",
				time.Date(2021, 2, 6, 18, 42, 7, 0, time.Local)},
			Right: Content{File, Unchanged, "modified on 2021-02-06 at 18:41:58  size 1146      rw-r--r",
				time.Date(2021, 2, 6, 18, 41, 58, 0, time.Local)},
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
	assert.Zero(t, c.ProcOutput([]byte("Synchronization incomplete at 16:24:22  (14 items transferred, 2 skipped, 1 failed)\n")))
	assert.Zero(t, c.ProcOutput([]byte("  skipped: one hundred/one hundred two (conflicting updates)\n")))
	assert.Zero(t, c.ProcOutput([]byte("  skipped: twelve (skip requested)\n")))
	assert.Zero(t, c.ProcOutput([]byte("  failed: two\n")))
	assert.Zero(t, c.ProcExit(2, nil))
	assertEqual(t, c.Status, "Sync incomplete (14 items transferred, 2 skipped, 1 failed)")
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
	assert.Zero(t, c.ProcExit(3, nil))
	assertEqual(t, c.Status, "Interrupted")
	assert.False(t, c.Running)
}

func TestSSHFailure(t *testing.T) {
	c := NewCore()
	assert.Zero(t, c.ProcStart())
	assert.Zero(t, c.ProcOutput([]byte("Unison 2.51.3 (ocaml 4.11.1): Contacting server...\n")))
	assertEqual(t, c.ProcOutput([]byte("Fatal error: Lost connection with the server\n")),
		Update{Messages: []Message{
			{"Lost connection with the server", Error},
		}})
	assertEqual(t, c.Status, "Contacting server")
	assert.True(t, c.Busy)
	assert.Zero(t, c.ProcExit(3, nil))
	assertEqual(t, c.Status, "Unison exited unexpectedly")
	assert.False(t, c.Busy)
}

// assertEqual is just assert.Equal with arguments swapped,
// which makes for more readable code in places.
func assertEqual(t *testing.T, actual, expected interface{}) bool { //nolint:unparam
	t.Helper()
	return assert.Equal(t, expected, actual)
}

func initCoreMinimalReady(t *testing.T) *Core {
	t.Helper()

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
			Left: Content{File, Modified, "modified on 2021-02-08 at 18:30:50  size 1146      rw-r--r--",
				time.Date(2021, 2, 8, 18, 30, 50, 0, time.Local)},
			Right: Content{File, Unchanged, "modified on 2021-02-08 at 18:30:50  size 1146      rw-r--r--",
				time.Date(2021, 2, 8, 18, 30, 50, 0, time.Local)},
			Action: LeftToRight,
		},
	})
	assertEqual(t, c.Plan, map[string]Action{
		"one": LeftToRight,
	})

	require.NotNil(t, c.Diff)
	require.NotNil(t, c.Sync)
	require.NotNil(t, c.Quit)
	require.NotNil(t, c.Abort)
	require.NotNil(t, c.Interrupt)
	require.NotNil(t, c.Kill)

	return c
}
