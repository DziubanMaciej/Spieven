# Spieven
*Spieven* is a process supervisor for Linux. While this may sound daunting, a process supervisor is simply a process for running and managing other processes, here called *tasks*. Something like *systemd*, but simpler to use. *Spieven* is meant to be used in scripts trivially - without countless configuration files, myriads of options, and parsers.



# Display-awareness
*Spieven* is written to be display-aware. This means it treats a display session used by a task (X11, Wayland or no display) as a first-class citizen. This is contrary to *systemd*, which is oblivious to displays and can make it difficult to properly launch graphical tasks. *Spieven* can detect task's display, force a specific display or list all tasks running on a given display. It also internally monitors (pun not intended) whether the displays used by its tasks are running and terminates the tasks whenever display servers are closed.



# Usage
This application is CLI-only. It exposes various commands, such as `spieven schedule`. The `schedule` command is by far the most important. It registers a task to be run along with all additional parameters, including timeout between executions, maximum number of failures, a name to easily identify the task, or the display to run the task on.

Tasks can be deactivated for various reasons, meaning *Spieven* will stop running them. It does not forget about them, however. The logs can be inspected with `spieven peek TASK_ID`, and the tasks can be reactivated with the `spieven reschedule TASK_ID` command.

Tasks can be queried with `spieven list`. This command returns various metadata about all active tasks and optionally inactive tasks as well. This command also supports `--json` switch to serialize all data into JSON, making it easily parseable in scripts.

Typically *Spieven* tasks should be scheduled in a script that is run once per display init, for example `.xinitrc` or `~/.config/autostart/*.desktop` files.



# Examples
Schedule a task to display a GUI notification every 2 seconds:
```
spieven schedule -s 2000 notify-send "Hello world"
```

Cancel the 2 second wait and display the notification immediately (assuming the task ID was 0):
```
spieven refresh -i 0
```

Schedule a task to try to run `picom` on Xorg display `:2`, but try only 3 times. After 3 failures, deactivate the task:
```
spieven schedule -p x:2 -m 3 picom
```

List tasks running on Xorg display `:2`, including deactivated tasks:
```
spieven list -p x:2 -D
```

Inspect logs for the task with ID 3:
```
spieven peek 3
```

Reactivate the task with the same parameters:
```
spieven reschedule 3
```

Get the help message with all available options:
```
spieven -h
spieven schedule -h
spieven list -h
```


# Architecture
Internally *Spieven* works in a client-server architecture, here called frontend and backend. The frontend and backend connect via a local TCP socket. All commands such as `spieven schedule`, `spieven list`, `spieven refresh`, etc. are considered frontend commands. The backend is run by the `spieven serve` command, but generally it does not have to be manually started by the user, because frontend commands automatically launch the backend if it is not running. Alternatively, it could be run with an OS process supervisor, such as systemd, but there is no real need for that.

The majority of *Spieven* logic lives in the backend, which manages and runs the tasks, caches the results, monitors display state, and handles frontend commands. Frontend commands mainly convert command-line arguments to TCP packets and send them to the backend. Most of the frontend commands exit immediately after sending a packet to the backend and receiving a response. For example, if the `spieven schedule` command exits immediately, it does not mean the task has ended. It is running in the background as a backend's subprocess.

It is worth noting that *Spieven*'s frontend and backend are really the very same binary file and *Spieven* verifies that. There is no compatibility support in the communication protocol between them. Only command-line interface is meant to be stable.


# Installing
## From binaries:
Download from [GitHub releases](https://github.com/DziubanMaciej/Spieven/releases)

## From source
Install `go` compiler toolchain. Refer to your distribution's package manager.
Run `go build -tags user` and grab the `spieven` binary file.
Alternatively, add `$GOPATH/bin` into your `PATH` and run `go install -tags user`, which will build and install `spieven` binary in `$GOPATH/bin`.
