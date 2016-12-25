package main

import (
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"strconv"
)

type msgType int

const (
	msgPrivmsg msgType = iota + 1
	msgTimeout
)

type Message struct {
	typ msgType

	Time        time.Time         `json:"time"`
	Channel     string            `json:"channel"`
	Username    string            `json:"username"`
	DisplayName string            `json:"displayName"`
	UserType    string            `json:"userType"`
	Color       string            `json:"color"`
	Badges      map[string]int    `json:"badges"`
	Emotes      map[string]*Emote `json:"emotes"`
	Tags        map[string]string `json:"tags"`
	Text        string            `json:"text"`

	TextWithEmotes template.HTML `json:"-"`
	TimeStamp      string        `json:"-"`
}

type Emote struct {
	Name  string `json:"name"`
	ID    string `json:"id"`
	Type  string `json:"type"`
	Count int    `json:"count"`
}

func parseMessage(line string) *Message {
	if strings.Contains(line, "CLEARCHAT") {
		logrus.Debug(line)
	}
	if !strings.HasPrefix(line, "@") {
		return &Message{
			Text: line,
		}
	}
	spl := strings.SplitN(line, " :", 3)
	if len(spl) < 3 {
		return &Message{
			Text: line,
		}
	}
	tags, middle, text := spl[0], spl[1], spl[2]
	if strings.HasPrefix(text, "\u0001ACTION") {
		text = text[7 : len(text)-1]
	}
	msg := &Message{
		Time: time.Now(),
		Text: text,
		Tags: map[string]string{},
	}
	parseMiddle(msg, middle)
	parseTags(msg, tags[1:])
	if msg.typ == msgTimeout {
		msg.Username = "twitch"
		targetUser := msg.Text
		seconds, _ := strconv.Atoi(msg.Tags["ban-duration"])

		msg.Text = fmt.Sprintf("%s was timed out for %s: %s",
			targetUser,
			time.Duration(time.Duration(seconds)*time.Second),
			msg.Tags["ban-reason"])
	}
	return msg
}

func parseMiddle(msg *Message, middle string) {
	for i, c := range middle {
		if c == '!' {
			msg.Username = middle[:i]
			middle = middle[i:]
		}
	}
	start := -1
	for i, c := range middle {
		if c == ' ' {
			if start == -1 {
				start = i + 1
			} else {
				typ := middle[start:i]
				switch typ {
				case "PRIVMSG":
					msg.typ = msgPrivmsg
				case "CLEARCHAT":
					msg.typ = msgTimeout
				}
				middle = middle[i:]
			}
		}
	}
	for i, c := range middle {
		if c == '#' {
			msg.Channel = middle[i+1:]
		}
	}
}

/* @badges=broadcaster/1;color=#0B8E70;display-name=nuuls;emotes=;
id=0cabc0b9-bb6e-448c-9688-1b8ea654dc27;mod=1;room-id=100229878;
subscriber=0;tmi-sent-ts=1482614970705;turbo=0;user-id=100229878;
user-type=mod :nuuls!nuuls@nuuls.tmi.twitch.tv PRIVMSG #nuuls :NaM
*/

func parseTags(msg *Message, tagsRaw string) {
	tags := strings.Split(tagsRaw, ";")
	for _, tag := range tags {
		spl := strings.SplitN(tag, "=", 2)
		if len(spl) < 2 {
			return
		}
		value := strings.Replace(spl[1], "\\:", ";", -1)
		value = strings.Replace(value, "\\s", " ", -1)
		value = strings.Replace(value, "\\\\", "\\", -1)
		switch spl[0] {
		case "badges":
			msg.Badges = parseBadges(value)
		case "color":
			msg.Color = value
		case "display-name":
			msg.DisplayName = value
		case "emotes":
			msg.Emotes = parseTwitchEmotes(value, msg.Text)
		case "user-type":
			msg.UserType = value
		default:
			msg.Tags[spl[0]] = value
		}
	}
}

func parseBadges(badges string) map[string]int {
	m := map[string]int{}
	spl := strings.Split(badges, ",")
	for _, badge := range spl {
		s := strings.SplitN(badge, "/", 2)
		if len(s) < 2 {
			continue
		}
		n, _ := strconv.Atoi(s[1])
		m[s[0]] = n
	}
	return m
}

// 25:0-4,6-10,12-16/1902:18-22/88:24-31,33-40

func parseTwitchEmotes(emoteTag, text string) map[string]*Emote {
	emotes := map[string]*Emote{}

	if emoteTag == "" {
		return emotes
	}

	runes := []rune(text)

	emoteSlice := strings.Split(emoteTag, "/")
	for i := range emoteSlice {
		spl := strings.Split(emoteSlice[i], ":")
		pos := strings.Split(spl[1], ",")
		sp := strings.Split(pos[0], "-")
		start, _ := strconv.Atoi(sp[0])
		end, _ := strconv.Atoi(sp[1])
		id := spl[0]
		e := &Emote{
			Type:  "twitch",
			ID:    id,
			Count: strings.Count(emoteSlice[i], "-"),
			Name:  string(runes[start : end+1]),
		}

		emotes[e.Name] = e
	}
	return emotes
}
