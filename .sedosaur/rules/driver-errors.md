+++
description = "Driver code returns errors instead of logging them"
paths = ["driver/**"]
+++

Driver functions return errors to their caller instead of logging and going on. An error
that the task should retry, such as a lost connection to hardware, matches
`errors::TEMPORARY_HARDWARE_ERROR`: the read task in `driver/common/read_task.h` then
warns and retries. Report an error that is only logged, or a temporary hardware fault
returned as a critical error.
