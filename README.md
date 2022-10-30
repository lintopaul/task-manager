# Task Manager
Task Manager is an API to create and manage multiple tasks and allows pausing and resuming long running tasks. 

Tasks expire after a predetermined amount of time. A task can be terminated(killed) manually as well.

Concurrency is achieved using goroutines.

## Routes
- GET /create

    Spawns a new task. The response from the api has a uuid which can be used as reference for performing various actions related to the spawned task.

- GET /pause/{uuid}

    Pauses the execution of task referenced by the given uuid.

- GET /resume/{uuid}
    
    Resumes the execution of a paused task referenced by given uuid.

- GET /delete/{uuid}
    
    Kills the task gracefully and calls a function to perform proper rollback.

## Run the app

```
$ go build
$ ./task-manager 
task-manager-api 2022/10/30 22:44:54 Starting server on port 9090
```

You may also run as a docker container by building an image from the Dockerfile provided.

