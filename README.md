# Gunison

Gunison is a new GUI (GTK 3) frontend for the [Unison][] file synchronizer.
Unison already has a built-in GTK 2 frontend, but Gunison is nicer and more convenient.
Gunison works by wrapping Unison’s *console* frontend in an [expect][]-like fashion.

**WARNING:** Although I use Gunison daily, it has not been tested widely; and due to its very nature
(parsing human-readable output from Unison), it will never be as reliable as Unison itself. 
Expect severe bugs.

[Unison]: https://www.cis.upenn.edu/~bcpierce/unison/
[expect]: https://en.wikipedia.org/wiki/Expect


## Usage

Run `gunison` with the same command-line arguments as you would run `unison` (they will be passed on).
Unlike `unison-gtk2`, Gunison will not offer to select a profile from the GUI: you must specify
the profile or roots on the command line.

Do not set [preferences][prefs] that affect Unison's console behavior, such as `terse` or `repeat`.
They may break Gunison.

It's best to set the `diff` preference to a GUI tool that produces no console output,
such as [`meld`][Meld]. Otherwise, diffs will be opened as files in your operating system,
but, due to an inefficiency in Gunison, loading large diffs may be very slow.

Keyboard navigation is via common GTK features:

* press Ctrl+F1 to toggle a tooltip with details for the selected file
* press Menu or Shift+F10 to pop up an action menu for the selected files
* use mnemonics (underlined keys) to access buttons and menu items, e.g. Alt+S for Sync

[prefs]: https://www.cis.upenn.edu/~bcpierce/unison/download/releases/stable/unison-manual.html#prefs
[Meld]: https://meldmerge.org/
