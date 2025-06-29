package util

import (
	"log"
	"time"
)

const (
	datetimeLayout string = "2006-01-02 15:04:05"
	dateLayout     string = "2006-01-02"
)

func UnixToDatetime(timestamp int64) time.Time {
	datetime := time.Unix(timestamp, 0)
	return datetime
}

func IsSameDay(today, now time.Time) bool {
	return today.Format("2006-01-02") == now.Format("2006-01-02")
}

func FormatStringToDate(timeStr string) (time.Time, error) {
	date, err := time.Parse(dateLayout, timeStr)
	if err != nil {
		log.Printf("time.Parse:%v", err)
		return time.Time{}, err
	}
	return date, nil
}

func DatetimeToDate(datetime time.Time) time.Time {
	return time.Date(datetime.Year(), datetime.Month(), datetime.Day(), 0, 0, 0, 0, datetime.Location())
}

func DatetimeToDateStr(datetime time.Time) string {
	return datetime.Format(dateLayout)
}
