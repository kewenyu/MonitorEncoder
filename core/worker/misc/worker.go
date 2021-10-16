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

package misc

import (
	"MonitorEncoder/core/common"
	"MonitorEncoder/core/status"
	"errors"
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"
)

type Worker struct {
	id            uint
	stop          <-chan int
	wg            *sync.WaitGroup
	workDirPath   string
	outputDirPath string
	inputStream   <-chan common.Task
	isRunning     bool
}

func NewMiscWorker(stop <-chan int, wg *sync.WaitGroup, workDirPath string, outputDirPath string, inputStream <-chan common.Task, id uint) (*Worker, error) {
	if _, err := os.Stat(workDirPath); os.IsNotExist(err) {
		return nil, errors.New("work dir path not exist")
	}

	if _, err := os.Stat(outputDirPath); os.IsNotExist(err) {
		return nil, errors.New("output dir path not exist")
	}

	w := Worker{
		id:            id,
		stop:          stop,
		wg:            wg,
		workDirPath:   workDirPath,
		outputDirPath: outputDirPath,
		inputStream:   inputStream,
		isRunning:     false,
	}

	return &w, nil
}

func (w *Worker) Start() error {
	if w.isRunning == true {
		return errors.New("misc worker already running")
	}

	w.isRunning = true
	w.wg.Add(1)
	go w.workerLoop()
	log.Printf("[info] misc worker #%d started\n", w.id)

	return nil
}

func (w *Worker) workerLoop() {
	defer func() {
		w.isRunning = false
		w.wg.Done()
		log.Printf("[info] misc worker #%d exited\n", w.id)
	}()

	exitFlag := false
	for {
		if exitFlag == true {
			log.Printf("[info] misc worker #%d receive exit signal\n", w.id)
			break
		}

		select {
		case <-w.stop:
			exitFlag = true
			continue
		case task := <-w.inputStream:
			log.Printf("[info] misc worker #%d handle task: %s\n", w.id, task.Src)
			err := w.handleNewTask(&task)
			if err != nil {
				log.Printf("[error] misc worker #%d encounter error during handle task %s: %s\n", w.id, task.Src, err.Error())
				continue
			}
			log.Printf("[info] misc worker #%d finish task: %s\n", w.id, task.Src)
		}

		runtime.Gosched()
	}
}

func (w *Worker) handleNewTask(task *common.Task) error {
	srcFile := task.Src
	if _, err := os.Stat(srcFile); os.IsNotExist(err) {
		errDesc := fmt.Sprintf("src file not exist: %s", srcFile)
		status.SetStatusCode(srcFile, status.ERROR)
		status.SetStatusDesc(srcFile, errDesc)
		return errors.New(errDesc)
	}

	status.SetStatusCode(srcFile, status.MISC)
	status.SetStatusDesc(srcFile, "handling audio task")

	for _, audioTask := range task.Audio {
		codecHandler, exist := AudioCodecHandlerMap[audioTask.Codec]
		if !exist {
			errDesc := fmt.Sprintf("unknown audio codec for track %d: %s", audioTask.Track, audioTask.Codec)
			status.SetStatusCode(srcFile, status.ERROR)
			status.SetStatusDesc(srcFile, errDesc)
			return errors.New(errDesc)
		}

		status.SetStatusDesc(srcFile, fmt.Sprintf("encoding audio track #%d to %s", audioTask.Track, audioTask.Codec))

		audioPath, err := codecHandler(srcFile, w.workDirPath, &audioTask)
		if err != nil {
			status.SetStatusCode(srcFile, status.ERROR)
			status.SetStatusDesc(srcFile, err.Error())
			return err
		}

		task.AddResultPath(audioPath)

		runtime.Gosched()
	}

	status.SetStatusDesc(srcFile, "handling demux task")

	for _, demuxTask := range task.Demux {
		status.SetStatusDesc(srcFile, fmt.Sprintf("demuxing track #%d, format %s", demuxTask.Track, demuxTask.Format))

		outputPath, err := Demux(srcFile, w.workDirPath, &demuxTask)
		if err != nil {
			status.SetStatusCode(srcFile, status.ERROR)
			status.SetStatusDesc(srcFile, err.Error())
			return err
		}

		task.AddResultPath(outputPath)

		runtime.Gosched()
	}

	status.SetStatusDesc(srcFile, "moving output files to output dir")

	moveFileList := task.GetResultPathList()
	moveFileList = append(moveFileList, task.TaskFile)
	moveFileList = append(moveFileList, task.ScriptFile)
	for _, outputPath := range moveFileList {
		err := common.MoveFile(outputPath, w.outputDirPath)
		if err != nil {
			status.SetStatusCode(srcFile, status.ERROR)
			status.SetStatusDesc(srcFile, err.Error())
			return err
		}
	}

	status.SetStatusCode(srcFile, status.DONE)
	status.SetStatusDesc(srcFile, "all done")

	return nil
}
