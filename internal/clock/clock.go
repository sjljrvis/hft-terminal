package clock

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// MarketClock emits time signals aligned to an intraday trading session.
//
// Basic requirement: kick start algo at 09:15:00 (IST) sharply.
//
// It provides:
// - OpenC: fires once at market open (09:15:00)
// - MinuteC: fires on each minute boundary during the session (inclusive of open and close)
//
// The clock runs daily; after deactivation time it schedules the next day.
type MarketClock struct {
	Loc *time.Location

	OpenHour  int
	OpenMin   int
	CloseHour int
	CloseMin  int

	// Deactivate is the time after which we consider the day done and roll to next day.
	// This is useful for "post market" cleanup (example: 15:40).
	DeactivateHour int
	DeactivateMin  int

	// Now is injectable for tests; defaults to time.Now.
	Now func() time.Time
}

// NewISTMarketClock returns a market clock configured for IST:
// open 09:15, close 15:30, deactivate 15:40.
func NewISTMarketClock() *MarketClock {
	loc, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		// Fallback: fixed IST offset if tzdata isn't available.
		loc = time.FixedZone("IST", 5*60*60+30*60)
	}
	return &MarketClock{
		Loc:            loc,
		OpenHour:       9,
		OpenMin:        15,
		CloseHour:      15,
		CloseMin:       30,
		DeactivateHour: 15,
		DeactivateMin:  40,
		Now:            time.Now,
	}
}

// NewMarketClockFromStrings builds a MarketClock from time-of-day strings.
//
// - location: IANA TZ name (e.g. "Asia/Kolkata"). If empty, defaults to "Asia/Kolkata".
// - start/end/deactivate: examples "09:00", "9:00 AM", "11PM".
// - if deactivate is empty, it defaults to end + 10 minutes.
func NewMarketClockFromStrings(location, start, end, deactivate string) (*MarketClock, error) {
	if strings.TrimSpace(location) == "" {
		location = "Asia/Kolkata"
	}
	var loc *time.Location
	switch strings.ToLower(strings.TrimSpace(location)) {
	case "local":
		loc = time.Local
	default:
		var err error
		loc, err = time.LoadLocation(location)
		if err != nil {
			// If tzdata isn't available and caller wanted IST, fallback to fixed IST.
			if location == "Asia/Kolkata" {
				loc = time.FixedZone("IST", 5*60*60+30*60)
			} else {
				return nil, fmt.Errorf("load location %q: %w", location, err)
			}
		}
	}

	if strings.TrimSpace(start) == "" {
		start = "09:15"
	}
	if strings.TrimSpace(end) == "" {
		end = "15:30"
	}

	sh, sm, err := parseTimeOfDay(start)
	if err != nil {
		return nil, fmt.Errorf("parse clock.start: %w", err)
	}
	eh, em, err := parseTimeOfDay(end)
	if err != nil {
		return nil, fmt.Errorf("parse clock.end: %w", err)
	}

	var dh, dm int
	if strings.TrimSpace(deactivate) == "" {
		total := eh*60 + em + 10
		dh = (total / 60) % 24
		dm = total % 60
	} else {
		dh, dm, err = parseTimeOfDay(deactivate)
		if err != nil {
			return nil, fmt.Errorf("parse clock.deactivate: %w", err)
		}
	}

	return &MarketClock{
		Loc:            loc,
		OpenHour:       sh,
		OpenMin:        sm,
		CloseHour:      eh,
		CloseMin:       em,
		DeactivateHour: dh,
		DeactivateMin:  dm,
		Now:            time.Now,
	}, nil
}

