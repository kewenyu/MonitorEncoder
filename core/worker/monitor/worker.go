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
	"context"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Worker struct {
	id           uint
	wg           *sync.WaitGroup
	monitorPath  string
	recyclePath  string
	workDirPath  string
	outputStream chan common.Task
	isRunning    bool
}

func NewMonitor(wg *sync.WaitGroup, monitorPath string, workDirPath string, id uint) (*Worker, error) {
	if _, err := os.Stat(monitorPath); os.IsNotExist(err) {
		return nil, errors.New("monitor path not exist")
	}

	if _, err := os.Stat(workDirPath); os.IsNotExist(err) {
		return nil, errors.New("work dir path not exist")
	}

	m := Worker{
		id:           id,
		wg:           wg,
		monitorPath:  monitorPath,
		recyclePath:  filepath.Join(monitorPath, "recycle"),
		workDirPath:  workDirPath,
		outputStream: make(chan common.Task),
		isRunning:    false,
	}

	return &m, nil
}

func (w *Worker) Start(ctx context.Context) error {
	if w.isRunning == true {
		return errors.New("monitor already running")
	}

	if _, err := os.Stat(w.recyclePath); os.IsNotExist(err) {
		err := os.Mkdir(w.recyclePath, 0777)
		if err != nil {
			return errors.New("failed to create recycle folder: " + err.Error())
		}
	}

	w.isRunning = true
	w.wg.Add(1)
	go w.workerLoop(ctx)
	log.Printf("[info] monitor #%d started\n", w.id)

	return nil
}

func (w *Worker) GetOutputStream() chan common.Task {
	return w.outputStream
}

func (w *Worker) workerLoop(ctx context.Context) {
	defer func() {
		w.isRunning = false
		w.wg.Done()
		log.Printf("[info] monitor #%d exited\n", w.id)
	}()

	exitFlag := false
	for {
		if exitFlag == true {
			log.Printf("[info] monitor #%d receive exit signal\n", w.id)
			break
		}

		select {
		case <-ctx.Done():
			exitFlag = true
			continue
		default:
		}

		newTaskPath := w.checkNewTask(ctx)
		if newTaskPath != "" {
			newTask, err := common.NewTaskFromJson(newTaskPath)

			if err != nil {
				log.Printf("[error] monitor #%d failed to load task: %s: %s\n", w.id, newTaskPath, err.Error())
				err = common.MoveFile(ctx, newTaskPath, w.recyclePath)
				if err != nil {
					log.Printf("[error] monitor #%d failed to move bad task to recycle bin: %s\n", w.id, err.Error())
				}
				continue
			}

			newTask.TaskFile = newTaskPath

			log.Printf("[info] monitor #%d load new task: %s\n", w.id, newTaskPath)
			status.SetStatusCode(newTask.Src, status.WAIT)
			status.SetStatusDesc(newTask.Src, "waiting")

			select {
			case <-ctx.Done():
				exitFlag = true
				continue
			case w.outputStream <- *newTask:
				continue
			}
		}

		time.Sleep(1 * time.Second)
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
				log.Printf("[error] monitor #%d failed to move file: %s\n", w.id, err.Error())
				continue
			}
		}
	}

	return newTaskPath
}
