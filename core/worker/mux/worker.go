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
	"context"
	"errors"
	"log"
	"os"
	"runtime"
	"sync"
)

type Worker struct {
	id           uint
	wg           *sync.WaitGroup
	workDirPath  string
	inputStream  <-chan common.Task
	outputStream chan common.Task
	isRunning    bool
}

func NewMuxWorker(wg *sync.WaitGroup, param *common.Parameter, inputStream <-chan common.Task, id uint) (*Worker, error) {
	if _, err := os.Stat(param.WorkDirPath); os.IsNotExist(err) {
		return nil, errors.New("work dir path not exist")
	}

	w := Worker{
		id:           id,
		wg:           wg,
		workDirPath:  param.WorkDirPath,
		inputStream:  inputStream,
		outputStream: make(chan common.Task),
		isRunning:    false,
	}

	return &w, nil
}

func (w *Worker) Start(ctx context.Context) error {
	if w.isRunning == true {
		return errors.New("mux worker already running")
	}

	w.isRunning = true
	w.wg.Add(1)
	go w.workerLoop(ctx)
	log.Printf("[info] mux worker #%d started\n", w.id)

	return nil
}

func (w *Worker) GetOutputStream() chan common.Task {
	return w.outputStream
}

func (w *Worker) workerLoop(ctx context.Context) {
	defer func() {
		w.isRunning = false
		w.wg.Done()
		log.Printf("[info] mux worker #%d exited\n", w.id)
	}()

	exitFlag := false
	for {
		if exitFlag == true {
			log.Printf("[info] mux worker #%d receive exit signal\n", w.id)
			break
		}

		select {
		case <-ctx.Done():
			exitFlag = true
			continue
		case task := <-w.inputStream:
			if task.Mux == "" {
				log.Printf("[info] mux worker bypass task: %s\n", task.Src)
			} else {
				log.Printf("[info] mux worker handle task: %s\n", task.Src)
				err := w.handleNewTask(ctx, &task)
				if err != nil {
					log.Printf("[error] mux worker %d encounter error during handle task %s: %s\n", w.id, task.Src, err.Error())
					continue
				}
			}

			select {
			case <-ctx.Done():
				exitFlag = true
				continue
			case w.outputStream <- task:
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
