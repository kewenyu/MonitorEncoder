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

package final

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
	id            uint
	wg            *sync.WaitGroup
	workDirPath   string
	outputDirPath string
	inputStream   <-chan common.Task
	outputStream  chan common.Task
	isRunning     bool
}

func NewFinalWorker(wg *sync.WaitGroup, param *common.Parameter, inputStream <-chan common.Task, id uint) (*Worker, error) {
	if _, err := os.Stat(param.WorkDirPath); os.IsNotExist(err) {
		return nil, errors.New("work dir path not exist")
	}

	if _, err := os.Stat(param.OutputDirPath); os.IsNotExist(err) {
		return nil, errors.New("output dir path not exist")
	}

	w := Worker{
		id:            id,
		wg:            wg,
		workDirPath:   param.WorkDirPath,
		outputDirPath: param.OutputDirPath,
		inputStream:   inputStream,
		outputStream:  make(chan common.Task),
		isRunning:     false,
	}

	return &w, nil
}

func (w *Worker) Start(ctx context.Context) error {
	if w.isRunning == true {
		return errors.New("final worker already running")
	}

	w.isRunning = true
	w.wg.Add(1)
	go w.workerLoop(ctx)
	log.Printf("[info] final worker #%d started\n", w.id)

	return nil
}

func (w *Worker) GetOutputStream() chan common.Task {
	return w.outputStream
}

func (w *Worker) workerLoop(ctx context.Context) {
	defer func() {
		w.isRunning = false
		w.wg.Done()
		log.Printf("[info] final worker #%d exited\n", w.id)
	}()

	exitFlag := false
	for {
		if exitFlag == true {
			log.Printf("[info] final worker #%d receive exit signal\n", w.id)
			break
		}

		select {
		case <-ctx.Done():
			exitFlag = true
			continue
		case task := <-w.inputStream:
			log.Printf("[info] final worker #%d handle task: %s\n", w.id, task.Src)
			err := w.handleNewTask(ctx, &task)
			if err != nil {
				log.Printf("[error] final worker #%d encounter error during handle task %s: %s\n", w.id, task.Src, err.Error())
				continue
			}
			log.Printf("[info] final worker #%d finish task: %s\n", w.id, task.Src)
		}

		runtime.Gosched()
	}
}

func (w *Worker) handleNewTask(ctx context.Context, task *common.Task) error {
	srcFile := task.Src
	status.SetStatusCode(srcFile, status.FINAL)
	status.SetStatusDesc(srcFile, "copying output files")

	err := common.MoveFile(ctx, task.ScriptFile, w.outputDirPath)
	if err != nil {
		status.SetStatusCode(srcFile, status.ERROR)
		status.SetStatusDesc(srcFile, "failed to move " + task.ScriptFile)
		return err
	}

	err = common.MoveFile(ctx, task.TaskFile, w.outputDirPath)
	if err != nil {
		status.SetStatusCode(srcFile, status.ERROR)
		status.SetStatusDesc(srcFile, "failed to move " + task.TaskFile)
		return err
	}

	resultList := task.GetResultList()

	if task.MuxedFile != "" {
		for _, result := range resultList {
			err = common.DeleteFile(ctx, result.Path)
			if err != nil {
				status.SetStatusCode(srcFile, status.ERROR)
				status.SetStatusDesc(srcFile, "failed to delete " + result.Path)
				return err
			}
		}

		err = common.MoveFile(ctx, task.MuxedFile, w.outputDirPath)
		if err != nil {
			status.SetStatusCode(srcFile, status.ERROR)
			status.SetStatusDesc(srcFile, "failed to move " + task.MuxedFile)
			return err
		}
	} else {
		for _, result := range resultList {
			err = common.MoveFile(ctx, result.Path, w.outputDirPath)
			if err != nil {
				status.SetStatusCode(srcFile, status.ERROR)
				status.SetStatusDesc(srcFile, "failed to move " + result.Path)
				return err
			}
		}
	}

	status.SetStatusCode(srcFile, status.DONE)
	status.SetStatusDesc(srcFile, "everything is finished")

	return nil
}
