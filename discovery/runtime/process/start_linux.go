// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//go:build linux

package process

import (
	"bytes"
	"os"
	"strconv"
	"strings"
	"time"
)

// clockTicks is USER_HZ, the unit of /proc/<pid>/stat's start time. It is 100 on
// every Linux architecture Go supports.
const clockTicks = 100

// processStartTime returns when the process with the given PID started: boot time
// (/proc/stat btime) plus its start time in clock ticks after boot (/proc/<pid>/stat).
func processStartTime(pid int) (time.Time, bool) {
	stat, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return time.Time{}, false
	}

	// The command name (field 2) can contain spaces; fields after it follow the last ')'.
	_, rest, ok := bytes.CutLast(stat, []byte(")"))
	if !ok {
		return time.Time{}, false
	}

	fields := strings.Fields(string(rest))
	// starttime is field 22; fields[0] here is field 3 (state).
	const startTimeIndex = 22 - 3
	if len(fields) <= startTimeIndex {
		return time.Time{}, false
	}

	ticks, err := strconv.ParseInt(fields[startTimeIndex], 10, 64)
	if err != nil || ticks < 0 {
		return time.Time{}, false
	}

	boot, ok := bootTime()
	if !ok {
		return time.Time{}, false
	}

	// Divide before multiplying: ticks * time.Second overflows a Duration after ~1,068 days.
	secs, frac := ticks/clockTicks, ticks%clockTicks

	return boot.Add(time.Duration(secs)*time.Second + time.Duration(frac)*(time.Second/clockTicks)), true
}

func bootTime() (time.Time, bool) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return time.Time{}, false
	}

	for line := range strings.SplitSeq(string(data), "\n") {
		if rest, ok := strings.CutPrefix(line, "btime "); ok {
			secs, err := strconv.ParseInt(strings.TrimSpace(rest), 10, 64)
			if err != nil {
				return time.Time{}, false
			}

			return time.Unix(secs, 0), true
		}
	}

	return time.Time{}, false
}
