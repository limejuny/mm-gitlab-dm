package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"

	"github.com/limejuny/mm-gitlab-dm/config"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/pkg/errors"
	"github.com/samber/lo"
)

type dict map[string]any

func (d dict) d(k string) dict {
	return d[k].(map[string]any)
}

func (d dict) s(k string) string {
	return d[k].(string)
}

func (d dict) i(k string) int {
	return int(d[k].(float64))
}

type Plugin struct {
	plugin.MattermostPlugin
}

func NewPlugin() *Plugin {
	plugin := &Plugin{}

	return plugin
}

func (p *Plugin) OnActivate() error {
	config.Mattermost = p.API

	if err := p.OnConfigurationChange(); err != nil {
		return err
	}

	return nil
}

func (p *Plugin) OnConfigurationChange() error {
	if config.Mattermost == nil {
		return nil
	}
	var configuration config.Configuration

	if err := config.Mattermost.LoadPluginConfiguration(&configuration); err != nil {
		config.Mattermost.LogError("Error in LoadPluginConfiguration.", "Error", err.Error())
		return err
	}

	if err := configuration.ProcessConfiguration(); err != nil {
		config.Mattermost.LogError("Error in ProcessConfiguration.", "Error", err.Error())
		return err
	}

	if err := configuration.IsValid(); err != nil {
		config.Mattermost.LogError("Error in Validating Configuration.", "Error", err.Error())
		return err
	}

	config.SetConfig(&configuration)
	return nil
}

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return
	}
	var data dict
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		return
	}
	if _, ok := data["object_kind"]; !ok {
		return
	}

	client := model.NewAPIv4Client(MMDOMAIN)
	client.SetToken(MMTOKEN)

	if data.s("object_kind") == "merge_request" {
		author := data.d("user").s("username")
		name := data.d("user").s("name")
		url := data.d("object_attributes").s("url")
		title := data.d("object_attributes").s("title")
		description := data.d("object_attributes").s("description")
		namespace := data.d("project").s("namespace")
		project := data.d("project").s("name")
		project_url := data.d("project").s("homepage")
		action := data.d("object_attributes").s("action")

		payload := name + ` (` + author + `) ` + action + ` merge request ` + `[` + title + `](` + url + `) in [` + namespace + ` / ` + project + `](` + project_url + `)`

		var usernames []string
		if _, ok := data["assignees"]; ok {
			for _, a := range data["assignees"].([]any) {
				usernames = append(usernames, a.(map[string]any)["username"].(string))
			}
		}
		if _, ok := data["reviewers"]; ok {
			for _, a := range data["reviewers"].([]any) {
				usernames = append(usernames, a.(map[string]any)["username"].(string))
			}
		}

		config.Mattermost.LogDebug("merge_request:: retrieveUsernames()", "usernames", usernames ,"error", errors.WithStack(err), "payload", data)
		lo.ForEach(lo.Uniq(usernames), func(username string, _ int) {
			createPost(client, username, payload, title, url, description)
		})
	} else if data.s("object_kind") == "note" {
		author := data.d("user").s("username")
		name := data.d("user").s("name")
		url := data.d("object_attributes").s("url")
		description := data.d("object_attributes").s("note")
		namespace := data.d("project").s("namespace")
		project := data.d("project").s("name")
		project_url := data.d("project").s("homepage")
		project_id := data.d("project").i("id")

		switch t := data.d("object_attributes").s("noteable_type"); t {
		case "MergeRequest":
			title := data.d("merge_request").s("title")

			payload := name + ` (` + author + `) add comment to [` + title + `](` + url + `) in [` + namespace + ` / ` + project + `](` + project_url + `)`

			usernames, err := retrieveUsernames(project_id, data.d("merge_request").i("iid"))
			config.Mattermost.LogDebug("notes:: retrieveUsernames()", "usernames", usernames ,"error", errors.WithStack(err), "payload", data)
			if err == nil && len(usernames) > 0 {
				for _, username := range usernames {
					if username != author {
						createPost(client, username, payload, title, url, description)
					}
				}
			}
		case "Commit":
			// Commit
		case "Issue":
			// Issue
		}
	}
}

func createPost(client *model.Client4, username, message, title, title_link, text string) {
	config.Mattermost.LogError("[G-20] start createPost", "username", username, "message", message, "title", title, "title_link", title_link, "text", text)
	ctx := context.Background()
	config.Mattermost.LogError("[G-20] created ctx")

	user, res, err := client.GetUserByUsername(ctx, username, "")
	if err != nil || res.StatusCode >= 400 {
		config.Mattermost.LogError("[G-2] err:: GetUserByUsername", "error", errors.WithStack(err), "user", user, "res", res)
		return
	}

	channel, res, err := client.CreateDirectChannel(ctx, MMBOTID, user.Id)
	if err != nil || res.StatusCode >= 400 {
		config.Mattermost.LogError("[G-3] err:: CreateDirectChannel", "error", errors.WithStack(err), "channel", channel, "res", res)
		return
	}

	attachment := &model.SlackAttachment{
		Fallback:  "",
		Color:     "#db3b21",
		Title:     title,
		TitleLink: title_link,
		Text:      text,
	}
	post := &model.Post{
		UserId:    MMBOTID,
		ChannelId: channel.Id,
		Message:   message,
	}

	model.ParseSlackAttachment(post, []*model.SlackAttachment{attachment})

	client.CreatePost(ctx, post)
}

func main() {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	plugin.ClientMain(NewPlugin())
}
