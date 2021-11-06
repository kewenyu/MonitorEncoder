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
	activeTimeStr = "00:00:00-00:00:00"
	activeTime    ActiveTime
	continueChan  = make(chan struct{})
	resetChan     = make(chan struct{})
)

func init() {
	activeTime, _ = parseActiveTimeStr(activeTimeStr)
	go continueApproveLoop(resetChan)
}

func SetActiveTime(s string) error {
	newActiveTime, err := parseActiveTimeStr(s)
	if err != nil {
		return err
	}

	activeTime = newActiveTime
	activeTimeStr = s

	select {
	case resetChan <- struct{}{}:
	}

	return nil
}

func IsContinue() chan<- struct{} {
	return continueChan
}

func parseActiveTimeStr(s string) (ActiveTime, error) {
	regMatcher := regexp.MustCompile(`(\d+):(\d+):(\d+)-(\d+):(\d+):(\d+)`)
	regMatchResult := regMatcher.FindStringSubmatch(s)
	if len(regMatchResult) != 7 {
		return ActiveTime{}, errors.New("invalid active time format")
	}

	newActiveTime := [6]int{}

	for i := 1; i <= 6; i++ {
		value, err := strconv.ParseInt(regMatchResult[i], 10, 32)
		if err != nil {
			return ActiveTime{}, err
		}
		newActiveTime[i-1] = int(value)
	}

	for i, v := range newActiveTime {
		if (i == 0 || i == 3) && v >= 24 {
			return ActiveTime{}, errors.New("invalid active time value")
		}

		if v < 0 || v >= 60 {
			return ActiveTime{}, errors.New("invalid active time value")
		}
	}

	return newActiveTime, nil
}

func continueApproveLoop(resetChan <-chan struct{}) {
	for {
		if !isActiveTimeDisable() {
			log.Printf("[info] active time setting: %s\n", activeTimeStr)
			outerExitFlag := false
			for {
				if outerExitFlag {
					break
				}

				if !isInActiveTime() {
					remainTimeToBegin := getTimeToBegin()
					log.Printf("[info] not in active time. remaining %.2f seconds to begin\n", remainTimeToBegin.Seconds())
					timeToBeginOut := time.NewTimer(remainTimeToBegin)
					select {
					case <-resetChan:
						timeToBeginOut.Stop()
						outerExitFlag = true
						continue
					case <-timeToBeginOut.C:
						log.Printf("[info] active time begin\n")
					}
				}

				remainTimeToEnd := getTimeToEnd()
				log.Printf("[info] in active time. remaining %.2f seconds to end\n", remainTimeToEnd.Seconds())
				timeToEndOut := time.NewTimer(remainTimeToEnd)

				exitFlag := false
				for {
					if exitFlag {
						timeToEndOut.Stop()
						break
					}

					select {
					case <-resetChan:
						exitFlag = true
						outerExitFlag = true
						continue
					case <-continueChan:
					case <-timeToEndOut.C:
						log.Printf("[info] active time is up\n")
						exitFlag = true
						continue
					}
				}
			}
		} else {
			log.Printf("[info] active time is disable\n")
			exitFlag := false
			for {
				if exitFlag {
					break
				}

				select {
				case <-resetChan:
					exitFlag = true
					continue
				case <-continueChan:
				}
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
