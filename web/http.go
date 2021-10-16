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

package web

import (
	"MonitorEncoder/core/common"
	"MonitorEncoder/core/status"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

var (
	addr        string
	isRunning   bool
	monitorPath string
)

func StartWeb(ip string, port string, monitorDir string) error {
	if _, err := os.Stat(monitorDir); os.IsNotExist(err) {
		return errors.New("monitor dir path not exist")
	}

	monitorPath = monitorDir

	if isRunning == true {
		return errors.New("http interface already running")
	}

	addr = fmt.Sprintf("%s:%s", ip, port)

	go func() {
		defer func() {
			isRunning = false
			log.Printf("[info] http interface exited\n")
		}()

		http.HandleFunc("/status", pageStatus)
		http.HandleFunc("/api/status", apiStatus)
		http.HandleFunc("/api/newtask", apiNewTask)

		err := http.ListenAndServe(addr, nil)
		if err != nil {
			log.Printf("[error] http interface failed to listen at %s\n", addr)
		}
	}()

	isRunning = true
	log.Printf("[info] http interface listening at %s:%s\n", ip, port)

	return nil
}

func pageStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		dataList := status.GetAllStatus()
		if len(dataList) <= 0 {
			_, _ = w.Write([]byte("There is no task in the list."))
			return
		}

		output := "<h1>Monitor Encoder Status List</h1>\n"
		output += "<table border=\"1\">\n"
		output += "<tr>\n"
		output += "<th>Id</th>\n"
		output += "<th>Source File</th>\n"
		output += "<th>Status Code</th>\n"
		output += "<th>Detail</th>\n"
		output += "</tr>\n"

		for _, data := range dataList {
			output += "<tr>\n"
			output += fmt.Sprintf("<th>%d</th>\n", data.Id)
			output += fmt.Sprintf("<th>%s</th>\n", data.SrcFile)
			output += fmt.Sprintf("<th>%d</th>\n", data.Code)
			output += fmt.Sprintf("<th>%s</th>\n", data.Desc)
			output += "</tr>\n"
		}

		output += "</table>\n"

		_, _ = w.Write([]byte(output))
	}
}

func apiStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		w.Header().Set("Content-Type", "application/json")
		data := status.GetAllStatusJson()
		if data != nil {
			_, _ = w.Write(data)
		} else {
			_, _ = w.Write([]byte("error: nil json from status"))
		}
	}
}

func apiNewTask(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		var task common.Task
		err = json.Unmarshal(body, &task)
		if err != nil {
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		scriptFilePath := common.GenerateNewFilePath(task.Src, monitorPath, "json", "", 0)
		jsonFile, err := os.OpenFile(scriptFilePath, os.O_RDWR|os.O_CREATE, 0777)
		if err != nil {
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		defer func() {
			closeErr := jsonFile.Close()
			if err == nil {
				err = closeErr
			}
		}()

		_, err = jsonFile.Write(body)
		if err != nil {
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		succMsg := fmt.Sprintf("new task added via http REST api: %s", task.Src)
		_, _ = w.Write([]byte(succMsg))
		log.Printf("[info] %s\n", succMsg)
	}
}
