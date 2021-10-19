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
	"context"
	"errors"
	"fmt"
	"os/exec"
)

type AudioCodecHandler func(context.Context, string, string, *common.AudioTask) (string, error)

var AudioCodecHandlerMap = map[string]AudioCodecHandler{
	"flac": handlerFLAC,
	"opus": handlerOpus,
	"aac":  handlerAAC,
	"ac3":  generateAudioCopyHandler("ac3"),
	"dts":  generateAudioCopyHandler("dts"),
	"thd":  generateAudioCopyHandler("thd"),
}

func handlerFLAC(ctx context.Context, srcPath string, workDirPath string, audioTask *common.AudioTask) (string, error) {
	outputPath := common.GenerateNewFilePath(srcPath, workDirPath, "flac", audioTask.Language, audioTask.Track)

	track := fmt.Sprintf("%d:", audioTask.Track)
	eac3toParam := []string{srcPath, track, outputPath, "-log=NUL"}

	eac3toPath := common.GetEac3toPath()
	eac3toProcess := exec.CommandContext(ctx, eac3toPath, eac3toParam...)
	err := eac3toProcess.Run()
	if err != nil {
		return "", err
	}

	return outputPath, nil
}

func handlerOpus(ctx context.Context, srcPath string, workDirPath string, audioTask *common.AudioTask) (string, error) {
	outputPath := common.GenerateNewFilePath(srcPath, workDirPath, "opus", audioTask.Language, audioTask.Track)

	track := fmt.Sprintf("%d:", audioTask.Track)
	eac3toParam := []string{srcPath, track, "stdout.wav", "-log=NUL"}

	if audioTask.Bitrate <= 0 {
		return "", errors.New("invalid bitrate setting for opus codec")
	}
	bitrate := fmt.Sprintf("%d", audioTask.Bitrate)
	opusencParam := []string{"--ignorelength", "--vbr", "--bitrate", bitrate, "-", outputPath}

	eac3toPath := common.GetEac3toPath()
	eac3toProcess := exec.CommandContext(ctx, eac3toPath, eac3toParam...)

	opusencPath := common.GetOpusencPath()
	opusencProcess := exec.CommandContext(ctx, opusencPath, opusencParam...)
	opusencProcess.Stdin, _ = eac3toProcess.StdoutPipe()

	err := eac3toProcess.Start()
	if err != nil {
		return "", err
	}

	err = opusencProcess.Start()
	if err != nil {
		return "", err
	}

	err = opusencProcess.Wait()
	if err != nil {
		return "", err
	}

	err = eac3toProcess.Wait()
	if err != nil {
		return "", err
	}

	return outputPath, nil
}

func handlerAAC(ctx context.Context, srcPath string, workDirPath string, audioTask *common.AudioTask) (string, error) {
	outputPath := common.GenerateNewFilePath(srcPath, workDirPath, "aac", audioTask.Language, audioTask.Track)

	track := fmt.Sprintf("%d:", audioTask.Track)
	eac3toParam := []string{srcPath, track, "stdout.wav", "-log=NUL"}

	if audioTask.Bitrate <= 0 {
		return "", errors.New("invalid bitrate setting for aac codec")
	}
	bitrate := fmt.Sprintf("%d", audioTask.Bitrate)
	qaacParam := []string{"-R", "--adts", "-v", bitrate, "-o", outputPath, "-"}

	eac3toPath := common.GetEac3toPath()
	eac3toProcess := exec.CommandContext(ctx, eac3toPath, eac3toParam...)

	qaacPath := common.GetQaacPath()
	qaacProcess := exec.CommandContext(ctx, qaacPath, qaacParam...)
	qaacProcess.Stdin, _ = eac3toProcess.StdoutPipe()

	err := eac3toProcess.Start()
	if err != nil {
		return "", nil
	}

	err = qaacProcess.Start()
	if err != nil {
		return "", nil
	}

	err = qaacProcess.Wait()
	if err != nil {
		return "", nil
	}

	err = eac3toProcess.Wait()
	if err != nil {
		return "", nil
	}

	return outputPath, nil
}

func generateAudioCopyHandler(ext string) AudioCodecHandler {
	return func(ctx context.Context, srcPath string, workDirPath string, audioTask *common.AudioTask) (string, error) {
		outputPath := common.GenerateNewFilePath(srcPath, workDirPath, ext, audioTask.Language, audioTask.Track)

		track := fmt.Sprintf("%d:", audioTask.Track)
		eac3toParam := []string{srcPath, track, outputPath, "-log=NUL"}

		eac3toPath := common.GetEac3toPath()
		eac3toProcess := exec.CommandContext(ctx, eac3toPath, eac3toParam...)
		err := eac3toProcess.Run()
		if err != nil {
			return "", err
		}

		return outputPath, nil
	}
}

func Demux(ctx context.Context, srcFile string, workDirPath string, demuxTask *common.DemuxTask) (string, error) {
	outputPath := common.GenerateNewFilePath(srcFile, workDirPath, demuxTask.Format, demuxTask.Language, demuxTask.Track)

	track := fmt.Sprintf("%d:", demuxTask.Track)
	eac3toParam := []string{srcFile, track, outputPath, "-log=NUL"}

	eac3toPath := common.GetEac3toPath()
	eac3toProcess := exec.CommandContext(ctx, eac3toPath, eac3toParam...)
	err := eac3toProcess.Start()
	if err != nil {
		return "", err
	}

	err = eac3toProcess.Wait()
	if err != nil {
		return "", err
	}

	return outputPath, nil
}
