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

package common

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
)

type Task struct {
	Src      string      `json:"src"`
	Template string      `json:"template"`
	Param    string      `json:"param"`
	Video    string      `json:"video"`
	Audio    []AudioTask `json:"audio"`
	Demux    []DemuxTask `json:"demux"`
	HardSub  string      `json:"hardsub"`

	TotalFrameNum  uint
	FPSNum         uint
	FPSDen         uint
	ScriptFile     string
	TaskFile       string
	resultPathList []string
}

type AudioTask struct {
	Track    uint   `json:"track"`
	Codec    string `json:"codec"`
	Bitrate  uint   `json:"bitrate"`
	Language string `json:"language"`
}

type DemuxTask struct {
	Track    uint   `json:"track"`
	Format   string `json:"format"`
	Language string `json:"language"`
}

func NewTaskFromJson(jsonPath string) (*Task, error) {
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		return nil, errors.New("json path not exist")
	}

	jsonFile, err := os.Open(jsonPath)
	if err != nil {
		return nil, errors.New("failed to open json file: " + err.Error())
	}
	defer func() {
		closeErr := jsonFile.Close()
		if err == nil {
			err = closeErr
		}
	}()

	jsonStr := ""
	jsonScanner := bufio.NewScanner(jsonFile)
	for jsonScanner.Scan() {
		line := jsonScanner.Text()
		jsonStr += line
	}

	if jsonScanner.Err() != nil {
		return nil, errors.New("failed to read whole json file: " + jsonScanner.Err().Error())
	}

	var task Task
	err = json.Unmarshal([]byte(jsonStr), &task)
	if err != nil {
		return nil, errors.New("failed to unmarshal json task: " + err.Error())
	}

	task.resultPathList = make([]string, 0)

	return &task, nil
}

func (t Task) GetResultPathList() []string {
	return t.resultPathList
}

func (t *Task) AddResultPath(path string) {
	t.resultPathList = append(t.resultPathList, path)
}
