package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type dmenu struct {
	pipe     []string
	execList []string
	strPipe  string
	alias    []string
	command  []string
	dir      []string
}

type config struct{ terminal string }

// Global variable so we can access it wherever.
var c config

func (c *config) getTerm() {
	f, err := os.ReadFile(os.Getenv("HOME") + "/.config/dmenu/config")
	if err != nil {
		fmt.Println(err)
		return
	}

	str := string(f)
	s := strings.SplitN(str[:len(str)-1], "\n", -1)
	for i := range s {
		if s[i][:2] == "//" {
			continue
		}
		if s[i][:1] == "$" {
			// Assure that any command containing < won't be touched!
			// Using just SpliN provides a total length of 1, this provides all.
			s := strings.SplitN(strings.Replace(s[i], "<", "\n", 1), "\n", -1)
			t := s[1]
			if t[len(t)-1:] == ">" {
				t = t[:len(t)-1]
			}

			if s[0] == "$term" || s[0] == "$terminal" {
				c.terminal = t + " "
			}
		}
	}
}

func (d *dmenu) searchDir(path string, dir []os.DirEntry) {
	for _, l := range dir {
		if !strings.HasSuffix(path, "/") {
			path = path + "/"
		}
		f, err := os.Lstat(path + l.Name())
		if f.IsDir() {
			// if strings.Contains(l.Name(), "applications") {
			// Technically could use Contains, but we don't really care
			// to search through anything that contains that word.
			if strings.ToLower(l.Name()) == "applications" {
				d.dir = append(d.dir, path+l.Name())
			}
			continue
		}
		if err != nil {
			fmt.Println("os.Lstat:", err)
			continue
		}

		// In case you randomly have the case messed up.
		if strings.HasSuffix(strings.ToLower(l.Name()), ".desktop") {
			fname, err := os.ReadFile(path + l.Name())
			if err != nil {
				fmt.Println("os.ReadFile:", err)
				continue
			}
			s := strings.SplitN(string(fname), "\n", -1)
			// Only append if the .desktop file contains both name and exec. Some don't do that.
			remember := [3]string{"", "", ""}

			for i := range s {
				line := strings.SplitN(strings.Replace(s[i], "=", "\n", 1), "\n", -1)

				if strings.ToLower(line[0]) == "name" {
					remember[0] = line[1]
				}
				if strings.ToLower(line[0]) == "exec" {
					remember[1] = line[1]
				}

				if remember[0] != "" && remember[1] != "" {
					d.pipe = append(d.pipe, remember[0]+" ("+strings.ReplaceAll(l.Name(), ".desktop", "")+")")
					if strings.HasSuffix(remember[1][len(remember[1])-3:len(remember[1])-1], " %") {
						remember[1] = remember[1][:len(remember[1])-3]
					}
					if strings.Contains(strings.ToLower(string(fname)), "terminal=true") {
						remember[2] = c.terminal
					}
					d.execList = append(d.execList, remember[2]+remember[1])
					break
				}
			}
		} else { // Safety check; some people might have no extension items in here.
			if !strings.HasSuffix(strings.ToLower(l.Name()), ".cache") {
				d.pipe = append(d.pipe, l.Name())
				d.execList = append(d.execList, path+l.Name())
			}
		}
	}

	for d.dir != nil {
		var arr string
		if len(d.dir) > 1 {
			arr = d.dir[0]
			d.dir = d.dir[1:len(d.dir)]
		} else {
			arr = d.dir[0]
			d.dir = nil
		}

		file, err := os.ReadDir(arr)
		if err == nil {
			dir := arr
			d.searchDir(dir, file)
		}
	}
}

func (d *dmenu) replaceHomeVars(s string) string {
	home := os.Getenv("HOME")
	if strings.HasPrefix(s, "~") {
		s = strings.Replace(s, "~", home, 1)
	}
	if strings.HasPrefix(s, "$HOME") {
		s = strings.Replace(s, "$HOME", home, 1)
	}
	return s
}

func (d *dmenu) readAliases() {
	f, err := os.ReadFile(os.Getenv("HOME") + "/.config/dmenu/aliases")
	if err != nil {
		fmt.Println(err)
		return
	}

	s := string(f)
	al := strings.SplitN(s[:len(s)-1], "\n", -1)
	for i := range al {
		if al[i][:2] == "//" {
			continue
		}
		if al[i][:1] == "$" {
			// Assure that any command containing < won't be touched!
			// Using just SpliN provides a total length of 1, this provides all.
			s := strings.SplitN(strings.Replace(al[i], "<", "\n", 1), "\n", -1)
			c := s[1]
			if c[len(c)-1:] == ">" {
				c = c[:len(c)-1]
			}
			d.alias = append(d.alias, s[0])
			d.command = append(d.command, c)
		}
	}
}

