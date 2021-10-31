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
	Mux      string      `json:"mux"`

	TotalFrameNum uint
	FPSNum        uint
	FPSDen        uint
	ScriptFile    string
	TaskFile      string
	MuxedFile     string
	resultList    []Result
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

type ResultCategory int

const (
	ResultVideo ResultCategory = iota
	ResultNonVideo
)

type Result struct {
	Category ResultCategory
	Path     string
	Lang     string
	Track    uint
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

	task.resultList = make([]Result, 0)

	return &task, nil
}

func (t Task) GetResultList() []Result {
	return t.resultList
}

func (t *Task) AddResult(result Result) {
	t.resultList = append(t.resultList, result)
}

func NewResult(path string, category ResultCategory, lang string, track uint) Result {
	r := Result{
		Category: category,
		Path:     path,
		Lang:     lang,
		Track:    track,
	}
	return r
}
