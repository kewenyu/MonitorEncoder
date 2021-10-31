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
	"context"
	"errors"
	"fmt"
	"sync"
)

type MultiWorker struct {
	workerList   []*Worker
	outputStream chan common.Task
	isRunning    bool
}

func NewMultiWorker(wg *sync.WaitGroup, param *common.Parameter, id uint) *MultiWorker {
	mw := MultiWorker{
		workerList:   make([]*Worker, 0),
		outputStream: make(chan common.Task),
		isRunning:    false,
	}

	for i := 0; i < param.WorkerNum; i++ {
		worker := NewVideoWorker(wg, param, id + uint(i))
		mw.workerList = append(mw.workerList, worker)
	}

	return &mw
}

func (mw *MultiWorker) Start(ctx context.Context) error {
	if mw.isRunning == true {
		return errors.New("video worker already running")
	}

	if len(mw.workerList) <= 0 {
		return errors.New("worker list is empty")
	}

	for _, worker := range mw.workerList {
		err := worker.Start(ctx)
		if err != nil {
			return err
		}
	}

	for _, worker := range mw.workerList {
		go func(ctx context.Context, inputChan chan common.Task) {
			for {
				select {
				case <-ctx.Done():
					return
				case task := <-inputChan:
					mw.outputStream <- task
					continue
				}
			}
		}(ctx, worker.GetOutputStream())
	}

	mw.isRunning = true

	return nil
}

func (mw *MultiWorker) SetInputStream(inputStream <-chan common.Task) {
	for _, worker := range mw.workerList {
		worker.SetInputStream(inputStream)
	}
}

func (mw *MultiWorker) GetOutputStream() chan common.Task {
	return mw.outputStream
}

func (mw *MultiWorker) GetPrettyName() string {
	return fmt.Sprintf("multi video worker wrapper")
}
