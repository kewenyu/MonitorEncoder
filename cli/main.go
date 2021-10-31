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
	"MonitorEncoder/core/worker"
	"MonitorEncoder/core/worker/final"
	"MonitorEncoder/core/worker/misc"
	"MonitorEncoder/core/worker/monitor"
	"MonitorEncoder/core/worker/mux"
	"MonitorEncoder/core/worker/video"
	"MonitorEncoder/web"
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
)

func main() {
	param := common.Parameter{}
	flag.IntVar(&param.WorkerNum, "n", 1, "worker num")
	flag.StringVar(&param.MonitorDirPath, "md", "monitor_dir", "monitor dir")
	flag.StringVar(&param.WorkDirPath, "wd", "work_dir", "work dir")
	flag.StringVar(&param.OutputDirPath, "od", "output_dir", "output dir")
	flag.StringVar(&param.Ip, "ip", "127.0.0.1", "web interface's ip")
	flag.StringVar(&param.Port, "port", "8899", "web interface's port")
	flag.Parse()

	err := common.CheckToolsAvailability()
	if err != nil {
		log.Printf("[fatal] external tool not available: %s", err.Error())
		return
	}

	var wg sync.WaitGroup
	workerSequence := []worker.Worker{
		monitor.NewMonitor(&wg, &param, 0),
		video.NewMultiWorker(&wg, &param, 0),
		misc.NewMiscWorker(&wg, &param, 0),
		mux.NewMuxWorker(&wg, &param, 0),
		final.NewFinalWorker(&wg, &param, 0),
	}

	var upperStream chan common.Task
	for _, w := range workerSequence {
		w.SetInputStream(upperStream)
		upperStream = w.GetOutputStream()
	}

	mainCtx, mainCtxCancelFunc := context.WithCancel(context.Background())
	defer mainCtxCancelFunc()

	var startErr error
	for _, w := range workerSequence {
		startErr = w.Start(mainCtx)
		if startErr != nil {
			log.Printf("[fatal] failed to start %s: %s\n", w.GetPrettyName(), err.Error())
			return
		}
	}

	err = web.StartWeb(&param)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	userInputStream := make(chan string)
	go func() {
		for {
			select {
			case <-mainCtx.Done():
				return
			default:
			}

			buffer := bufio.NewReader(os.Stdin)
			line, err := buffer.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					return
				}
				fmt.Printf("input error: %s\n", err.Error())
				continue
			}

			userInputStream <- strings.TrimSpace(line)
		}
	}()

	signalInt := make(chan os.Signal, 1)
	signal.Notify(signalInt, os.Interrupt)

mainLoop:
	for {
		select {
		case <-signalInt:
			log.Println("[info] receive SIGINT")
			break mainLoop
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

	mainCtxCancelFunc()
	wg.Wait()

	log.Println("[info] all finish")
}
