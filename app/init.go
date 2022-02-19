package app

import (
	"bytes"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"context"
	"encoding/gob"
	"fmt"
	"github.com/blendle/zapdriver"
	"github.com/jomei/notionapi"
	"go.einride.tech/aip/resourcename"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func InitNotion(
	ctx context.Context,
	config *Config,
	secretmanager *secretmanager.Client,
	logger *zap.Logger,
) (*notionapi.Client, error) {
	logger.Info("init Notion client")
	accessRequest := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf(config.Notion.APISecret),
	}
	secret, err := secretmanager.AccessSecretVersion(ctx, accessRequest)
	if err != nil {
		return nil, fmt.Errorf("fetch api key: %w", err)
	}
	tok := notionapi.Token(secret.Payload.Data)
	client := notionapi.NewClient(tok)
	return client, nil
}

// InitGoogleCalendar TODO handle expired refresh tokens.
func InitGoogleCalendar(
	ctx context.Context,
	config *Config,
	secretmanager *secretmanager.Client,
	logger *zap.Logger,
) (
	*calendar.Service,
	error,
) {
	logger.Info("init Google Calendar client")
	// Fetch credentials for Oath2.
	oath2Request := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf(config.GoogleCalendar.Oauth2),
	}
	o2, err := secretmanager.AccessSecretVersion(ctx, oath2Request)
	if err != nil {
		return nil, fmt.Errorf("fetch oath2 credentials: %w", err)
	}
	oauth2Config, err := google.ConfigFromJSON(o2.Payload.Data, calendar.CalendarReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("create oath2 config: %w", err)
	}
	// Fetch access token for calendar API.
	accessRequest := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf(config.GoogleCalendar.APISecret),
	}
	secret, err := secretmanager.AccessSecretVersion(ctx, accessRequest)
	// If no access token is found, fetch one.
	tok := &oauth2.Token{}
	switch status.Code(err) {
	case codes.OK:
		buf := bytes.NewBuffer(secret.Payload.Data)
		dec := gob.NewDecoder(buf)
		if err := dec.Decode(tok); err != nil {
			return nil, fmt.Errorf("convert secret to token: %w", err)
		}
	case codes.NotFound:
		t, err := getToken(oauth2Config, logger)
		if err != nil {
			return nil, fmt.Errorf("get oauth2 token: %w", err)
		}
		tok = t
		if err := saveToken(ctx, config, tok, secretmanager, logger); err != nil {
			return nil, fmt.Errorf("save oauth2 token: %w", err)
		}
	default:
		return nil, fmt.Errorf("fetch access token: %w", err)
	}
	httpClient := oauth2Config.Client(ctx, tok)
	srv, err := calendar.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("create new calendar service: %w", err)
	}
	return srv, nil
}

func getToken(
	config *oauth2.Config,
	logger *zap.Logger,
) (*oauth2.Token, error) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	logger.Info(fmt.Sprintf("Go to the following link in your browser then type the authorization code: "+
		"\n%v\n", authURL))
	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		return nil, fmt.Errorf("reading authentication code: %w", err)
	}
	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		return nil, fmt.Errorf("convert auth code to access token: %w", err)
	}
	logger.Info("token", zap.Any("Token", tok))
	return tok, nil
}

func saveToken(
	ctx context.Context,
	config *Config,
	tok *oauth2.Token,
	secretmanager *secretmanager.Client,
	logger *zap.Logger,
) error {
	logger.Info(fmt.Sprintf("saving token to %s", config.GoogleCalendar.APISecret))
	var project, secretID, version string
	if err := resourcename.Sscan(config.GoogleCalendar.APISecret, "projects/{project}/secrets/{secretID}/versions/{version}", &project, &secretID, &version); err != nil {
		return fmt.Errorf("scanning secret name: %w", err)
	}
	createSecretReq := &secretmanagerpb.CreateSecretRequest{
		Parent:   fmt.Sprintf("projects/%s", project),
		SecretId: secretID,
		Secret: &secretmanagerpb.Secret{
			Replication: &secretmanagerpb.Replication{
				Replication: &secretmanagerpb.Replication_Automatic_{
					Automatic: &secretmanagerpb.Replication_Automatic{},
				},
			},
		},
	}
	secret, err := secretmanager.CreateSecret(ctx, createSecretReq)
	if err != nil {
		return fmt.Errorf("creating secret: %w", err)
	}
	var payload bytes.Buffer
	enc := gob.NewEncoder(&payload)
	if err := enc.Encode(tok); err != nil {
		return fmt.Errorf("convert access token to bytes: %w", err)
	}
	addSecretVersionReq := &secretmanagerpb.AddSecretVersionRequest{
		Parent: secret.Name,
		Payload: &secretmanagerpb.SecretPayload{
			Data: payload.Bytes(),
		},
	}
	if _, err := secretmanager.AddSecretVersion(ctx, addSecretVersionReq); err != nil {
		return fmt.Errorf("store secret version: %w", err)
	}
	return nil
}

func InitSecretManagerClient(
	ctx context.Context,
	logger *zap.Logger,
) (*secretmanager.Client, func(), error) {
	logger.Info("init Secret Manager client")
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("init Secret Manager client: %w", err)
	}
	cleanup := func() {
		logger.Info("closing Secret Manager client")
		if err := client.Close(); err != nil {
			logger.Error("close Secret Manager client", zap.Error(err))
		}
	}
	return client, cleanup, nil
}

func InitLogger(
	config *Config,
) (_ *zap.Logger, _ func(), err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("init logger: %w", err)
		}
	}()
	var zapConfig zap.Config
	var zapOptions []zap.Option
	if config.Logger.Development {
		zapConfig = zap.NewDevelopmentConfig()
		zapConfig.EncoderConfig.EncodeLevel = zapcore.LowercaseColorLevelEncoder
	} else {
		zapConfig = zap.NewProductionConfig()
		zapConfig.EncoderConfig = zapdriver.NewProductionEncoderConfig()
		zapOptions = append(
			zapOptions,
			zapdriver.WrapCore(
				zapdriver.ServiceName(config.Logger.ServiceName),
				zapdriver.ReportAllErrors(true),
			),
		)
	}
	if err := zapConfig.Level.UnmarshalText([]byte(config.Logger.Level)); err != nil {
		return nil, nil, err
	}
	logger, err := zapConfig.Build(zapOptions...)
	if err != nil {
		return nil, nil, err
	}
	logger = logger.WithOptions(zap.AddStacktrace(zap.ErrorLevel))
	logger.Info("logger initialized")
	cleanup := func() {
		logger.Info("closing logger, goodbye")
		_ = logger.Sync()
	}
	return logger, cleanup, nil
}
