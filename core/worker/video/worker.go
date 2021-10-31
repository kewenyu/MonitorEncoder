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
	"MonitorEncoder/core/worker"
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
	*worker.Base
	workDirPath  string
}

func NewVideoWorker(wg *sync.WaitGroup, param *common.Parameter, id uint) *Worker {
	w := Worker{
		Base:        worker.NewWorkerBase(wg, id),
		workDirPath: param.WorkDirPath,
	}

	return &w
}

func (w *Worker) Start(ctx context.Context) error {
	if w.IsRunning == true {
		return errors.New("video worker already running")
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
	return fmt.Sprintf("video worker #%d", w.Id)
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
				log.Printf("[error] %s encounter error during handle task %s: %s", w.GetPrettyName(), task.Src, err.Error())
				continue
			}
			log.Printf("[info] %s finish task: %s\n", w.GetPrettyName(), task.Src)

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

	task.AddResult(common.NewResult(resultPath, common.ResultVideo, "", 0))

	return nil
}

func indexTask(ctx context.Context, scriptPath string, task *common.Task) error {
	vspipePath := common.GetVspipePath()
	vspipeProcess := exec.CommandContext(ctx, vspipePath, "-i", scriptPath, "-")
	data, err := vspipeProcess.Output()
	if err != nil {
		return err
	}

	frameNumRegExp := regexp.MustCompile(`Frames:\s*(\d+)`)
	frameNumMatch := frameNumRegExp.FindStringSubmatch(string(data))
	if len(frameNumMatch) == 2 {
		totalFrameNum, err := strconv.ParseUint(frameNumMatch[1], 10, 32)
		if err != nil {
			return err
		} else {
			task.TotalFrameNum = uint(totalFrameNum)
		}
	}

	fpsRegExp := regexp.MustCompile(`FPS:\s*(\d+)/(\d+)`)
	fpsMatch := fpsRegExp.FindStringSubmatch(string(data))
	if len(fpsMatch) == 3 {
		fpsNum, err := strconv.ParseUint(fpsMatch[1], 10, 32)
		if err != nil {
			return err
		} else {
			task.FPSNum = uint(fpsNum)
		}

		fpsDen, err := strconv.ParseUint(fpsMatch[2], 10, 32)
		if err != nil {
			return err
		} else {
			task.FPSDen = uint(fpsDen)
		}
	}

	return nil
}
