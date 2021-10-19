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
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

type CodecHandler func(context.Context, string, string, *common.Task) (string, error)

var CodecHandlerMap = map[string]CodecHandler{
	"hevc": handlerHEVC,
	"avc":  handlerAVC,
}

func handlerHEVC(ctx context.Context, scriptPath string, workDirPath string, task *common.Task) (string, error) {
	hevcFilePath := common.GenerateNewFilePath(task.Src, workDirPath, "hevc", "", 0)

	baseX265Param := []string{"-D", "10", "--y4m", "--output", hevcFilePath, "-"}
	extraX265Param := strings.Split(task.Param, " ")
	x265Param := append(baseX265Param, extraX265Param...)

	vspipePath := common.GetVspipePath()
	vspipeProcess := exec.CommandContext(ctx, vspipePath, "-y", scriptPath, "-")

	x265Path := common.GetX265Path()
	x265Process := exec.CommandContext(ctx, x265Path, x265Param...)
	x265Process.Stdin, _ = vspipeProcess.StdoutPipe()
	x265StdErr, _ := x265Process.StderrPipe()

	err := vspipeProcess.Start()
	if err != nil {
		return "", err
	}

	err = x265Process.Start()
	if err != nil {
		return "", err
	}

	exitFlag := false
	finishFlag := false
	x265Reader := bufio.NewReader(x265StdErr)
	x265ProgressRegexp := regexp.MustCompile(`(\d+) frames:.*`)
	x265FinishRegexp := regexp.MustCompile(`encoded \d+ frames`)
	for {
		if exitFlag == true {
			break
		}

		select {
		case <-ctx.Done():
			exitFlag = true
			continue
		default:
		}

		errLine, rErr := x265Reader.ReadString('\r')

		progress := x265ProgressRegexp.FindStringSubmatch(errLine)
		if len(progress) >= 2 {
			status.SetStatusDesc(task.Src, fmt.Sprintf("hevc encoding frame: %s/%d", progress[1], task.TotalFrameNum))
		}

		finish := x265FinishRegexp.FindString(errLine)
		if finish != "" {
			status.SetStatusDesc(task.Src, "hevc encoding done")
			finishFlag = true
		}

		if rErr != nil || rErr == io.EOF {
			break
		}

		runtime.Gosched()
	}

	if finishFlag != true {
		return "", errors.New("x265 encoding didn't finish")
	}

	return hevcFilePath, nil
}

func handlerAVC(ctx context.Context, scriptPath string, workDirPath string, task *common.Task) (string, error) {
	avcFilePath := common.GenerateNewFilePath(task.Src, workDirPath, "264", "", 0)

	baseX264Param := []string{"--demuxer", "y4m", "--output", avcFilePath, "-"}
	extraX264Param := strings.Split(task.Param, " ")
	x264Param := append(baseX264Param, extraX264Param...)

	vspipePath := common.GetVspipePath()
	vspipeProcess := exec.CommandContext(ctx, vspipePath, "-y", scriptPath, "-")

	x264Path := common.GetX264Path()
	x264Process := exec.CommandContext(ctx, x264Path, x264Param...)
	x264Process.Stdin, _ = vspipeProcess.StdoutPipe()
	x264StdErr, _ := x264Process.StderrPipe()

	err := vspipeProcess.Start()
	if err != nil {
		return "", err
	}

	err = x264Process.Start()
	if err != nil {
		return "", err
	}

	exitFlag := false
	finishFlag := false
	x264Reader := bufio.NewReader(x264StdErr)
	x264ProgressRegexp := regexp.MustCompile(`(\d+) frames:.*`)
	x264FinishRegexp := regexp.MustCompile(`encoded \d+ frames`)
	for {
		if exitFlag == true {
			break
		}

		select {
		case <-ctx.Done():
			exitFlag = true
			continue
		default:
		}

		errLine, rErr := x264Reader.ReadString('\r')

		progress := x264ProgressRegexp.FindStringSubmatch(errLine)
		if len(progress) >= 2 {
			status.SetStatusDesc(task.Src, fmt.Sprintf("avc encoding frame: %s/%d", progress[1], task.TotalFrameNum))
		}

		finish := x264FinishRegexp.FindString(errLine)
		if finish != "" {
			status.SetStatusDesc(task.Src, "avc encoding done")
			finishFlag = true
		}

		if rErr != nil || rErr == io.EOF {
			break
		}

		runtime.Gosched()
	}

	if finishFlag != true {
		return "", errors.New("x264 encoding didn't finish")
	}

	return avcFilePath, nil
}
