package backlogslackify

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kenzo0107/backlog"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	ErrNoApiKey           = errors.New("BacklogApiKey is required")
	ErrNoBacklogUrl       = errors.New("BacklogBaseUrl is required")
	ErrNoSlackUrl         = errors.New("SlackWebhookUrl is required")
	ErrNoSlackChannel     = errors.New("SlackChannel is required")
	ErrNoSearchConditions = errors.New("SearchConditions is required")
	ErrNoCondition        = errors.New("parameter invalid, SearchConditions.Condition is blank")
	ErrDueDateInvalid     = errors.New(`BacklogDueDate is "weekend" or "end_of_month" or relative days number like "3"`)
)

// ClientOption is input options to build client
// BacklogDueDate is "weekend" or "end_of_month" or relative days number like "3"
// required parameter is below
// BacklogApiKey
// BacklogBaseUrl
// SlackWebhookUrl
// SlackChannel
// SearchConditions
type ClientOption struct {
	BacklogApiKey    string            `json:"backlog_api_key"`
	BacklogBaseUrl   string            `json:"backlog_base_url"`
	BacklogDueDate   string            `json:"backlog_due_date"`
	SlackWebhookUrl  string            `json:"slack_webhool_url"`
	SlackChannel     string            `json:"slack_channel"`
	SlackAccountName string            `json:"slack_account_name"`
	SlackIconEmoji   string            `json:"slack_icon_emoji"`
	SlackIconUrl     string            `json:"slack_icon_url"`
	IsSinglePost     bool              `json:"is_single_post"`
	DryRun           bool              `json:"dry_run"`
	SearchConditions []SearchCondition `json:"search_conditions"`
}

// SearchCondition is conditions to search backlog ticket
// it depends on github.com/kenzo0107/backlog
type SearchCondition struct {
	Name      string                    `json:"name"`
	Condition *backlog.GetIssuesOptions `json:"condition"`
}

// Client has backlog and slack client
// Build the client and call Post() function
type Client struct {
	backlogBaseUrl   string
	backlogClient    *backlog.Client
	slackClient      *slackClient
	backlogDueDate   string
	searchConditions []SearchCondition
	isSinglePost     bool
	dryRun           bool
}

// Post creates slackPost while validating ClientOption parameters.
// input ClientOption and reference time like time.Now()
func Post(opts ClientOption, t time.Time) error {
	cl, err := newClient(opts, t)
	if err != nil {
		return err
	}
	return cl.execute()
}

func newClient(opts ClientOption, t time.Time) (*Client, error) {
	if opts.BacklogApiKey == "" {
		return nil, ErrNoApiKey
	}
	if opts.BacklogBaseUrl == "" {
		return nil, ErrNoBacklogUrl
	}
	if opts.SlackWebhookUrl == "" {
		return nil, ErrNoSlackUrl
	}
	if opts.SlackChannel == "" {
		return nil, ErrNoSlackChannel
	}
	if opts.SearchConditions == nil {
		return nil, ErrNoSearchConditions
	}
	if opts.SlackAccountName == "" {
		opts.SlackAccountName = "Backlog-bot"
	}
	var dd string
	switch opts.BacklogDueDate {
	case "weekend":
		dd = weekEnd(t).Format("2006-01-02")
	case "end_of_month":
		dd = endOfMonth(t).Format("2006-01-02")
	default:
		d, err := strconv.Atoi(opts.BacklogDueDate)
		if err != nil {
			return nil, ErrDueDateInvalid
		}
		dd = t.Add(time.Duration(24*d) * time.Hour).Format("2006-01-02")
	}
	cl := Client{
		backlogClient:    backlog.New(opts.BacklogApiKey, opts.BacklogBaseUrl),
		slackClient:      newSlackClient(opts.SlackWebhookUrl, opts.SlackChannel, opts.SlackAccountName, opts.SlackIconEmoji, opts.SlackIconUrl),
		backlogDueDate:   dd,
		searchConditions: opts.SearchConditions,
		isSinglePost:     opts.IsSinglePost,
		backlogBaseUrl:   opts.BacklogBaseUrl,
		dryRun:           opts.DryRun,
	}
	return &cl, nil
}

func (cl *Client) execute() error {
	var posts []string
	for _, v := range (*cl).searchConditions {
		issues, err := cl.fetchIssues(v.Condition)
		if err != nil {
			return err
		}
		p := cl.buildPost(issues, v.Name)
		if cl.isSinglePost {
			posts = append(posts, p)
		} else {
			cl.post(p)
		}
	}
	if cl.isSinglePost {
		cl.post(strings.Join(posts, "\n"))
	}
	return nil
}

func (cl *Client) fetchIssues(condition *backlog.GetIssuesOptions) ([]*backlog.Issue, error) {
	if condition == nil {
		return nil, ErrNoCondition
	}
	if condition.DueDateUntil == nil {
		condition.DueDateUntil = &cl.backlogDueDate
	}
	issues, err := cl.backlogClient.GetIssues(condition)
	if err != nil {
		return nil, err
	}
	return issues, nil
}

func (cl *Client) buildPost(issues []*backlog.Issue, name string) string {
	msg := []string{"```"}
	if name != "" {
		msg = []string{name, "```"}
	}
	for i, v := range issues {
		if i >= 10 {
			msg = append(msg, "and more...")
			break
		}

		if v == nil {
			continue
		}
		if v.IssueKey != nil {
			msg = append(msg, fmt.Sprintf("%v/view/%v", cl.backlogBaseUrl, *v.IssueKey))
		}
		if v.Summary != nil {
			msg = append(msg, fmt.Sprintf("%+v", *v.Summary))
		}
		if v.Assignee != nil {
			msg = append(msg, fmt.Sprintf("%+v", *v.Assignee.Name))
		}
		msg = append(msg, "")
	}
	msg = append(msg, "```")
	return strings.Join(msg, "\n")
}

func (cl *Client) post(p string) {
	if cl.dryRun {
		fmt.Println(p)
	} else {
		cl.slackClient.post(p)
	}
}

func weekEnd(t time.Time) time.Time {
	wd := t.Weekday()
	if wd >= 5 {
		return t
	} else {
		return t.Add(time.Duration(24*(5-t.Weekday())) * time.Hour)
	}
}

func endOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
}

///////////////////////
// slack
///////////////////////

type slackClient struct {
	params slackParams
	url    string
}

type slackParams struct {
	Text      string `json:"text"`
	Username  string `json:"username"`
	IconEmoji string `json:"icon_emoji"`
	IconURL   string `json:"icon_url"`
	Channel   string `json:"channel"`
}

func newSlackClient(url, channel, userName, iconEmoji, iconURL string) *slackClient {
	p := slackParams{
		Username:  userName,
		IconEmoji: iconEmoji,
		IconURL:   iconURL,
		Channel:   channel,
	}

	return &slackClient{
		params: p,
		url:    url,
	}
}

func (s *slackClient) post(p string) error {
	s.params.Text = p
	params, err := json.Marshal(s.params)
	if err != nil {
		return err
	}

	resp, err := http.PostForm(
		s.url,
		url.Values{"payload": {string(params)}},
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	fmt.Printf("post message: %s\n", body)
	return nil
}
