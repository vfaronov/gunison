# Gunison

Gunison is a new GUI (GTK 3) frontend for the [Unison][] file synchronizer.
Unison already has a built-in GTK 2 frontend, but Gunison is nicer and more
convenient. Gunison works by wrapping Unison's *console* frontend
in an [expect][]-like fashion.

**Caution:** Although I use Gunison daily, it is less reliable than Unison
itself (and will always be, if only because it parses Unison’s human-readable
output). There are [known bugs](https://github.com/vfaronov/gunison/issues)
and probably unknown ones, too.

Gunison looks like this:

![Neat widgets, sync plan arranged into a collapsible tree][gunison.png]

For comparison, the built-in `unison-gtk2`:

![Flat list of items, many large buttons][unison.png]

[Unison]: https://www.cis.upenn.edu/~bcpierce/unison/
[expect]: https://en.wikipedia.org/wiki/Expect
[gunison.png]: tools/screenshots/gunison.png
[unison.png]: tools/screenshots/unison.png


## Installation

### Prerequisites

You need the [`unison`][] program on your `$PATH`, and GTK 3 runtime
libraries installed in your system.

If you use SSH with passwords, you will also need an [`SSH_ASKPASS`][]
program, because Gunison itself won't prompt you for the password. On Linux,
try installing `ssh-askpass-gnome`.

[`unison`]: https://github.com/bcpierce00/unison/wiki/Downloading-Unison
[`SSH_ASKPASS`]: https://man.openbsd.org/ssh#SSH_ASKPASS


### Gunison pre-built

If you use Linux x86_64, try the [pre-built binary][].

[pre-built binary]: https://github.com/vfaronov/gunison/releases


### Gunison from source

On other platforms, or if the pre-built binary doesn't work, build from source.
You need [Go][] 1.17+ and Git, as well as GTK 3 and the associated C toolchain.
`go install github.com/vfaronov/gunison@latest` will download and compile
Gunison and its dependencies, and install the `gunison` (or `gunison.exe`)
executable in `$GOBIN`. Alternatively, `go install .` in a source checkout.

Here's a complete recipe for Debian/Ubuntu:

```
sudo apt install build-essential git libgtk-3-dev
wget https://golang.org/dl/go1.17.linux-amd64.tar.gz
tar -xzf go1.17.linux-amd64.tar.gz
GOBIN=$PWD go/bin/go install -v github.com/vfaronov/gunison@latest
./gunison
```

[Go]: https://golang.org/


## Basic usage

Run `gunison` with the same command-line arguments as you would run `unison`
(they will be passed on). Unlike `unison-gtk2`, Gunison will not offer
to select a profile from the GUI: you must specify the profile or roots
on the command line.

Do not set [preferences][prefs] that affect Unison's console behavior, such as
`terse` or `repeat`. They may break Gunison.

It's best to set the `diff` preference to a GUI tool that produces no console
output, such as [`meld`][Meld]. Otherwise, diffs will be opened as temporary
files in your operating system, but loading large diffs [may be slow][].


## Working with the sync plan

Bring up the menu by right-clicking on the tree or pressing the Menu
or Shift+F10 keys. This menu lets you set the action to be performed on items,
as well as view differences between files. You can select multiple items
or folders at once and operate on them all together.

Gunison remembers which directories you have collapsed in the tree — very
useful for `.git` directories, for example. But, this doesn’t distinguish 
profiles or roots: if you collapse a `Documents/work` directory in one profile,
this may affect an unrelated `Documents/work` directory in another.

If the tree is too bushy for your liking, try enabling the *Squash
single-item folders* option in the menu. This will display `dir1/file.txt`
in one line when `dir1` contains only `file.txt`.

By clicking on the column headers, you can sort items by path or action.
Sorting by action in both directions (click twice) is a quick way to check if
all actions are the same.

You can also rearrange and resize columns by dragging them, as usual.
This will be remembered for subsequent runs.


## Keyboard shortcuts

There are currently no shortcuts specific to Gunison, but you can use 
the common GTK shortcuts, which [by default][bindings] are:

* \- (minus) or Shift+Left to collapse a folder
* \+ (plus) to expand a folder, Shift+Right to expand with all its children
* Ctrl+F1 to toggle a tooltip with details for the selected item
* Menu or Shift+F10 to pop up the context menu
* mnemonics (underlined keys) to access buttons and menu entries,
  e.g. Alt+S for Sync


## Configuration files

Gunison saves your UI options, as well as the list of collapsed directories,
to a file named `state.json` in a platform-dependent config directory —
usually `~/.config/gunison` on Unix. You can edit this file by hand. Or,
by symlinking it do `/dev/null`, you can prevent Gunison from saving anything.

[prefs]: https://www.cis.upenn.edu/~bcpierce/unison/download/releases/stable/unison-manual.html#prefs
[Meld]: https://meldmerge.org/
[may be slow]: https://github.com/vfaronov/gunison/issues/1
[bindings]: https://docs.gtk.org/gtk3/key-bindings.html
