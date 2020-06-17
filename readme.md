# Remote Admin Agent

**RAagent is functioning but still beta and not fully tested on all platforms. Possibly still some bugs to iron or some fine tuning to do but feel free to try it, use it, expand on it...**

RAagent is one of two related small and simple tools, particularly useful for system administration of multiple servers and remote task automation. Those tools are the result of my first project in GO so the code design might not be the best but they do what they were designed to do just fine :-).  
Running services allowing remote program execution can represent a high security risk when not properly firewalled, make sure to setup appropriate firewall rules prior to deployment or do not use at all if you don't know what you are doing.

RAagent can be used standalone, in which case the **'modules'** can just be added manually and **'tasks'** can be executed by sending simple HTTPS POST requests from scripts or admin panels. If multiple agents are used, RAserver can act as the **'modules and binaries repository'** for the agents, making modules deployment and agents update easier. I have done my best to design those tools to run on Linux/Osx/Windows but only used them on production on Linux systems. The RAagent requires 'bash' to be installed on Windows to execute the commands/modules. 

Check out the [RAserver repository](https://github.com/miky4u2/RAserver) for the related RAserver code and info.

## Description
- RAagent is a Secure REST API service designed to accept **'tasks'**, requests from allowed origins and execute external **'modules'** on the machine it is running on (Windows/Linux/OSX). This can be useful for example to remotely restart services on servers, add users, trigger updates, download repositories, modify system files, manage dockers etc... The list of commands which can be executed is limited by the 'modules' you install on the agent.

- A **'module'** can be any executable, binary or script written in any language as long as it can be executed with **'bash'** and optionally accept arguments, some of which might need to be base64 encoded. 

- A **'task'** is the Json request for the execution of a specific module with provided arguments. A task can be executed in *attached* or *detached* mode (see below).
The API endpoint to create a task is `/tasks/new`
The API endpoint to check the progress of a *detached* task is `/tasks/status`

- The RAserver is optional and can act as the 'repository' for the agents making modules deployment and agents updates easier.. The RAserver holds the agent list as well as each agent's dedicated modules. When notified by the RAserver, the agents pull their modules, binary, config file and TLS certificate from the RAserver. This is a easy way to bulk add or remove modules from agents as well as bulk update TLS certificate or agent binaries.


## Task modes
- *attached* mode: the agent will execute the task by running the module and reply with the output of the execution (adapted to task that are quick to execute like restarting a service or updating a system file).
- *detached* mode: the agent will detach the *module* execution and return a UUID which can then be used to check the progress of the task via api call to `/tasks/status`. This is particularly useful for tasks that require an extended amount of time to complete and would otherwise cause a time out, like running a backup, synchronizing files, cloning repositories etc...
- Both modes accept an optional *notifyURL* parameter which is called when the task is completed. The notifyURL must be of type https:// and is instantly called when a new task is submitted in *attached* mode and deferred in *detached* mode. 

## Request Examples
New Task in *Detached* mode to `/tasks/new` (Method POST)
Request:
```json
{
    "name": "Restart Crond",
    "mode": "detached",
    "notifyURL": "https://www.blah.com/api/taskresponse",
    "module": "service_restart.sh",
    "args": [
        "crond"
    ]
}
```
Response:
```json
{
    "UUID": "8db6408d-c4e9-4144-92aa-46d1201a88d0",
    "status": "in progress",
    "errorMsgs": null,
    "startTime": "2020-05-28 23:11:36",
    "endTime": "",
    "duration": "",
    "output": "",
    "endPoint": "/tasks/new",
    "name": "Restart Crond",
    "mode": "detached",
    "notifyURL": "https://www.notifydomain.com/api/taskresponse",
    "module": "service_restart.sh",
    "args": [
        "crond"
    ]
}
```
Task Status to `/tasks/status` (Method POST)
Request:
```json
{
    "uuid": "8db6408d-c4e9-4144-92aa-46d1201a88d0"
}
```
Response (possible returned task *status* : *in progress*, *done*, *failed*), *output* is the base64 encoded output of the module execution:
```json
{
    "status": "ok",
    "errorMsgs": null,
    "task": {
        "UUID": "8db6408d-c4e9-4144-92aa-46d1201a88d0",
        "status": "done",
        "errorMsgs": null,
        "startTime": "2020-05-28 23:11:36",
        "endTime": "2020-05-28 23:11:37",
        "duration": "93.0322ms",
        "output": "cnVudGltZS9tb2R1bGVzL3NlcnZpY2VfcmVzdGFydC5zeXN0ZW1jdGw6IGNvbW1hbmQgbm90IGZvdW5kCg==",
        "endPoint": "/tasks/new",
        "name": "Restart Crond",
        "mode": "detached",
        "notifyURL": "https://www.notifydomain.com/api/taskresponse",
        "module": "service_restart.sh",
        "args": [
            "crond"
        ]
    }
}
```
## RAserver reserved api calls
- `/agent/update` 
    1. The server sends the agent an update request of type *full* or *modules*.
    2. The agent then sends a request to the server to get a tar.gz file with the updated modules and binary.
    3. The agent updates itself and restarts)
