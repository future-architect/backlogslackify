package backlogslackify

import (
	"errors"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/kenzo0107/backlog"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		opts    ClientOption
		want    Client
		wantErr error
	}{
		{
			name: "success",
			opts: ClientOption{
				BacklogApiKey:    "dummy",
				BacklogBaseUrl:   "https://backlog.example.com",
				BacklogDueDate:   "weekend",
				SlackWebhookUrl:  "https://slack.example.com",
				SlackChannel:     "test-channel",
				SlackAccountName: "test-slack-bot",
				SlackIconEmoji:   ":backlog:",
				SlackIconUrl:     "",
				IsSinglePost:     true,
				DryRun:           true,
				SearchConditions: []SearchCondition{
					{
						Name: "test",
						Condition: &backlog.GetIssuesOptions{
							ProjectIDs:  []int{1},
							CategoryIDs: []int{1},
							StatusIDs:   []int{1},
						},
					},
				},
			},
			want: Client{
				backlogBaseUrl: "https://backlog.example.com",
				backlogDueDate: "2020-01-03",
				searchConditions: []SearchCondition{
					{
						Name: "test",
						Condition: &backlog.GetIssuesOptions{
							ProjectIDs:  []int{1},
							CategoryIDs: []int{1},
							StatusIDs:   []int{1},
						},
					},
				},
				isSinglePost: true,
				dryRun:       true,
			},
			wantErr: nil,
		},
		{
			name: "ErrNoApiKey",
			opts: ClientOption{
				BacklogApiKey:    "",
				BacklogBaseUrl:   "https://backlog.example.com",
				BacklogDueDate:   "weekend",
				SlackWebhookUrl:  "https://slack.example.com",
				SlackChannel:     "test-channel",
				SlackAccountName: "test-slack-bot",
				SlackIconEmoji:   ":backlog:",
				SlackIconUrl:     "",
				IsSinglePost:     true,
				DryRun:           true,
				SearchConditions: []SearchCondition{
					{
						Name: "test",
						Condition: &backlog.GetIssuesOptions{
							ProjectIDs:  []int{1},
							CategoryIDs: []int{1},
							StatusIDs:   []int{1},
						},
					},
				},
			},
			wantErr: ErrNoApiKey,
		},
		{
			name: "ErrNoBacklogUrl",
			opts: ClientOption{
				BacklogApiKey:    "dummy",
				BacklogBaseUrl:   "",
				BacklogDueDate:   "weekend",
				SlackWebhookUrl:  "https://slack.example.com",
				SlackChannel:     "test-channel",
				SlackAccountName: "test-slack-bot",
				SlackIconEmoji:   ":backlog:",
				SlackIconUrl:     "",
				IsSinglePost:     true,
				DryRun:           true,
				SearchConditions: []SearchCondition{
					{
						Name: "test",
						Condition: &backlog.GetIssuesOptions{
							ProjectIDs:  []int{1},
							CategoryIDs: []int{1},
							StatusIDs:   []int{1},
						},
					},
				},
			},
			wantErr: ErrNoBacklogUrl,
		},
		{
			name: "ErrNoSlackUrl",
			opts: ClientOption{
				BacklogApiKey:    "dummy",
				BacklogBaseUrl:   "https://backlog.example.com",
				BacklogDueDate:   "weekend",
				SlackWebhookUrl:  "",
				SlackChannel:     "test-channel",
				SlackAccountName: "test-slack-bot",
				SlackIconEmoji:   ":backlog:",
				SlackIconUrl:     "",
				IsSinglePost:     true,
				DryRun:           true,
				SearchConditions: []SearchCondition{
					{
						Name: "test",
						Condition: &backlog.GetIssuesOptions{
							ProjectIDs:  []int{1},
							CategoryIDs: []int{1},
							StatusIDs:   []int{1},
						},
					},
				},
			},
			wantErr: ErrNoSlackUrl,
		},
		{
			name: "ErrNoSlackChannel",
			opts: ClientOption{
				BacklogApiKey:    "dummy",
				BacklogBaseUrl:   "https://backlog.example.com",
				BacklogDueDate:   "weekend",
				SlackWebhookUrl:  "https://slack.example.com",
				SlackChannel:     "",
				SlackAccountName: "test-slack-bot",
				SlackIconEmoji:   ":backlog:",
				SlackIconUrl:     "",
				IsSinglePost:     true,
				DryRun:           true,
				SearchConditions: []SearchCondition{
					{
						Name: "test",
						Condition: &backlog.GetIssuesOptions{
							ProjectIDs:  []int{1},
							CategoryIDs: []int{1},
							StatusIDs:   []int{1},
						},
					},
				},
			},
			wantErr: ErrNoSlackChannel,
		},
		{
			name: "ErrDueDateInvalid",
			opts: ClientOption{
				BacklogApiKey:    "dummy",
				BacklogBaseUrl:   "https://backlog.example.com",
				BacklogDueDate:   "invalid",
				SlackWebhookUrl:  "https://slack.example.com",
				SlackChannel:     "test-channel",
				SlackAccountName: "test-slack-bot",
				SlackIconEmoji:   ":backlog:",
				SlackIconUrl:     "",
				IsSinglePost:     true,
				DryRun:           true,
				SearchConditions: []SearchCondition{
					{
						Name: "test",
						Condition: &backlog.GetIssuesOptions{
							ProjectIDs:  []int{1},
							CategoryIDs: []int{1},
							StatusIDs:   []int{1},
						},
					},
				},
			},
			wantErr: ErrDueDateInvalid,
		},
	}

	opts := []cmp.Option{
		cmpopts.IgnoreUnexported(Client{}),
		cmpopts.IgnoreFields(Client{}, "backlogClient", "slackClient"),
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testTime := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
			got, err := NewClient(tt.opts, testTime)
			if err != nil && !errors.Is(err, tt.wantErr) {
				t.Fatal(err)
			}
			if got == nil {
				return
			}
			if diff := cmp.Diff(*got, tt.want, opts...); diff != "" {
				t.Errorf("Client mismatch (-got +want):\n%s", diff)
			}
		})
	}
}

