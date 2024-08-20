/*
Copyright 2021 The KodeRover Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"fmt"
	"time"
)

// Age returns a string representing the time elapsed since unixTime in the form "1d", "2h", "3m", or "4s".
func Age(unixTime int64) string {
	duration := time.Now().Unix() - unixTime

	if duration >= 0 && duration < 60 {
		return fmt.Sprintf("%ds", duration)
	}

	if duration >= 60 && duration < 60*60 {
		return fmt.Sprintf("%dm", duration/60)
	}

	if duration >= 60*60 && duration < 60*60*24 {
		return fmt.Sprintf("%dh", duration/(60*60))
	}

	if duration >= 60*60*24 {
		return fmt.Sprintf("%dd", duration/(60*60*24))
	}
	return "0s"
}

// GetDailyStartTimestamps returns a slice of Unix timestamps representing the start of each day between startTimestamp and endTimestamp.
// It is worth mentioning that we will add an additional date timestamp just to make life easier.
func GetDailyStartTimestamps(startTimestamp, endTimestamp int64) []int64 {
	startTime := time.Unix(startTimestamp, 0)
	endTime := time.Unix(endTimestamp, 0)

	numDays := int(endTime.Sub(startTime).Hours() / 24)

	dailyStartTimestamps := make([]int64, numDays+2)

	for i := 0; i <= numDays; i++ {
		// Calculate the start of the current day
		currentDay := startTime.Add(time.Duration(i*24) * time.Hour)
		startOfDay := time.Date(currentDay.Year(), currentDay.Month(), currentDay.Day(), 0, 0, 0, 0, time.UTC)

		// Convert the start of the day to a Unix timestamp
		dailyStartTimestamps[i] = startOfDay.Unix()
	}

	// add a day to the last day
	dailyStartTimestamps[numDays+1] = endTimestamp + 24*60*60

	return dailyStartTimestamps
}

func GetMidnightTimestamp(timestamp int64) int64 {
	t := time.Unix(timestamp, 0).Local()
	midnight := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	return midnight.Unix()
}

// GetMonday returns the unix timestamp of the monday of this week
func GetMonday(t time.Time) time.Time {
	// Find the difference in days to the previous Monday
	daysToMonday := int(time.Monday - t.Weekday())
	if daysToMonday > 0 {
		daysToMonday = -6 // If today is after Monday, adjust to go back to the previous Monday
	}

	// Calculate the date of this week's Monday
	thisWeeksMonday := t.AddDate(0, 0, daysToMonday)

	// Set the time to the start of the day
	thisWeeksMonday = time.Date(thisWeeksMonday.Year(), thisWeeksMonday.Month(), thisWeeksMonday.Day(), 0, 0, 0, 0, time.Local)

	return thisWeeksMonday
}

func GetFirstOfMonthDay(now time.Time) int64 {
	now = now.Local()
	// get current year and month
	year, month, _ := now.Date()

	// build the firstOfMonth
	firstOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, now.Location())

	// convert to unix timestamp
	return firstOfMonth.Unix()
}

// GetDaysInCurrentMonth returns the number of days in the current month.
func GetDaysInCurrentMonth(now time.Time) int {
	now = now.Local()
	// Get the next month's time
	nextMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location())

	// Get the number of days in the current month
	daysInMonth := nextMonth.Day()

	return daysInMonth
}

// IsSameDay checks if two timestamps represent the same day.
func IsSameDay(timestamp1 int64, timestamp2 int64) bool {
	t1 := time.Unix(timestamp1, 0).Local()
	t2 := time.Unix(timestamp2, 0).Local()

	return t1.Year() == t2.Year() && t1.Month() == t2.Month() && t1.Day() == t2.Day()
}

func UnixStampToCronExpr(unixStamp int64) string {
	// 将 Unix 时间戳转换为时间
	t := time.Unix(unixStamp, 0)

	// 提取分钟、小时、日期等信息
	// second := t.Second()
	minute := t.Minute()
	hour := t.Hour()
	day := t.Day()
	month := int(t.Month())
	// year := t.Year()

	// 构建 Cron 表达式
	cronExpr := fmt.Sprintf("%d %d %d %d", minute, hour, day, month)
	return cronExpr
}

func GetEndOfWeekDayTimeStamp(t time.Time) int64 {
	// 找到该时间的星期几
	weekday := t.Weekday()

	// 计算距离周日的天数
	daysUntilSunday := (7 - int(weekday)) % 7

	// 计算该周结束时那天的 0 点
	endOfWeek := t.AddDate(0, 0, daysUntilSunday).Truncate(24 * time.Hour)

	// 将计算结果转换为 Unix 时间戳
	return endOfWeek.Unix()
}