// Main worker.
func (d *dmenu) formDirsSlice() {
	var exeList []string
	exeList = append(exeList, os.Getenv("HOME")+"/.local/share/applications")
	exeList = append(exeList, strings.SplitN(os.Getenv("XDG_DATA_DIRS"), ":", -1)...)
	exeList = append(exeList, strings.SplitN(os.Getenv("PATH"), ":", -1)...)

	f, err := os.ReadFile(os.Getenv("HOME") + "/.config/dmenu/dirs")
	if err == nil {
		s := strings.SplitN(string(f), "\n", -1)
		for i := range s {
			s[i] = d.replaceHomeVars(s[i])
			exeList = append(exeList, s[i])
		}
	}

	// Make sure there are no repeats:
	var s string
	for i := range exeList {
		if !strings.Contains(s, exeList[i]) {
			s = s + exeList[i] + "\n"
		}
	}
	exeList = strings.SplitN(s[:len(s)-1], "\n", -1)

	var wg sync.WaitGroup

	lenExe := len(exeList)
	wg.Add(lenExe + 1)

	go func() {
		d.readAliases()
		wg.Done()
	}()

	dm := make([]dmenu, lenExe)
	for i := range exeList {
		dir := exeList[i]
		dirE, err := os.ReadDir(dir)
		go func(i int, dir string, dirE []os.DirEntry, err error) {
			if err == nil {
				dm[i].searchDir(dir, dirE)
			}
			wg.Done()
		}(i, dir, dirE, err)
	}

	wg.Wait()

	for i := range dm {
		d.pipe = append(d.pipe, dm[i].pipe...)
		d.execList = append(d.execList, dm[i].execList...)
	}

	// Dmenu pipe:
	for i := range d.pipe {
		d.strPipe += d.pipe[i] + "\n"
	}
}

func (d dmenu) run() {
	dCMD := false
	if len(os.Args) > 1 {
		for i := range os.Args {
			if strings.ToLower(os.Args[i]) == "-h" || strings.ToLower(os.Args[i]) == "--help" {
				fmt.Println("To successfully assign additional directories, make a file called \"dirs\" in ~/.config/dmenu/, and add your directories there.\nFor example: /home/username/Personal/bins/\nThis will cause the program to search through this directory for all file types (so make sure those are only executable files, like .desktop or regular bin like in /usr/bin/). The program will additionally look into \"applications\" folder if it is found.\n$PATH, $XDG_DATA_DIRS, ~/.local/applications are the default folders of the program (+ /applications if found).\nYou can use ~ and $HOME variables (in directory/alias prefixes only).\n\nYou can use -d flag, or --debug, to see execution stats - this is helpful when you encounter any errors.\n\nTo explicitly use a shell command, you can use $ prefix (e.g. $notify-send \"Josh was last seen in New York around 3 pm\"). You can omit it, though, it's helpful if you want to assure to execute a shell command.\n\nIt is possible to assign aliases, in ~/.config/dmenu/aliases (aliases is the file containing all aliases), similar like in shell, where you define them as following:\n$name<command>\nfor example: $info<notify-send>\nyou can then open Dmenu and type \"$info 'Johnathan was home at 11 am, but he left at 11:35.'\", and notify-send with this parameter would be triggered.\nCommenting out is done with double slash (//), and spaces are allowed, so you can add alias like:\n$not me! :)<notify-send 'This was not sent by James.'>\n\nYou can execute terminal applications if you add \"$term<terminal name -e>\" to \"config\" file in $HOME/.config/dmenu/, for example $term<alacritty -e> or $terminal<alacritty -e>. The \"-e\" is the terminal execution flag.")
				return
			}
			if !dCMD {
				dCMD = strings.ToLower(os.Args[i]) == "-d" || strings.ToLower(os.Args[i]) == "--debug"
			}
		}
	}

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "sh"
	}

	os.WriteFile("/tmp/dmenu_tmp", []byte(d.strPipe), 0666)
	cmd, _ := exec.Command(shell, "-c", "cat /tmp/dmenu_tmp | dmenu \"$@\"").CombinedOutput()
	s := string(cmd)
	go os.Remove("/tmp/dmenu_tmp") // No need to keep that file.

	if s == "" {
		fmt.Println("No command specified. Aborting.")
		return
	}

	stdout := s[:len(s)-1] //strings.SplitN(s, "\n", -1)[0]

	stdout = d.replaceHomeVars(stdout)

	for i := range d.alias {
		if strings.Contains(stdout, d.alias[i]) {
			stdout = strings.ReplaceAll(stdout, d.alias[i], d.command[i])
		}
	}

	if stdout[:1] != "$" {
		for i := range d.pipe {
			if d.pipe[i] == stdout {
				if dCMD {
					c, e := exec.Command(shell, "-c", d.execList[i]).CombinedOutput()
					fmt.Println("Debug message:\n Input:", stdout, "\n Command:", d.execList[i], "\n Output:", string(c), "\n Error:", e, "\n\n Executables:", len(d.pipe))
					return
				}
				exec.Command(shell, "-c", d.execList[i]+" &").Run()
				return
			}
		}
	} else {
		stdout = stdout[1:]
	}
	exec.Command(shell, "-c", stdout).Run()
}

func main() {
	c.getTerm()
	var d dmenu
	d.formDirsSlice()
	d.run()
}
