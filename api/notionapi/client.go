package notionapi

import (
	"context"
	"fmt"
	"github.com/jomei/notionapi"
	"go.uber.org/zap"
	"google.golang.org/api/calendar/v3"
	"strings"
	"sync"
	"time"
)

type Client struct {
	Config *Config
	Client *notionapi.Client
	Logger *zap.Logger
}

type Config struct {
	DBLink string
}

// EnsureDatabase checks that the database exists and has the right properties. If some properties are missing they
// be added to the database.
func (c *Client) EnsureDatabase(ctx context.Context) (string, error) {
	c.Logger.Info("ensuring database")
	Id := getDBId(c.Config.DBLink)
	DB, err := c.Client.Database.Get(ctx, notionapi.DatabaseID(Id))
	if err != nil {
		return "", fmt.Errorf("fetching DB with id %s: %w", Id, err)
	}
	props := []string{"Task", "Id", "Start", "End"}
	confs := make(map[string]notionapi.PropertyConfig)
	for _, s := range props {
		if _, ok := DB.Properties[s]; ok {
			delete(DB.Properties, s)
			continue
		}
		confs["Time"] = notionapi.FormulaPropertyConfig{
			Type: notionapi.PropertyConfigTypeFormula,
			Formula: notionapi.FormulaConfig{
				Expression: "concat(formatDate(prop(\"Start\"),\"HH:mm\"),\" - \",formatDate(prop(\"End\"),\"HH:mm\"))",
			},
		}
		delete(DB.Properties, "Time")
		confs["Id"] = notionapi.RichTextPropertyConfig{
			Type: notionapi.PropertyConfigTypeRichText,
		}
		delete(DB.Properties, "Id")
		confs["Start"] = notionapi.DatePropertyConfig{
			Type: notionapi.PropertyConfigTypeDate,
		}
		delete(DB.Properties, "Start")
		confs["End"] = notionapi.DatePropertyConfig{
			Type: notionapi.PropertyConfigTypeDate,
		}
		delete(DB.Properties, "End")
		break
	}
	var title string
	for k := range DB.Properties {
		if DB.Properties[k].GetType() == notionapi.PropertyConfigTypeTitle {
			title = k
			continue
		}
		confs[k] = nil
	}
	if len(confs) == 0 {
		return "", nil
	}
	_, err = c.Client.Database.Update(ctx, notionapi.DatabaseID(Id), &notionapi.DatabaseUpdateRequest{
		Properties: confs,
	})
	if err != nil {
		return "", fmt.Errorf("update database: %w", err)
	}
	return title, nil
}

// PutEvents parses and puts an array of calendar.Event into a notion database.
func (c *Client) PutEvents(ctx context.Context, title string, events []*calendar.Event) {
	c.Logger.Info("putting events")
	var wg sync.WaitGroup
	for _, event := range events {
		wg.Add(1)
		go func(ctx context.Context, event *calendar.Event) {
			defer wg.Done()
			err := c.PutEvent(ctx, title, event)
			if err != nil {
				c.Logger.Warn(fmt.Sprintf("put events: %v", err))
			}
		}(ctx, event)
	}
	wg.Wait()
}

// PutEvent parses and puts a calendar.Event into a notion database.
func (c *Client) PutEvent(ctx context.Context, title string, event *calendar.Event) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("put event: %v", err)
		}
	}()
	c.Logger.Info(fmt.Sprintf("put event: %s", event.Summary))
	start, err := time.Parse(time.RFC3339, event.Start.DateTime)
	if err != nil {
		return fmt.Errorf("convert start time to RFC3339: %w", err)
	}
	startObj := notionapi.Date(start)
	end, err := time.Parse(time.RFC3339, event.End.DateTime)
	if err != nil {
		return fmt.Errorf("convert end time to RFC3339: %w", err)
	}
	endObj := notionapi.Date(end)
	body := parseEventBody(event)
	_, err = c.Client.Page.Create(ctx, &notionapi.PageCreateRequest{
		Parent: notionapi.Parent{
			Type:       "database_id",
			DatabaseID: notionapi.DatabaseID(getDBId(c.Config.DBLink)),
		},
		Properties: notionapi.Properties{
			title: notionapi.TitleProperty{
				Title: []notionapi.RichText{
					{Text: notionapi.Text{Content: event.Summary}},
				},
			},
			"Id": notionapi.RichTextProperty{
				RichText: []notionapi.RichText{
					{
						Type:      notionapi.ObjectTypeText,
						Text:      notionapi.Text{Content: event.Id},
						PlainText: event.Id,
					},
				},
			},
			"Start": notionapi.DateProperty{
				Date: notionapi.DateObject{
					Start: &startObj,
				},
			},
			"End": notionapi.DateProperty{
				Date: notionapi.DateObject{
					Start: &endObj,
				},
			},
		},
		Children: body,
	})
	return err
}

func parseEventBody(event *calendar.Event) (body []notionapi.Block) {
	if event.Description != "" {
		desc := parseDescription(event)
		body = append(body, desc...)
	}
	if len(event.Attendees) != 0 {
		att := parseAttendees(event)
		body = append(body, att...)
	}
	if event.HangoutLink != "" {
		meet := parseMeeting(event)
		body = append(body, meet...)
	}
	if event.Location != "" {
		loc := parseLocation(event)
		body = append(body, loc...)
	}
	if len(event.Attachments) != 0 {
		atch := parseAttachments(event)
		body = append(body, atch...)
	}
	return body
}

