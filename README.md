# Google Calendar to Notion

A Notion integration for importing events from Google Calendar

## Usage

To use th service, the following environment variables have to be set:

Variable Name | Description
---|---
LOGGER_SERVICENAME | Will add the ServiceContext to the log with the specified service name.
LOGGER_LEVEL| The minimum enabled logging level. Recommended: **debug**.
LOGGER_DEVELOPMENT | If the logger is set to development mode or not. Recommended: **false**.
GOOGLECALENDAR_APISECRET | To be able to use Google's APIs you have to create an API key. This can be done through the Google console, under Credentials in the API & Services section. te key should get read access to the Google Calendars API and should be configured for desktop applications. When the key is created, download it and store it in the Google Secret Manager. This Variable should be set to the full resource id of the created secret.
GOOGLECALENDAR_OAUTH2 | When a user gives the application permission to access its calendar a token will be created. The application stores that token as a secret in the secrets manager. set this variable to whatever that secret should be called.
NOTION_APISECRET | The application need access to you Notion. It gets this by being registered as an integration. Read more about Notion integrations [here][notion-integration]. The API-key for the integration should be stored as a secret in the GCP Secrets Manager. Set this variable as the full resource ID.
CALENDARSETTINGS_TIMEZONE | Your desired timezone as specified by [Carbon][carbon].
CALENDARSETTINGS_CALENDARS | An array of the calendars you want to import.
NOTIONSETTINGS_DBLINK | The full link to the database you want to import the events into. It does not need to be set up in any way. 

s## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

Please make sure to add and update tests as appropriate.

Contributions should adhere to the [Conventional Commits][commits] specification.

## License

[MIT](https://choosealicense.com/licenses/mit/)

[slack-api-key]:https://api.slack.com/authentication/token-types#bot

[commits]:https://www.conventionalcommits.org/en/v1.0.0/

[notion-integration]:https://www.notion.so/help/guides/connect-tools-to-notion-api

[carbon]:https://github.com/golang-module/carbon
