//+build wireinject

package app

import (
	"context"
	"github.com/google/wire"
	"go.uber.org/zap"
	"google-calendar-to-notion/api/calendarapi"
	"google-calendar-to-notion/api/notionapi"
)

func InitApp(ctx context.Context, logger *zap.Logger, config *Config) (*App, func(), error) {
	panic(
		wire.Build(
			wire.Struct(new(App), "*"),
			InitSecretManagerClient,
			InitGoogleCalendar,
			InitNotion,
			wire.Struct(new(calendarapi.Client), "*"), wire.FieldsOf(&config, "CalendarSettings"),
			wire.Struct(new(notionapi.Client), "*"), wire.FieldsOf(&config, "NotionSettings"),
		),
	)
}
