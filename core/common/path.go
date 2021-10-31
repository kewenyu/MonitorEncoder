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
	"errors"
	"os"
	"path/filepath"
)

var binPathMap = map[string]string{
	"eac3toPath":   "eac3to\\eac3to.exe",
	"vspipePath":   "vspipe.exe",
	"x265Path":     "x265-10b.exe",
	"x264Path":     "x264_64.exe",
	"opusencPath":  "opusenc.exe",
	"qaacPath":     "qaac.exe",
	"mkvmergePath": "mkvtoolnix\\mkvmerge.exe",
	"lsmashPath":   "lsmashmuxer.exe",
}

func init() {
	binPathBase := os.Getenv("MONITOR_ENCODER_BIN_PATH")
	if binPathBase != "" {
		for k, v := range binPathMap {
			binPathMap[k] = filepath.Join(binPathBase, v)
		}
	}
}

func CheckToolsAvailability() error {
	for _, path := range binPathMap {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return errors.New(path + " not exist")
		}
	}
	return nil
}

func GetEac3toPath() string {
	return binPathMap["eac3toPath"]
}

func GetX265Path() string {
	return binPathMap["x265Path"]
}

func GetX264Path() string {
	return binPathMap["x264Path"]
}

func GetOpusencPath() string {
	return binPathMap["opusencPath"]
}

func GetVspipePath() string {
	return binPathMap["vspipePath"]
}

func GetQaacPath() string {
	return binPathMap["qaacPath"]
}

func GetMkvmergePath() string {
	return binPathMap["mkvmergePath"]
}

func GetLsmashPath() string {
	return binPathMap["lsmashPath"]
}