- `/agent/ctl` 
    1. The server can send control commands to the agent of type *status*, *restart* and *stop*

## RAagent config file (conf/config.json)

- **serverIP** : Array of one or more IPV4/IPV6. This is the IP of the RAserver. RAagent will validate the IP when receiving *update* and *ctl* requests(see above) from the RAserver. If no server is used, leave this field blank.  
- **serverURL** : Base URL of the RAserver, must be https. RAagent will request the tar.gz update archive from this URL. If no server is used, leave this field blank.
- **validateServerTLS** : If the RAserver uses a self signed TLS certificate, set this to *false*, otherwise set this to *true*.
- **agentID** : Unique identifier for this agent. Allowed characters [A to Z, a to z - _ .] 
- **agentBindPort** : Port the agent should bind on.
- **agentBindIP** : IP the agent should bind on. Leave blank to bind on all IPs.
- **allowedIPs** : Array of IPV4/IPV6 IP allowed to send requests to `/tasks/new` and `/tasks/status`
- **logToFile** : *true* to log to log file, *false* or blank to log to stdOut  
- **logFile** : When logToFile is *true*, either leave this field blank to use the default log file **log/agent.log** or provide a path to a log file. Use windows(\\) or linux(/) path format depending on which platform the agent is running on. 
- **validateNotifyTLS** : Use *false* of *NotifyURL* is used in task requests and the notified server is using a self signed certificate. Otherwise use *true*
- **taskHistoryKeepDays** : Number of days task statuses are kept and can be requested via `/tasks/status`


```json
{
    "serverIP": ["127.0.0.1","::1"],
    "serverURL": "https://localhost:8081",
    "validateServerTLS": false,
    "agentID": "example.pc",
    "agentBindPort": "8080",
    "agentBindIP": "",
    "allowedIPs": ["127.0.0.1","::1"],
    "logFile":"",
    "logToFile": true,
    "validateNotifyTLS": false,
    "taskHistoryKeepDays": 7
}
```
# Runtime deployment file layout
```
runtime
    |
    +--bin
    |    |
    |    +--agent (or agent.exe) agent executable, launched by startagent
    |    +--startagent (or startagent.exe) wrapper to start/restart the agent
    |
    +--conf
    |     |
    |     +--config.json (agent config file)
    |     +--cert.pem (agent TLS certificate & chains)
    |     +--key.pem (agent TLS certificate key)
    |   
    +--log
    |    |
    |    +--agent.log (default log file. Auto truncated when it reaches 500k)
    |
    +--modules
    |        |
    |        +--service_restart.sh (possible linux service restart script)
    |        +--hello.bat (possible windows module)
    |        +--some_module.exe (possible windows module)
    |        +--etc..
    |
    +--tasks (Required folder to keep the task status history)
    |
    +--temp (required temp folder to download and unpack updates from optional RAserver)


```
## Building RAagent

Both 'agent' and 'startagent' need to be built and placed in the /bin folder. The *makeosx.sh*, *makewindows.sh*, *makelinux.sh* can be used, alternatively this can be done as below. 

Note: For quiet background startup on Windows, both agent and startagent need building with *-ldflags -H=windowsgui*  (Be aware the agent process can then only be killed via the Windows task manager or using RAserver)
```
# Linux
#
GOOS=linux GOARCH=amd64 go build -o ./runtime/bin/startagent  ./startagent/startagent.go
GOOS=linux GOARCH=amd64 go build -o ./runtime/bin/agent  ./agent/agent.go

#Windows (Quiet start requires building both agent and startagent with -ldflags -H=windowsgui)
#
GOOS=windows GOARCH=amd64 go build -o ./runtime/bin/startagent.exe  ./startagent/startagent.go
GOOS=windows GOARCH=amd64 go build -o ./runtime/bin/agent.exe  ./agent/agent.go

#OSX (not tested)
#
GOOS=darwin GOARCH=amd64 go build -o ./runtime/bin/startagent  ./startagent/startagent.go
GOOS=darwin GOARCH=amd64 go build -o ./runtime/bin/agent  ./agent/agent.go

```

## Starting RAagent

The agent needs to be started with **startagent** or **startagent.exe** if used with RAserver as the wrapper is required to restart the agent after updates. In stand alone mode, either **agent** or **startagent** can be used to start the agent.

I am not sure how to start it at boot or as a service on Windows but below is a systemd service file that can be used for linux

```
[Unit]
Description = RAagent
After = network.target

[Service]
Type=forking
ExecStart = /path/to/RAagent/bin/startagent

[Install]
WantedBy = multi-user.target
```