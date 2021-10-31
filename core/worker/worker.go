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

package worker

import (
	"MonitorEncoder/core/common"
	"context"
	"sync"
)

type Worker interface {
	Start(ctx context.Context) error
	GetOutputStream() chan common.Task
	SetInputStream(inputStream <-chan common.Task)
	GetPrettyName() string
}

type Base struct {
	Id uint
	Wg *sync.WaitGroup
	InputStream  <-chan common.Task
	OutputStream chan common.Task
	IsRunning    bool
}

func NewWorkerBase(wg *sync.WaitGroup, id uint) *Base {
	b := Base{
		Id:           id,
		Wg:           wg,
		InputStream:  nil,
		OutputStream: make(chan common.Task),
		IsRunning:    false,
	}
	return &b
}

func (b *Base) GetOutputStream() chan common.Task {
	return b.OutputStream
}

func (b *Base) SetInputStream(inputStream <-chan common.Task) {
	b.InputStream = inputStream
}
