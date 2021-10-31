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

package mux

import (
	"MonitorEncoder/core/common"
	"context"
	"fmt"
	"os/exec"
)

type FormatHandler func(context.Context, string, *common.Task) (string, error)

var formatHandlerMap = map[string]FormatHandler{
	"mkv": handlerMkv,
	"mp4": handlerMp4,
}

func handlerMkv(ctx context.Context, workDirPath string, task *common.Task) (string, error) {
	mkvFilePath := common.GenerateNewFilePath(task.Src, workDirPath, "mkv", "", 0)

	mkvmergeParam := []string{"-o", mkvFilePath}
	resultList := task.GetResultList()
	for _, result := range resultList {
		if result.Lang != "" {
			mkvmergeParam = append(mkvmergeParam, "--language")
			mkvmergeParam = append(mkvmergeParam, fmt.Sprintf("0:%s", result.Lang))
		}
		mkvmergeParam = append(mkvmergeParam, result.Path)
	}

	mkvmergePath := common.GetMkvmergePath()
	mkvmergeProcess := exec.CommandContext(ctx, mkvmergePath, mkvmergeParam...)
	err := mkvmergeProcess.Run()
	if err != nil {
		return "", err
	}

	return mkvFilePath, nil
}

func handlerMp4(ctx context.Context, workDirPath string, task *common.Task) (string, error) {
	mp4FilePath := common.GenerateNewFilePath(task.Src, workDirPath, "mp4", "", 0)

	lsmashParam := []string{"-o", mp4FilePath}
	resultList := task.GetResultList()
	for _, result := range resultList {
		if result.Category == common.ResultVideo {
			lsmashParam = append(lsmashParam, "-i")
			lsmashParam = append(lsmashParam, fmt.Sprintf("%s?fps=%d/%d", result.Path, task.FPSNum, task.FPSDen))
		} else {
			lsmashParam = append(lsmashParam, "-i")
			trackOpts := result.Path
			if result.Lang != "" {
				trackOpts += fmt.Sprintf("?language=%s", result.Lang)
			}
			lsmashParam = append(lsmashParam, trackOpts)
		}
	}

	lsmashPath := common.GetLsmashPath()
	lsmashProcess := exec.CommandContext(ctx, lsmashPath, lsmashParam...)
	err := lsmashProcess.Run()
	if err != nil {
		return "", err
	}

	return mp4FilePath, nil
}
