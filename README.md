# Gunison

Gunison is a new GUI (GTK 3) frontend for the [Unison][] file synchronizer.
Unison already has a built-in GTK 2 frontend, but Gunison is nicer and more 
convenient (see below). Gunison works by wrapping Unison's *console* frontend 
in an [expect][]-like fashion.

**WARNING:** Although I use Gunison daily, it has not been tested widely; and
due to its very nature (parsing human-readable output from Unison), it will
never be as reliable as Unison itself. Expect severe bugs.

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


### Gunison from source

You need [Go][] 1.16+ and Git, as well as GTK 3 and the associated C toolchain.
`go install github.com/vfaronov/gunison@latest` will download and compile
Gunison and its dependencies, and install the `gunison` (or `gunison.exe`)
executable in `$GOBIN`. Alternatively, `go build` in a source checkout will
produce the same executable in `$PWD`.

Here's a complete recipe for Debian/Ubuntu:

```
sudo apt install build-essential git libgtk-3-dev
wget https://golang.org/dl/go1.16.3.linux-amd64.tar.gz
tar -xzf go1.16.3.linux-amd64.tar.gz
GOBIN=$PWD go/bin/go install -v github.com/vfaronov/gunison@latest
./gunison
```

[Go]: https://golang.org/


## Usage

Run `gunison` with the same command-line arguments as you would run `unison`
(they will be passed on). Unlike `unison-gtk2`, Gunison will not offer
to select a profile from the GUI: you must specify the profile or roots
on the command line.

Do not set [preferences][prefs] that affect Unison's console behavior, such as
`terse` or `repeat`. They may break Gunison.

It's best to set the `diff` preference to a GUI tool that produces no console
output, such as [`meld`][Meld]. Otherwise, diffs will be opened as temporary
files in your operating system, but, due to an [inefficiency in Gunison][],
loading large diffs may be very slow.

Keyboard navigation is via common GTK features:

* press - (minus) or Shift+Left to collapse a directory
* press + (plus) to expand a directory,
  Shift+Right to expand with all its children
* press Ctrl+F1 to toggle a tooltip with details for the selected file
* press Menu or Shift+F10 to pop up an action menu for the selected files
* use mnemonics (underlined keys) to access buttons and menu items, 
  e.g. Alt+S for Sync

At exit, Gunison saves some UI state (collapsed directories, window geometry,
column order) to a file named `state.json` in a platform-dependent config
directory, such as `~/.config/gunison` on Unix. You can prevent this by
symlinking that file to `/dev/null`.

[prefs]: https://www.cis.upenn.edu/~bcpierce/unison/download/releases/stable/unison-manual.html#prefs
[Meld]: https://meldmerge.org/
[inefficiency in Gunison]: https://github.com/vfaronov/gunison/issues/1
