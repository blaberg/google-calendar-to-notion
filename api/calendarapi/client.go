package calendarapi

import (
	"fmt"
	"github.com/golang-module/carbon"
	"go.uber.org/zap"
	"google.golang.org/api/calendar/v3"
)

type Client struct {
	Config         Config
	CalendarClient *calendar.Service
	Logger         *zap.Logger
}

type Config struct {
	TimeZone  string
	Calendars []string
}

func (c *Client) ListEvents() (events []*calendar.Event, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("list event from calendars: %v", err)
		}
	}()
	morning := carbon.SetTimezone(c.Config.TimeZone).Now().StartOfDay().ToRfc3339String()
	night := carbon.SetTimezone(c.Config.TimeZone).Now().EndOfDay().ToRfc3339String()
	for _, cal := range c.Config.Calendars {
		ev, err := c.CalendarClient.Events.List(cal).TimeMin(morning).TimeMax(night).Do()
		if err != nil {
			return nil, err
		}
		events = append(events, ev.Items...)
	}
	return events, nil
}
