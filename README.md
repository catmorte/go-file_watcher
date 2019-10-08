# go-file_watcher

Check files' (not directories) changes every N

### How to use

Call watcher.Watch(time.Duration, files ...string)

"Watch" function returns an instance of watch-session which contains the following:

Session's field:
 - C - channel which returns WatchRessult instance

Session's methods:
 - Stop - to stop watching
 - UpdateWatched - to change list of files to listen
 
WatchResult's fields:
 - Old - map of old existing files to their sha256-hashes
 - New - map of new existing files to their sha256-hashes
