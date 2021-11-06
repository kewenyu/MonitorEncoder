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

package activetime

import (
	"errors"
	"log"
	"regexp"
	"strconv"
	"time"
)

type ActiveTime [6]int

var (
	activeTimeStr string
	activeTime    ActiveTime
	continueChan  = make(chan struct{})
)

func init() {
}

func SetActiveTime(activeTimeStr string) error {
	err := parseActiveTimeStr(activeTimeStr)
	if err != nil {
		return err
	}

	go continueApproveLoop()

	return nil
}

func IsContinue() chan<- struct{} {
	return continueChan
}

func parseActiveTimeStr(s string) error {
	regMatcher := regexp.MustCompile(`(\d+):(\d+):(\d+)-(\d+):(\d+):(\d+)`)
	regMatchResult := regMatcher.FindStringSubmatch(s)
	if len(regMatchResult) != 7 {
		return errors.New("invalid active time format")
	}

	for i := 1; i <= 6; i++ {
		value, err := strconv.ParseInt(regMatchResult[i], 10, 32)
		if err != nil {
			return err
		}
		activeTime[i-1] = int(value)
	}

	for i, v := range activeTime {
		if (i == 0 || i == 3) && v >= 24 {
			return errors.New("invalid active time value")
		}

		if v < 0 || v >= 60 {
			return errors.New("invalid active time value")
		}
	}

	activeTimeStr = s

	return nil
}

func continueApproveLoop() {
	if !isActiveTimeDisable() {
		log.Printf("[info] active time setting: %s\n", activeTimeStr)
		for {
			if !isInActiveTime() {
				remainTimeToBegin := getTimeToBegin()
				log.Printf("[info] not in active time. remaining %.2f seconds to begin\n", remainTimeToBegin.Seconds())
				timeToBeginOut := time.After(remainTimeToBegin)
				select {
				case <-timeToBeginOut:
					log.Printf("[info] active time begin\n")
					break
				}
			}

			remainTimeToEnd := getTimeToEnd()
			log.Printf("[info] in active time. remaining %.2f seconds to end\n", remainTimeToEnd.Seconds())
			timeToEndOut := time.After(remainTimeToEnd)

		approveLoop:
			for {
				select {
				case <-continueChan:
				case <-timeToEndOut:
					log.Printf("[info] active time is up\n")
					break approveLoop
				}
			}
		}
	} else {
		log.Printf("[info] active time is disable\n")
		for {
			select {
			case <-continueChan:
			}
		}
	}
}

func isActiveTimeDisable() bool {
	currentTime := time.Now()
	beginTime := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), activeTime[0], activeTime[1], activeTime[2], 0, time.Local)
	endTime := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), activeTime[3], activeTime[4], activeTime[5], 0, time.Local)

	return beginTime.Equal(endTime)
}

func isInActiveTime() bool {
	currentTime := time.Now()
	beginTime := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), activeTime[0], activeTime[1], activeTime[2], 0, time.Local)
	endTime := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), activeTime[3], activeTime[4], activeTime[5], 0, time.Local)
	if endTime.Before(beginTime) {
		endTime = endTime.Add(24 * time.Hour)
	}

	return currentTime.After(beginTime) && currentTime.Before(endTime)
}

func getTimeToBegin() time.Duration {
	currentTime := time.Now()
	beginTime := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), activeTime[0], activeTime[1], activeTime[2], 0, time.Local)
	if beginTime.Before(currentTime) {
		beginTime = beginTime.Add(24 * time.Hour)
	}

	return beginTime.Sub(currentTime)
}

func getTimeToEnd() time.Duration {
	currentTime := time.Now()
	endTime := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), activeTime[3], activeTime[4], activeTime[5], 0, time.Local)
	if endTime.Before(currentTime) {
		endTime = endTime.Add(24 * time.Hour)
	}

	return endTime.Sub(currentTime)
}
