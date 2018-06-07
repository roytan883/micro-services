package main

import (
	"strconv"
	"time"
)

func getNowTimestamp() string {
	ret := strconv.Itoa(int(time.Now().UnixNano() / 1e6))
	return ret
}

func timestampToTime(t string) (*time.Time, error) {
	timestamp, err := strconv.Atoi(t)
	if err != nil {
		return nil, err
	}
	ret := time.Unix(int64(timestamp/1e3), int64(timestamp%1e3*1e6))
	return &ret, nil
}

func genUniqueMid(userID string, cid string, mid string) string {
	return mid + "." + userID + "." + cid
}
