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
	workDirPath   string
	outputDirPath string
}

func NewFinalWorker(wg *sync.WaitGroup, param *common.Parameter, id uint) *Worker {
	w := Worker{
		Base:          worker.NewWorkerBase(wg, id),
		workDirPath:   param.WorkDirPath,
		outputDirPath: param.OutputDirPath,
	}

	return &w
}

func (w *Worker) Start(ctx context.Context) error {
	if w.IsRunning == true {
		return errors.New("final worker already running")
	}

	if _, err := os.Stat(w.workDirPath); os.IsNotExist(err) {
		return errors.New("work dir path not exist")
	}

	if _, err := os.Stat(w.outputDirPath); os.IsNotExist(err) {
		return errors.New("output dir path not exist")
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
	return fmt.Sprintf("final worker #%d", w.Id)
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
			log.Printf("[info] %s handle task: %s\n", w.GetPrettyName(), task.Src)
			err := w.handleNewTask(ctx, &task)
			if err != nil {
				log.Printf("[error] %s encounter error during handle task %s: %s\n", w.GetPrettyName(), task.Src, err.Error())
				continue
			}
			log.Printf("[info] %s finish task: %s\n", w.GetPrettyName(), task.Src)
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
		status.SetStatusDesc(srcFile, "failed to move "+task.ScriptFile)
		return err
	}

	err = common.MoveFile(ctx, task.TaskFile, w.outputDirPath)
	if err != nil {
		status.SetStatusCode(srcFile, status.ERROR)
		status.SetStatusDesc(srcFile, "failed to move "+task.TaskFile)
		return err
	}

	resultList := task.GetResultList()

	if task.MuxedFile != "" {
		for _, result := range resultList {
			err = common.DeleteFile(ctx, result.Path)
			if err != nil {
				status.SetStatusCode(srcFile, status.ERROR)
				status.SetStatusDesc(srcFile, "failed to delete "+result.Path)
				return err
			}
		}

		err = common.MoveFile(ctx, task.MuxedFile, w.outputDirPath)
		if err != nil {
			status.SetStatusCode(srcFile, status.ERROR)
			status.SetStatusDesc(srcFile, "failed to move "+task.MuxedFile)
			return err
		}
	} else {
		for _, result := range resultList {
			err = common.MoveFile(ctx, result.Path, w.outputDirPath)
			if err != nil {
				status.SetStatusCode(srcFile, status.ERROR)
				status.SetStatusDesc(srcFile, "failed to move "+result.Path)
				return err
			}
		}
	}

	status.SetStatusCode(srcFile, status.DONE)
	status.SetStatusDesc(srcFile, "everything is finished")

	return nil
}
