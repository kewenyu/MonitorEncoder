/*
 * MonitorEncoder
 * Copyright (C) 2021  kewenyu
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package main

import (
	"MonitorEncoder/core/common"
	"MonitorEncoder/core/status"
	"MonitorEncoder/core/worker/misc"
	"MonitorEncoder/core/worker/monitor"
	"MonitorEncoder/core/worker/video"
	"MonitorEncoder/web"
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
)

func main() {
	var (
		workerNum      = flag.Int("n", 1, "worker num")
		monitorDirPath = flag.String("md", "monitor_dir", "monitor dir")
		workDirPath    = flag.String("wd", "work_dir", "work dir")
		outputDirPath  = flag.String("od", "output_dir", "output dir")
		ip             = flag.String("ip", "127.0.0.1", "web interface's ip")
		port           = flag.String("port", "8899", "web interface's port")
	)
	flag.Parse()

	stop := make(chan int)
	var wg sync.WaitGroup

	monitorWorker, err := monitor.NewMonitor(stop, &wg, *monitorDirPath, *workDirPath, 0)
	if err != nil {
		log.Printf("[fatal] failed to create monitor worker: %s\n", err.Error())
		return
	}
	err = monitorWorker.Start()
	if err != nil {
		fmt.Printf("[fatal] failed to start monitor worker: %s\n", err.Error())
		return
	}

	videoWorkerList := make([]*video.Worker, 0)
	for i := 0; i < *workerNum; i++ {
		videoWorker, err := video.NewVideoWorker(stop, &wg, *workDirPath, monitorWorker.GetOutputStream(), uint(i))
		if err != nil {
			fmt.Printf("[fatal] failed to create video worker #%d: %s\n", i, err.Error())
			return
		}

		err = videoWorker.Start()
		if err != nil {
			fmt.Printf("[fatal] failed to start video worker #%d: %s\n", i, err.Error())
			return
		}

		videoWorkerList = append(videoWorkerList, videoWorker)
	}

	fanInVideoWorkerOutputStream := make(chan common.Task)
	for _, videoWorker := range videoWorkerList {
		go func(inputChan chan common.Task) {
			for {
				select {
				case <-stop:
					return
				case task := <-inputChan:
					fanInVideoWorkerOutputStream <- task
					continue
				}
			}
		}(videoWorker.GetOutputStream())
	}

	miscWorker, err := misc.NewMiscWorker(stop, &wg, *workDirPath, *outputDirPath, fanInVideoWorkerOutputStream, 0)
	if err != nil {
		log.Printf("[fatal] failed to create misc worker: %s\n", err.Error())
		return
	}
	err = miscWorker.Start()
	if err != nil {
		fmt.Printf("[fatal] failed to start misc worker: %s\n", err.Error())
		return
	}

	err = web.StartWeb(*ip, *port, *monitorDirPath)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	userInputStream := make(chan string)
	go func() {
		for {
			select {
			case <-stop:
				return
			default:
			}

			buffer := bufio.NewReader(os.Stdin)
			line, err := buffer.ReadString('\n')
			if err != nil {
				fmt.Printf("input error: %s\n", err.Error())
			}

			userInputStream <- strings.TrimSpace(line)
		}
	}()

mainLoop:
	for {
		select {
		case userInput := <-userInputStream:
			if userInput == "status" {
				status.PrintAllStatus()
				continue
			} else if userInput == "stop" {
				log.Println("[info] user stop")
				break mainLoop
			} else {
				fmt.Printf("unknown command: %s\n", userInput)
				continue
			}
		}
	}

	close(stop)
	wg.Wait()

	log.Println("[info] all finish")
}
