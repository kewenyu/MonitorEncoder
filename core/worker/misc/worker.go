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

func NewMiscWorker(wg *sync.WaitGroup, param *common.Parameter, id uint) *Worker {
	w := Worker{
		Base:          worker.NewWorkerBase(wg, id),
		workDirPath:   param.WorkDirPath,
		outputDirPath: param.OutputDirPath,
	}

	return &w
}

func (w *Worker) Start(ctx context.Context) error {
	if w.IsRunning == true {
		return errors.New("misc worker already running")
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
	return fmt.Sprintf("misc worker #%d", w.Id)
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

		audioPath, err := codecHandler(ctx, srcFile, w.workDirPath, &audioTask)
		if err != nil {
			status.SetStatusCode(srcFile, status.ERROR)
			status.SetStatusDesc(srcFile, err.Error())
			return err
		}

		task.AddResult(common.NewResult(audioPath, common.ResultNonVideo, audioTask.Language, audioTask.Track))

		runtime.Gosched()
	}

	status.SetStatusDesc(srcFile, "handling demux task")

	for _, demuxTask := range task.Demux {
		status.SetStatusDesc(srcFile, fmt.Sprintf("demuxing track #%d, format %s", demuxTask.Track, demuxTask.Format))

		outputPath, err := Demux(ctx, srcFile, w.workDirPath, &demuxTask)
		if err != nil {
			status.SetStatusCode(srcFile, status.ERROR)
			status.SetStatusDesc(srcFile, err.Error())
			return err
		}

		task.AddResult(common.NewResult(outputPath, common.ResultNonVideo, demuxTask.Language, demuxTask.Track))

		runtime.Gosched()
	}

	return nil
}
