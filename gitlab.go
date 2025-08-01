package main

import (
	"log"

	"github.com/samber/lo"
	"gitlab.com/gitlab-org/api/client-go"
)

func retrieveUsernames(id, merge_request_iid int) ([]string, error) {
	git, err := gitlab.NewClient(GITLAB_TOKEN, gitlab.WithBaseURL(GITLAB_URL))
	if err != nil {
		return nil, err
	}

	notes, _, err := git.Notes.ListMergeRequestNotes(id, merge_request_iid, &gitlab.ListMergeRequestNotesOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
		},
	})
	if err != nil {
		log.Printf("Failed to fetch notes: %v", err)
		return nil, err
	}

	return lo.Uniq(lo.Map(notes, func(note *gitlab.Note, _ int) string {
		return note.Author.Username
	})), nil
}
