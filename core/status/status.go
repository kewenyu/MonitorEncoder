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

package status

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"
)

type Code int

const (
	ERROR Code = iota
	WAIT
	VIDEO
	MISC
	MUX
	FINAL
	DONE
)

type Status struct {
	Id      uint64
	SrcFile string
	Code    Code
	Desc    string
}

var (
	statusMap  = make(map[string]*Status)
	taskIdLock sync.Mutex
	taskId     uint64 = 0
)

func NewStatus(srcFile string) *Status {
	taskIdLock.Lock()
	defer taskIdLock.Unlock()

	taskId += 1

	return &Status{
		Id:      taskId,
		SrcFile: srcFile,
		Code:    0,
		Desc:    "",
	}
}

func SetStatusCode(srcFile string, code Code) {
	_, exist := statusMap[srcFile]
	if !exist {
		statusMap[srcFile] = NewStatus(srcFile)
	}
	statusMap[srcFile].Code = code
}

func SetStatusDesc(srcFile string, desc string) {
	_, exist := statusMap[srcFile]
	if !exist {
		statusMap[srcFile] = NewStatus(srcFile)
	}
	statusMap[srcFile].Desc = desc
}

func PrintAllStatus() {
	fmt.Printf("----------------- Status ----------------\n")
	for srcFile, status := range statusMap {
		fmt.Printf("%s:\t\t%s\n", srcFile, status.Desc)
	}
	fmt.Printf("-----------------------------------------\n")
}

func GetAllStatus() []Status {
	statusList := make([]Status, 0)

	for _, status := range statusMap {
		statusList = append(statusList, *status)
	}

	sort.Slice(statusList, func (i int, j int) bool{
		return statusList[i].Id < statusList[j].Id
	})

	return statusList
}

func GetAllStatusJson() []byte {
	data, err := json.Marshal(statusMap)
	if err != nil {
		return nil
	}

	return data
}
