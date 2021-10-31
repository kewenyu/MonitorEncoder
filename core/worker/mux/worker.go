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

package mux

import (
	"MonitorEncoder/core/common"
	"MonitorEncoder/core/status"
	"MonitorEncoder/core/worker"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"
)

type Worker struct {
	*worker.Base
	workDirPath string
}

func NewMuxWorker(wg *sync.WaitGroup, param *common.Parameter, id uint) *Worker {
	w := Worker{
		Base:        worker.NewWorkerBase(wg, id),
		workDirPath: param.WorkDirPath,
	}

	return &w
}

func (w *Worker) Start(ctx context.Context) error {
	if w.IsRunning == true {
		return errors.New("mux worker already running")
	}

	if _, err := os.Stat(w.workDirPath); os.IsNotExist(err) {
		return errors.New("work dir path not exist")
	}

	if w.InputStream == nil {
		return errors.New("input stream not set")
	}

	w.IsRunning = true
	w.Wg.Add(1)
	go w.workerLoop(ctx)
	log.Printf("[info] %s started\n", w.GetPrettyName())

	return nil
}

func (w *Worker) GetPrettyName() string {
	return fmt.Sprintf("mux worker #%d", w.Id)
}

func (w *Worker) workerLoop(ctx context.Context) {
	defer func() {
		w.IsRunning = false
		w.Wg.Done()
		log.Printf("[info] %s exited\n", w.GetPrettyName())
	}()

	exitFlag := false
	for {
		if exitFlag == true {
			log.Printf("[info] %s receive exit signal\n", w.GetPrettyName())
			break
		}

		select {
		case <-ctx.Done():
			exitFlag = true
			continue
		case task := <-w.InputStream:
			if task.Mux == "" {
				log.Printf("[info] %s bypass task: %s\n", w.GetPrettyName(), task.Src)
			} else {
				log.Printf("[info] %s handle task: %s\n", w.GetPrettyName(), task.Src)
				err := w.handleNewTask(ctx, &task)
				if err != nil {
					log.Printf("[error] %s encounter error during handle task %s: %s\n", w.GetPrettyName(), task.Src, err.Error())
					continue
				}
			}

			select {
			case <-ctx.Done():
				exitFlag = true
				continue
			case w.OutputStream <- task:
			}
		}

		runtime.Gosched()
	}
}

func (w *Worker) handleNewTask(ctx context.Context, task *common.Task) error {
	srcFile := task.Src

	status.SetStatusCode(srcFile, status.MUX)
	status.SetStatusDesc(srcFile, "muxing " + task.Mux)

	formatHandler, exist := formatHandlerMap[task.Mux]
	if !exist {
		errDesc := "unknown mux format: " + task.Mux
		status.SetStatusCode(srcFile, status.ERROR)
		status.SetStatusDesc(srcFile, errDesc)
		return errors.New(errDesc)
	}

	outputPath, err := formatHandler(ctx, w.workDirPath, task)
	if err != nil {
		status.SetStatusCode(srcFile, status.ERROR)
		status.SetStatusDesc(srcFile, err.Error())
		return err
	}

	task.MuxedFile = outputPath

	return nil
}
