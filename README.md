# MonitorEncoder

## Introduction

MonitorEncoder is a simple batch/remote encoding tools which monitor a given directory for upcoming BDMV encoding tasks and perform encoding task according to respective task config file (in JSON format).

Since it primarily focuses on BDMV transcoding, the input file is assumed to be m2ts.

### Features

* basic vpy script generation based on given templates
* multiformat encoding/demuxing
    * video encoding: HEVC, AVC
    * audio encoding: FLAC, OPUS, AAC
    * demuxing: anything supported by eac3to
* multiple workers
* basic http interface for remote encoding

## Install

```
cd MonitorEncoder
go build -o build\MonitorEncoder.exe .\cli
```

## Usage

### Command line arguments

* -n: video encoding workers num (default: 1)
* -md: monitor dir path (default: "monitor_dir")
* -wd: work directory path (default: "work_dir")
* -od: output directory path (default: "output_dir")
* -ip: ip address that http interface listening on (default: "127.0.0.1")
* -port: port for http interface (default: "8899")

### Interactive Command

* status
    * print current status on command prompt
* stop
    * stop the program gracefully

### Http Interface

Caution: no authentication yet! Use in the local network only.

* GET /status
    * show current tasks' status
* GET /api/status
    * return all tasks' status in json
* POST /api/newtask
    * submit new task

### Environment Variable

* MONITOR_ENCODER_BIN_PATH: The root directory which contains external tools
    * %MONITOR_ENCODER_BIN_PATH%\eac3to\eac3to.exe
    * %MONITOR_ENCODER_BIN_PATH%\vspipe.exe
    * %MONITOR_ENCODER_BIN_PATH%\x265-10b.exe
    * %MONITOR_ENCODER_BIN_PATH%\x264_64.exe
    * %MONITOR_ENCODER_BIN_PATH%\opusenc.exe
    * %MONITOR_ENCODER_BIN_PATH%\qaac.exe

### VapourSynth Template

refer to example\example_template.vpy

### Task Config

refer to example\example_task.json

