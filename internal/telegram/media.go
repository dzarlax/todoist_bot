package telegram

import (
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// mediaInfo describes the Markdown rendering of a Telegram media attachment.
//
// For types with a file (photo/video/document/audio/voice/animation/sticker)
// Link is the direct Telegram file URL and Type is the human label.
// For types without a file (location/contact/poll/venue) the result is
// pre-rendered into Rendered and the caller should inline it as-is.
type mediaInfo struct {
	Link        string
	Type        string
	Rendered    string // pre-rendered markdown if IsDirect == true
	Description string
	IsDirect    bool
}

// asMarkdown returns the `[type](url)` or Rendered representation for inlining.
func (m *mediaInfo) asMarkdown() string {
	if m == nil {
		return ""
	}
	if m.Rendered != "" {
		return m.Rendered
	}
	return fmt.Sprintf("[%s](%s)", m.Type, m.Link)
}

// getMediaLink inspects a message and returns media info, or nil if the message
// has no supported media.
func getMediaLink(api *tgbotapi.BotAPI, msg *tgbotapi.Message) (*mediaInfo, error) {
	var fileID, mediaType, fileName string

	switch {
	case len(msg.Photo) > 0:
		fileID = msg.Photo[len(msg.Photo)-1].FileID
		mediaType = "фото"

	case msg.Video != nil:
		fileID = msg.Video.FileID
		mediaType = "видео"
		if msg.Video.FileName != "" {
			fileName = fmt.Sprintf(" (%s)", msg.Video.FileName)
		}

	case msg.Document != nil:
		fileID = msg.Document.FileID
		mediaType = "документ"
		if msg.Document.FileName != "" {
			fileName = fmt.Sprintf(" (%s)", msg.Document.FileName)
		}

	case msg.Audio != nil:
		fileID = msg.Audio.FileID
		mediaType = "аудио"
		var parts []string
		if msg.Audio.Title != "" {
			parts = append(parts, msg.Audio.Title)
		}
		if msg.Audio.Performer != "" {
			parts = append(parts, msg.Audio.Performer)
		}
		if len(parts) > 0 {
			fileName = fmt.Sprintf(" (%s)", strings.Join(parts, " - "))
		} else if msg.Audio.FileName != "" {
			fileName = fmt.Sprintf(" (%s)", msg.Audio.FileName)
		}

	case msg.Voice != nil:
		fileID = msg.Voice.FileID
		mediaType = "голосовое сообщение"

	case msg.Animation != nil:
		fileID = msg.Animation.FileID
		mediaType = "анимация"
		if msg.Animation.FileName != "" {
			fileName = fmt.Sprintf(" (%s)", msg.Animation.FileName)
		}

	case msg.Sticker != nil:
		fileID = msg.Sticker.FileID
		mediaType = "стикер"
		if msg.Sticker.Emoji != "" {
			fileName = fmt.Sprintf(" (%s)", msg.Sticker.Emoji)
		}

	case msg.Location != nil:
		link := fmt.Sprintf("https://www.google.com/maps?q=%v,%v", msg.Location.Latitude, msg.Location.Longitude)
		return &mediaInfo{
			Type:     "локация",
			Link:     link,
			Rendered: fmt.Sprintf("[локация](%s)", link),
			IsDirect: true,
		}, nil

	case msg.Contact != nil:
		nameParts := []string{}
		if msg.Contact.FirstName != "" {
			nameParts = append(nameParts, msg.Contact.FirstName)
		}
		if msg.Contact.LastName != "" {
			nameParts = append(nameParts, msg.Contact.LastName)
		}
		desc := fmt.Sprintf("%s: %s", strings.Join(nameParts, " "), msg.Contact.PhoneNumber)
		return &mediaInfo{
			Type:        "контакт",
			Link:        "tel:" + msg.Contact.PhoneNumber,
			Description: desc,
			Rendered:    fmt.Sprintf("[контакт](tel:%s) — %s", msg.Contact.PhoneNumber, desc),
			IsDirect:    true,
		}, nil

	case msg.Poll != nil:
		options := make([]string, 0, len(msg.Poll.Options))
		for _, o := range msg.Poll.Options {
			options = append(options, o.Text)
		}
		desc := fmt.Sprintf("Вопрос: %s\nВарианты: %s", msg.Poll.Question, strings.Join(options, ", "))
		return &mediaInfo{
			Type:        "опрос",
			Description: desc,
			Rendered:    "опрос — " + desc,
			IsDirect:    true,
		}, nil

	case msg.Venue != nil:
		loc := msg.Venue.Location
		link := fmt.Sprintf("https://www.google.com/maps?q=%v,%v", loc.Latitude, loc.Longitude)
		desc := fmt.Sprintf("%s, %s", msg.Venue.Title, msg.Venue.Address)
		return &mediaInfo{
			Type:        "место",
			Link:        link,
			Description: desc,
			Rendered:    fmt.Sprintf("[место](%s) — %s", link, desc),
			IsDirect:    true,
		}, nil

	default:
		return nil, nil
	}

	if fileID == "" {
		return nil, nil
	}

	link, err := api.GetFileDirectURL(fileID)
	if err != nil {
		return nil, fmt.Errorf("get file url: %w", err)
	}
	return &mediaInfo{Link: link, Type: mediaType + fileName}, nil
}
