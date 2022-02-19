package app

import (
	"context"
	"go.uber.org/zap"
	"google-calendar-to-notion/api/calendarapi"
	"google-calendar-to-notion/api/notionapi"
)

type App struct {
	CalendarAPI calendarapi.Client
	NotionAPI   notionapi.Client
	Logger      *zap.Logger
}

func (a *App) Run(ctx context.Context) error {
	a.Logger.Info("running")
	title, err := a.NotionAPI.EnsureDatabase(ctx);
	if err != nil {
		return err
	}
	events, err := a.CalendarAPI.ListEvents()
	if err != nil {
		return err
	}
	a.NotionAPI.PutEvents(ctx, title, events)
	defer a.Logger.Info("stopped")
	return nil
}
