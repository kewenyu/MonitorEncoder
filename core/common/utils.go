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
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

func MoveFile(ctx context.Context, srcPath string, dstPath string) error {
	err := exec.CommandContext(ctx, "cmd", "/C", "move", srcPath, dstPath).Run()
	if err != nil {
		return err
	}
	return nil
}

func GenerateNewFilePath(srcPath string, targetDirPath string, ext string, lang string, track uint) string {
	newFileName := strings.Replace(srcPath, "\\", "_", -1)
	newFileName = strings.Replace(newFileName, ":", "_", -1)

	if track > 0 {
		newFileName = newFileName + "." + fmt.Sprintf("track%d", track)
	}

	if lang != "" {
		newFileName = newFileName + "." + lang
	}

	newFileName = newFileName + "." + ext
	newFilePath := filepath.Join(targetDirPath, newFileName)
	return newFilePath
}
