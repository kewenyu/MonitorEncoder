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

package video

import (
	"MonitorEncoder/core/common"
	"MonitorEncoder/core/status"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
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

func NewVideoWorker(wg *sync.WaitGroup, workDirPath string, inputStream <-chan common.Task, id uint) (*Worker, error) {
	if _, err := os.Stat(workDirPath); os.IsNotExist(err) {
		return nil, errors.New("work dir path not exist")
	}

	w := Worker{
		id:           id,
		wg:           wg,
		workDirPath:  workDirPath,
		inputStream:  inputStream,
		outputStream: make(chan common.Task),
		isRunning:    false,
	}

	return &w, nil
}

func (w *Worker) Start(ctx context.Context) error {
	if w.isRunning == true {
		return errors.New("video worker already running")
	}

	w.isRunning = true
	w.wg.Add(1)
	go w.workerLoop(ctx)
	log.Printf("[info] video worker #%d started\n", w.id)

	return nil
}

func (w *Worker) GetOutputStream() chan common.Task {
	return w.outputStream
}

func (w *Worker) workerLoop(ctx context.Context) {
	defer func() {
		w.isRunning = false
		w.wg.Done()
		log.Printf("[info] video worker #%d exited\n", w.id)
	}()

	exitFlag := false
	for {
		if exitFlag == true {
			log.Printf("[info] video worker #%d receive exit signal\n", w.id)
			break
		}

		select {
		case <-ctx.Done():
			exitFlag = true
			continue
		case task := <-w.inputStream:
			log.Printf("[info] video worker #%d handle task: %s\n", w.id, task.Src)
			err := w.handleNewTask(ctx, &task)
			if err != nil {
				log.Printf("[error] video worker #%d encounter error during handle task %s: %s", w.id, task.Src, err.Error())
				continue
			}
			log.Printf("[info] video worker #%d finish task: %s\n", w.id, task.Src)

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
	if _, err := os.Stat(srcFile); os.IsNotExist(err) {
		errDesc := fmt.Sprintf("src file not exist: %s", srcFile)
		status.SetStatusCode(srcFile, status.ERROR)
		status.SetStatusDesc(srcFile, errDesc)
		return errors.New(errDesc)
	}

	codec := task.Video
	codecHandler, exist := CodecHandlerMap[codec]
	if !exist {
		errDesc := fmt.Sprintf("unknown video codec: %s", codec)
		status.SetStatusCode(srcFile, status.ERROR)
		status.SetStatusDesc(srcFile, errDesc)
		return errors.New(errDesc)
	}

	scriptPath, err := GenerateVpyFile(w.workDirPath, task)
	if err != nil {
		status.SetStatusCode(srcFile, status.ERROR)
		status.SetStatusDesc(srcFile, err.Error())
		return err
	}

	task.ScriptFile = scriptPath

	status.SetStatusCode(srcFile, status.VIDEO)
	status.SetStatusDesc(srcFile, "indexing")

	err = indexTask(ctx, scriptPath, task)
	if err != nil {
		status.SetStatusCode(srcFile, status.ERROR)
		status.SetStatusDesc(srcFile, "indexing failed: "+err.Error())
		return err
	}

	resultPath, err := codecHandler(ctx, scriptPath, w.workDirPath, task)
	if err != nil {
		status.SetStatusCode(srcFile, status.ERROR)
		status.SetStatusDesc(srcFile, err.Error())
		return err
	}

	task.AddResultPath(resultPath)

	return nil
}

func indexTask(ctx context.Context, scriptPath string, task *common.Task) error {
	vspipePath := common.GetVspipePath()
	vspipeProcess := exec.CommandContext(ctx, vspipePath, "-i", scriptPath, "-")
	data, err := vspipeProcess.Output()
	if err != nil {
		return err
	}

	infoRegExp := regexp.MustCompile(`Frames:\s*(\d+)`)
	match := infoRegExp.FindStringSubmatch(string(data))
	if len(match) == 2 {
		totalFrameNum, err := strconv.ParseUint(match[1], 10, 32)
		if err != nil {
			return err
		} else {
			task.TotalFrameNum = uint(totalFrameNum)
		}
	}

	return nil
}
