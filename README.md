# shnotify
zsh-hooks based watcher for commands execution written in Go

## Details
This utility implements two calls: register an invocation (save-invocation mode) and inform (notify mode) that invocation has finished its work based on some configurable conditions

### Usage
Possible option of integration of shnotify into your zsh config is shown in zsrch-hook-example.txt file

### Nearest plans
 - [ ] Support notifications with Telegram bot
 - [ ] Support Linux notifications with CGO libnotify
 - [ ] Support direct call to setup a hook on a single command execution
 - [ ] Abstract notifiers
 - [x] Abstract storage
 - [ ] Support non-file storage for invocations (sqlite for example)
 - [ ] Support allow lists and ban lists for the programs
 - [ ] Scan executing line for secrets and prevent them to be stored and included into the notification