func parseDescription(event *calendar.Event) notionapi.Blocks {
	desc := notionapi.Blocks{
		notionapi.Heading2Block{
			BasicBlock: notionapi.BasicBlock{
				Object: notionapi.ObjectTypeBlock,
				Type:   notionapi.BlockTypeHeading2,
			},
			Heading2: notionapi.Heading{
				Text: []notionapi.RichText{
					{
						Text: notionapi.Text{Content: "Description"},
					},
				},
			},
		},
		notionapi.ParagraphBlock{
			BasicBlock: notionapi.BasicBlock{
				Object: notionapi.ObjectTypeBlock,
				Type:   notionapi.BlockTypeParagraph,
			},
			Paragraph: notionapi.Paragraph{
				Text: []notionapi.RichText{
					{
						Text: notionapi.Text{
							Content: event.Description,
						},
					},
				},
			},
		},
	}
	return desc
}

func parseAttendees(event *calendar.Event) notionapi.Blocks {
	att := notionapi.Blocks{
		notionapi.Heading2Block{
			BasicBlock: notionapi.BasicBlock{
				Object: notionapi.ObjectTypeBlock,
				Type:   notionapi.BlockTypeHeading2,
			},
			Heading2: notionapi.Heading{
				Text: []notionapi.RichText{
					{
						Text: notionapi.Text{Content: "Attendees"},
					},
				},
			},
		},
	}
	if event.AttendeesOmitted {
		att = append(att, notionapi.ParagraphBlock{
			BasicBlock: notionapi.BasicBlock{
				Object: notionapi.ObjectTypeBlock,
				Type:   notionapi.BlockTypeParagraph,
			},
			Paragraph: notionapi.Paragraph{
				Text: []notionapi.RichText{
					{
						Text: notionapi.Text{
							Content: "Attendees omitted...",
						},
					},
				},
			},
		})
	}
	for _, a := range event.Attendees {
		var sb strings.Builder
		if a.Organizer {
			sb.WriteString("Organizer: ")
		}
		if a.DisplayName != "" {
			sb.WriteString(fmt.Sprintf("%s (%s)", a.DisplayName, a.Email))
		} else {
			sb.WriteString(a.Email)
		}
		att = append(att, notionapi.BulletedListItemBlock{
			BasicBlock: notionapi.BasicBlock{
				Object: notionapi.ObjectTypeBlock,
				Type:   notionapi.BlockTypeBulletedListItem,
			},
			BulletedListItem: notionapi.ListItem{
				Text: []notionapi.RichText{
					{Text: notionapi.Text{Content: sb.String()}},
				},
			},
		})
	}
	return att
}

func parseLocation(event *calendar.Event) notionapi.Blocks {
	loc := notionapi.Blocks{
		notionapi.Heading2Block{
			BasicBlock: notionapi.BasicBlock{
				Object: notionapi.ObjectTypeBlock,
				Type:   notionapi.BlockTypeHeading2,
			},
			Heading2: notionapi.Heading{
				Text: []notionapi.RichText{
					{
						Text: notionapi.Text{Content: "Location"},
					},
				},
			},
		},
		notionapi.ParagraphBlock{
			BasicBlock: notionapi.BasicBlock{
				Object: notionapi.ObjectTypeBlock,
				Type:   notionapi.BlockTypeParagraph,
			},
			Paragraph: notionapi.Paragraph{
				Text: []notionapi.RichText{
					{
						Text: notionapi.Text{
							Content: event.Location,
						},
					},
				},
			},
		},
	}
	return loc
}

func parseAttachments(event *calendar.Event) notionapi.Blocks {
	atch := notionapi.Blocks{
		notionapi.Heading2Block{
			BasicBlock: notionapi.BasicBlock{
				Object: notionapi.ObjectTypeBlock,
				Type:   notionapi.BlockTypeHeading2,
			},
			Heading2: notionapi.Heading{
				Text: []notionapi.RichText{
					{
						Text: notionapi.Text{Content: "Attachments"},
					},
				},
			},
		},
	}
	for _, att := range event.Attachments {
		fileObj := notionapi.FileObject{
			URL: att.FileUrl,
		}
		a := notionapi.FileBlock{
			BasicBlock: notionapi.BasicBlock{
				Object: notionapi.ObjectTypeBlock,
				Type:   notionapi.BlockTypeFile,
			},
			File: notionapi.BlockFile{
				Caption: []notionapi.RichText{
					{
						Text: notionapi.Text{
							Content: att.Title,
						},
					},
				},
				Type:     notionapi.FileTypeExternal,
				External: &fileObj,
			},
		}
		atch = append(atch, a)
	}
	return atch
}

func parseMeeting(event *calendar.Event) notionapi.Blocks {
	meet := notionapi.Blocks{
		notionapi.Heading2Block{
			BasicBlock: notionapi.BasicBlock{
				Object: notionapi.ObjectTypeBlock,
				Type:   notionapi.BlockTypeHeading2,
			},
			Heading2: notionapi.Heading{
				Text: []notionapi.RichText{
					{
						Text: notionapi.Text{
							Content: "Join Google Meet",
							Link: &notionapi.Link{
								Url: event.HangoutLink,
							},
						},
					},
				},
			},
		},
	}
	return meet
}

func getDBId(link string) string {
	subs := strings.Split(link[len("https://www.notion.so/"):], "?v")
	return subs[0]
}
