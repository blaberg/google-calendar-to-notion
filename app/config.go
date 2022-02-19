package app

import (
	"google-calendar-to-notion/api/calendarapi"
	"google-calendar-to-notion/api/notionapi"
)

type Config struct {
	Logger struct {
		ServiceName string `required:"true"`
		Level       string `required:"true"`
		Development bool   `required:"true"`
	}
	GoogleCalendar struct {
		APISecret string `required:"true"`
		Oauth2     string `required:"true"`
	}
	Notion struct{
		APISecret string `required:"true"`
	}
	CalendarSettings calendarapi.Config
	NotionSettings notionapi.Config
}