func parseTimeOfDay(s string) (hour int, min int, err error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, 0, fmt.Errorf("empty time")
	}

	// time.Parse is picky about AM/PM casing; normalize.
	normalized := strings.ToUpper(s)

	layouts := []string{
		"15:04",
		"15:04:05",
		"3:04 PM",
		"3:04PM",
		"3 PM",
		"3PM",
	}
	for _, layout := range layouts {
		t, e := time.Parse(layout, normalized)
		if e == nil {
			return t.Hour(), t.Minute(), nil
		}
	}

	return 0, 0, fmt.Errorf("invalid time %q (expected like 09:00, 9:00 AM, or 11PM)", s)
}

// Start begins emitting session signals in a background goroutine.
//
// - openC fires at 09:15:00 if started before open; if started mid-session it fires at the next minute boundary
// - minuteC fires on each minute boundary during the session, inclusive of open and close
//
// Both channels close when ctx is cancelled.
func (c *MarketClock) Start(ctx context.Context) (openC <-chan time.Time, minuteC <-chan time.Time) {
	if c == nil {
		ch1 := make(chan time.Time)
		ch2 := make(chan time.Time)
		close(ch1)
		close(ch2)
		return ch1, ch2
	}
	if c.Loc == nil {
		c.Loc = time.Local
	}
	if c.Now == nil {
		c.Now = time.Now
	}

	openCh := make(chan time.Time, 1)
	minCh := make(chan time.Time, 1)

	go func() {
		defer close(openCh)
		defer close(minCh)

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			now := c.Now().In(c.Loc)
			open, closeT, deactivate := c.sessionTimesFor(now)

			// If past deactivation, roll to next day.
			if !now.Before(deactivate) {
				nextOpen, _, _ := c.sessionTimesFor(open.Add(24 * time.Hour))
				if !sleepUntil(ctx, nextOpen) {
					return
				}
				continue
			}

			// Open: wait until open if before; otherwise align to next minute boundary if within session.
			if now.Before(open) {
				if !sleepUntil(ctx, open) {
					return
				}
				select {
				case openCh <- open:
				case <-ctx.Done():
					return
				}
				now = open
			} else if now.Before(closeT) || now.Equal(closeT) {
				kick := nextMinuteBoundary(now)
				if kick.Before(open) {
					kick = open
				}
				if !sleepUntil(ctx, kick) {
					return
				}
				select {
				case openCh <- kick:
				case <-ctx.Done():
					return
				}
				now = kick
			} else {
				// After close but before deactivate: wait for next day's open.
				nextOpen, _, _ := c.sessionTimesFor(open.Add(24 * time.Hour))
				if !sleepUntil(ctx, nextOpen) {
					return
				}
				continue
			}

			// Minute ticks: aligned and inclusive.
			firstTick := open
			if now.After(open) {
				firstTick = nextMinuteBoundary(now)
				if firstTick.Before(open) {
					firstTick = open
				}
			}

			for t := firstTick; !t.After(closeT); t = t.Add(time.Minute) {
				if !sleepUntil(ctx, t) {
					return
				}
				select {
				case minCh <- t:
				case <-ctx.Done():
					return
				}
			}

			// Wait for deactivation, then proceed to next day.
			if !sleepUntil(ctx, deactivate) {
				return
			}
		}
	}()

	return openCh, minCh
}

func (c *MarketClock) sessionTimesFor(now time.Time) (open time.Time, closeT time.Time, deactivate time.Time) {
	year, month, day := now.Date()
	open = time.Date(year, month, day, c.OpenHour, c.OpenMin, 0, 0, c.Loc)
	closeT = time.Date(year, month, day, c.CloseHour, c.CloseMin, 0, 0, c.Loc)
	deactivate = time.Date(year, month, day, c.DeactivateHour, c.DeactivateMin, 0, 0, c.Loc)
	return open, closeT, deactivate
}

func nextMinuteBoundary(t time.Time) time.Time {
	tr := t.Truncate(time.Minute)
	if t.Equal(tr) {
		return tr
	}
	return tr.Add(time.Minute)
}

func sleepUntil(ctx context.Context, t time.Time) bool {
	d := time.Until(t)
	if d <= 0 {
		return true
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}
