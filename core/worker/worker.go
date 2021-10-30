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
}

type Constructor func(wg *sync.WaitGroup, param *common.Parameter, inputStream <-chan common.Task, id uint) (Worker, error)