func TestWeekEnd(t *testing.T) {
	tests := []struct {
		name string
		in   time.Time
		want string
	}{
		{
			name: "Sunday",
			in:   time.Date(2000, time.January, 2, 0, 0, 0, 0, time.UTC),
			want: "2000-01-07",
		},
		{
			name: "Thursday",
			in:   time.Date(2000, time.January, 6, 0, 0, 0, 0, time.UTC),
			want: "2000-01-07",
		},
		{
			name: "Friday",
			in:   time.Date(2000, time.January, 7, 0, 0, 0, 0, time.UTC),
			want: "2000-01-07",
		},
		{
			name: "Saturday",
			in:   time.Date(2000, time.January, 8, 0, 0, 0, 0, time.UTC),
			want: "2000-01-08",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := weekEnd(tt.in)
			if got.Format("2006-01-02") != tt.want {
				t.Errorf("got=%s, want=%s", got.Format("2006-01-02"), tt.want)
			}
		})
	}
}

func TestEndOfMonth(t *testing.T) {
	tests := []struct {
		name string
		in   time.Time
		want string
	}{
		{
			name: "First Of Month",
			in:   time.Date(2000, time.February, 1, 0, 0, 0, 0, time.UTC),
			want: "2000-02-29",
		},
		{
			name: "EndOfMonth",
			in:   time.Date(2000, time.February, 29, 0, 0, 0, 0, time.UTC),
			want: "2000-02-29",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := endOfMonth(tt.in)
			if got.Format("2006-01-02") != tt.want {
				t.Errorf("got=%s, want=%s", got.Format("2006-01-02"), tt.want)
			}
		})
	}
}
