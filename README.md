# shnotify
zsh-hooks based watcher for commands execution written in Go

## Details
This utility implements two calls: register an invocation (save-invocation mode) and inform (notify mode) that invocation has finished its work based on some configurable conditions

### Usage
Possible option of integration of shnotify into your zsh config is shown in zsrch-hook-example.txt file

### Nearest plans
 - [x] Support notifications with Telegram 
 - [x] Abstract notifiers
 - [x] Abstract storage
 - [x] Implement client-server mode (add shnotifyd service) and move command implementation there
 - [x] Sync/Async notification
 - [ ] Add machine id to the stored invocation
 - [ ] Add logging
 - [ ] Support Linux notifications with CGO libnotify
 - [ ] Support allow lists and ban lists for the programs (add shell parser)
 - [ ] Support direct call to monitor a single command execution (without setting up shell hook)
 - [ ] Support non-file storage for invocations (sqlite for example)
 - [ ] Scan executing line for secrets and prevent them to be stored and included into the notification
 - [ ] Implement autocleaner for storage

### Source packages
 - [ ] Linux OS push notifications (CGO required) - https://github.com/GNOME/libnotify
 - [ ] Shell parser - https://github.com/mvdan/sh
 - [x] Notifiers - https://github.com/nikoksr/notify
