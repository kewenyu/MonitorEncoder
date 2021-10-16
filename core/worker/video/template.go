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
	"bufio"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var substitutionMap = map[string]func(string, *common.Task) (string, error){
	"###INPUTFILE###": substitutionInput,
	"###DEBUG###":     substitutionDebug,
	"###SUBTITLE###":  substitutionSubtitle,
}

func GenerateVpyFile(workDirPath string, task *common.Task) (string, error) {
	if _, err := os.Stat(workDirPath); os.IsNotExist(err) {
		return "", errors.New("work dir path not exist")
	}

	templatePath := task.Template
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		return "", errors.New("template path not exist")
	}

	templateFile, err := os.Open(templatePath)
	if err != nil {
		return "", errors.New("failed to open template file: " + err.Error())
	}
	defer func() {
		closeErr := templateFile.Close()
		if err == nil {
			err = closeErr
		}
	}()

	vpyFilePath := common.GenerateNewFilePath(task.Src, workDirPath, "vpy", "", 0)
	vpyFile, err2 := os.Create(vpyFilePath)
	if err2 != nil {
		return "", errors.New("failed to create vpy file: " + err.Error())
	}
	defer func() {
		closeErr := vpyFile.Close()
		if err == nil {
			err = closeErr
		}
	}()

	substitution := substitutionCopy
	templateScanner := bufio.NewScanner(templateFile)
	for templateScanner.Scan() {
		line := templateScanner.Text()

		if _, exist := substitutionMap[strings.TrimSpace(line)]; exist {
			substitution = substitutionMap[strings.TrimSpace(line)]
			continue
		}

		newLine, serr := substitution(line, task)
		if serr != nil {
			return "", errors.New(fmt.Sprintf("error during template substitution: %s", serr.Error()))
		}

		_, werr := vpyFile.WriteString(newLine)
		if werr != nil {
			return "", errors.New(fmt.Sprintf("error during writing vpy script %s: %s", vpyFilePath, werr.Error()))
		}

		substitution = substitutionCopy
	}

	return vpyFilePath, nil
}

func substitutionInput(line string, task *common.Task) (string, error) {
	srcVarReg := regexp.MustCompile(`(\w+)\s*=\s*\S+`)
	srcVarMatch := srcVarReg.FindStringSubmatch(line)
	if len(srcVarMatch) != 2 {
		return "", errors.New("failed to match template's src variable")
	}

	srcVar := srcVarMatch[1]
	newLine := fmt.Sprintf("%s = r\"%s\"\n", srcVar, task.Src)

	return newLine, nil
}

func substitutionDebug(line string, _ *common.Task) (string, error) {
	debugVarReg := regexp.MustCompile(`(\w+)\s*=\s*\S+`)
	debugVarMatch := debugVarReg.FindStringSubmatch(line)
	if len(debugVarMatch) != 2 {
		return "", errors.New("failed to match template's debug variable")
	}

	debugVar := debugVarMatch[1]
	newLine := fmt.Sprintf("%s = False\n", debugVar)

	return newLine, nil
}

func substitutionSubtitle(line string, task *common.Task) (string, error) {
	subVarReg := regexp.MustCompile(`(\w+)\s*=\s*\S+`)
	subVarMatch := subVarReg.FindStringSubmatch(line)
	if len(subVarMatch) != 2 {
		return "", errors.New("failed to match template's subtitle variable")
	}

	subVar := subVarMatch[1]
	newLine := fmt.Sprintf("%s = r\"%s\"\n", subVar, task.HardSub)

	return newLine, nil
}

func substitutionCopy(line string, _ *common.Task) (string, error) {
	return fmt.Sprintf("%s\n", line), nil
}
