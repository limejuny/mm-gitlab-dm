package main

import (
	"log"

	"github.com/limejuny/mm-gitlab-dm/config"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"gitlab.com/gitlab-org/api/client-go"
)

func retrieveUsernames(id, merge_request_iid int) ([]string, error) {
	config.Mattermost.LogError("[4-0] start:: retrieveUsernames()", "id", id, "merge_request_iid", merge_request_iid)
	git, err := gitlab.NewClient(GITLAB_TOKEN, gitlab.WithBaseURL(GITLAB_URL))
	config.Mattermost.LogError("[4-1] err:: retrieveUsernames()", "git", git, "error", errors.WithStack(err))
	if err != nil {
		return nil, err
	}

	notes, _, err := git.Notes.ListMergeRequestNotes(id, merge_request_iid, &gitlab.ListMergeRequestNotesOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
		},
	})
	config.Mattermost.LogError("[4-2] err:: retrieveUsernames()", "notes", notes, "error", errors.WithStack(err))
	if err != nil {
		log.Printf("Failed to fetch notes: %v", err)
		return nil, err
	}

	return lo.Uniq(lo.Map(notes, func(note *gitlab.Note, _ int) string {
		return note.Author.Username
	})), nil
}
