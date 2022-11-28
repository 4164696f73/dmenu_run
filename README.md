# dmenu_run

This small program is basically a dmenu wrapper that allows the user to:
- launch programs like in dmenu
- launch programs from XDG_DATA_DIRS, PATH and $HOME/.local/applications
- trigger shell commands, using your favorite shell (reading $SHELL, else defaulting to sh)
- add aliases specified by user (similar to shell aliases, just identified differently)
- add additional folders with executables, so that you can launch executables from anywhere, even from $HOME/.config if you're this crazy, so that you can use them to launch Appimages etc (though I myself have not tested it with Appimages myself!)*

Use `dmenu_run -h` for more information on how to use it.

*I don't know how Appimages work, but if it's just like running other programs, so ./name, then it'll work without any issues, though if it requires a program for it to run, you'd need to [for now] use an alias to run them. I'd be happy to adjust the code to run with Appimages, because the point of this wrapper is to be able to launch anything however you want via dmenu.

How to install:
It's basically as simple as copying the executable to dmenu folder and installing it (it replaces the original dmenu_run, and does not use dmenu_path at all!) by `sudo make install`.

If you want to build it yourself, clone the repo, use `go build dmenu_run.go` and them move it to the folder with dmenu, replacing the original dmenu_run, and install dmenu as usual.