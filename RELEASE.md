# Release Notes

## 0.2.1

This patch release fixes a [breaking api change](https://github.com/satori/go.uuid/commit/0ef6afb2f6cdd6cdaeee3885a95099c63f18fc8c) in the satori/go.uuid package where the function signature changed for `uuid.NewV4()`.

## 0.2.0

This release primarily focuses on the Command Log Persistence feature which allows the recovery of a workq-server.

* Added Command Log Persistence! Docs available at [doc/cmdlog](doc/cmdlog.md).
* Added TTR to "lease" command replies.
    * Allows for workers to use TTR as the maximum execution time for the specific job.
* Changed error "-TIMED-OUT" to "-TIMEOUT" for consistency.
* Fixed "run" job expiration issue on successful execution.
    * "run" commands did not always clean the completed job up after command returns.
* Fixed "lease" timeout priority and accuracy for lower timeouts (e.g. 10ms).
    * 10ms timeouts would return a -TIMEOUT intermittently even if there is a job available.
* Removed "log-file" option, all errors now direct to STDERR.

### Testing

* Combined test coverage is now at 97.048%.
* Race detector enabled.
* Additional system smoke test added for Command Log which generates 1k jobs, restarts workq-server, and verifies all expected jobs.

## 0.1.0

2016-08-23

First! Initial release.
