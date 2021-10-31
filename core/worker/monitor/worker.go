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

package monitor

import (
	"MonitorEncoder/core/common"
	"MonitorEncoder/core/status"
	"MonitorEncoder/core/worker"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Worker struct {
	*worker.Base
	monitorPath string
	recyclePath string
	workDirPath string
}

func NewMonitor(wg *sync.WaitGroup, param *common.Parameter, id uint) *Worker {
	m := Worker{
		Base:        worker.NewWorkerBase(wg, id),
		monitorPath: param.MonitorDirPath,
		recyclePath: filepath.Join(param.MonitorDirPath, "recycle"),
		workDirPath: param.WorkDirPath,
	}

	return &m
}

func (w *Worker) Start(ctx context.Context) error {
	if w.IsRunning == true {
		return errors.New("monitor already running")
	}

	if _, err := os.Stat(w.monitorPath); os.IsNotExist(err) {
		return errors.New("monitor path not exist")
	}

	if _, err := os.Stat(w.workDirPath); os.IsNotExist(err) {
		return errors.New("work dir path not exist")
	}

	if _, err := os.Stat(w.recyclePath); os.IsNotExist(err) {
		err := os.Mkdir(w.recyclePath, 0777)
		if err != nil {
			return errors.New("failed to create recycle folder: " + err.Error())
		}
	}

	w.IsRunning = true
	w.Wg.Add(1)
	go w.workerLoop(ctx)
	log.Printf("[info] %s started\n", w.GetPrettyName())

	return nil
}

func (w *Worker) GetPrettyName() string {
	return fmt.Sprintf("monitor #%d", w.Id)
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

		newTaskPath := w.checkNewTask(ctx)
		if newTaskPath != "" {
			newTask, err := common.NewTaskFromJson(newTaskPath)

			if err != nil {
				log.Printf("[error] %s failed to load task: %s: %s\n", w.GetPrettyName(), newTaskPath, err.Error())
				err = common.MoveFile(ctx, newTaskPath, w.recyclePath)
				if err != nil {
					log.Printf("[error] %s failed to move bad task to recycle bin: %s\n", w.GetPrettyName(), err.Error())
				}
				continue
			}

			newTask.TaskFile = newTaskPath

			log.Printf("[info] %s load new task: %s\n", w.GetPrettyName(), newTaskPath)
			status.SetStatusCode(newTask.Src, status.WAIT)
			status.SetStatusDesc(newTask.Src, "waiting")

			select {
			case <-ctx.Done():
				exitFlag = true
				continue
			case w.OutputStream <- *newTask:
				continue
			}
		}

		select {
		case <-ctx.Done():
			exitFlag = true
			continue
		case <-time.After(1 * time.Second):
			continue
		}
	}
}

func (w *Worker) checkNewTask(ctx context.Context) string {
	fileInfoList, err := ioutil.ReadDir(w.monitorPath)
	if err != nil {
		return ""
	}

	var newTaskPath string
	for _, fileInfo := range fileInfoList {
		fileName := fileInfo.Name()
		fileExt := filepath.Ext(fileName)
		if fileExt == ".json" {
			srcPath := filepath.Join(w.monitorPath, fileName)
			dstPath := w.workDirPath
			err = common.MoveFile(ctx, srcPath, dstPath)
			if err == nil {
				newTaskPath = filepath.Join(dstPath, fileName)
				break
			} else {
				log.Printf("[error] %s failed to move file: %s\n", w.GetPrettyName(), err.Error())
				continue
			}
		}
	}

	return newTaskPath
}
