package instagram

import (
	"errors"
	"log"
	"time"
)

var layouts = [...]string{
	"2006-01-02 15:04",
	"2006-01-02",
}

type timeSource interface {
	time() time.Time
}

type rangeStatus int

const (
	cont rangeStatus = iota + 1
	inRange
	outOfRange
)

type rangeInfo interface {
	includes(b timeSource) rangeStatus
}

func parseTime(val string) (time.Time, error) {
	var err error
	for _, l := range layouts {
		var tim time.Time
		tim, err = time.ParseInLocation(l, val, time.Local)
		if err == nil {
			return tim, nil
		}
	}
	return time.Time{}, err
}

func createRangeInfo(from, to string, offset, count int) (rangeInfo, error) {
	if from != "" && offset != -1 {
		return nil, errors.New("mutual exclusive options 'offset' and 'from'")
	}
	if to != "" && count > 0 {
		log.Println(to, count)
		return nil, errors.New("mutual exclusive options 'count' and 'to'")
	}
	const (
		flagFrom  = 0x01
		flagTo    = 0x02
		flagOff   = 0x04
		flagCount = 0x08

		rTimeRange      = flagFrom | flagTo
		rCountRange     = flagOff | flagCount
		rCountRange2    = flagCount
		rCountTimeRange = flagOff | flagTo
		rTimeCountRange = flagFrom | flagCount
	)
	var (
		err      error
		flags    uint
		timeFrom time.Time
		timeTo   time.Time
	)
	if from != "" {
		timeFrom, err = parseTime(from)
		if err != nil {
			return nil, err
		}
		flags |= flagFrom
	}
	if to != "" {
		timeTo, err = parseTime(to)
		if err != nil {
			return nil, err
		}
		flags |= flagTo
	}
	if offset > -1 {
		flags |= flagOff
	}
	if count > 0 {
		flags |= flagCount
	}
	switch flags {
	case rTimeRange:
		return &timeRange{start: timeFrom, end: timeTo}, nil
	case rCountRange, rCountRange2:
		off := 0
		if offset > -1 {
			off = offset
		}
		return &countRange{off: off, count: count}, nil
	case rCountTimeRange:
		return &countTimeRange{off: offset, to: timeTo}, nil
	case rTimeCountRange:
		return &timeCountRange{from: timeFrom, count: count}, nil
	}
	return nopRange{}, nil
}

type timeCountRange struct {
	from  time.Time
	count int
	curr  int
}

func (t *timeCountRange) includes(i timeSource) rangeStatus {
	tim := i.time()
	if tim.After(t.from) {
		return cont
	}
	if t.curr < t.count {
		t.curr++
		return inRange
	}
	return outOfRange
}

type countTimeRange struct {
	off  int
	curr int
	to   time.Time
}

func (t countTimeRange) includes(i timeSource) rangeStatus {
	tim := i.time()
	if tim.Before(t.to) || tim.Equal(t.to) {
		return outOfRange
	}
	if tim.After(t.to) {
		defer func() { t.curr++ }()
		if t.curr < t.off {
			return cont
		}
		return inRange
	}
	return outOfRange
}

type countRange struct {
	off   int
	count int
	next  int
}

func (c countRange) includes(timeSource) rangeStatus {
	if c.next < c.off {
		c.next++
		return cont
	}
	if c.next >= c.off && c.next < c.off+c.count {
		c.next++
		return inRange
	}
	return outOfRange
}

type timeRange struct {
	start time.Time
	end   time.Time
}

func (t *timeRange) includes(i timeSource) rangeStatus {
	c := i.time()
	if c.After(t.start) {
		return cont
	}
	if c.Equal(t.start) {
		return inRange
	}
	if c.Before(t.start) && c.After(t.end) {
		return inRange
	}
	return outOfRange
}

type nopRange struct{}

func (nopRange) includes(timeSource) rangeStatus {
	return inRange
}