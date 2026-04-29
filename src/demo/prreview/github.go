package prreview

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type PRRef struct {
	Owner  string
	Repo   string
	Number int
}

type PRMetadata struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	Body        string `json:"body"`
	Author      string `json:"author"`
	BaseBranch  string `json:"base_branch"`
	HeadBranch  string `json:"head_branch"`
	State       string `json:"state"`
	ChangedFiles int    `json:"changed_files"`
	Additions   int    `json:"additions"`
	Deletions   int    `json:"deletions"`
}

type PRPayload struct {
	Ref      PRRef       `json:"-"`
	Metadata PRMetadata  `json:"metadata"`
	Diff     string      `json:"diff"`
	Files    []PRFile    `json:"files"`
}

type PRFile struct {
	Filename  string `json:"filename"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Patch     string `json:"patch,omitempty"`
}

type GitHubClient struct {
	token  string
	http   *http.Client
}

func NewGitHubClient(token string) *GitHubClient {
	return &GitHubClient{
		token: token,
		http:  &http.Client{Timeout: 20 * time.Second},
	}
}

var prURLPattern = regexp.MustCompile(`^https?://github\.com/([^/]+)/([^/]+)/pull/(\d+)`)
var issueURLPattern = regexp.MustCompile(`^https?://github\.com/([^/]+)/([^/]+)/issues/(\d+)`)

func ParsePRURL(rawURL string) (PRRef, error) {
	m := prURLPattern.FindStringSubmatch(rawURL)
	if len(m) != 4 {
		return PRRef{}, fmt.Errorf("invalid GitHub PR URL: %s", rawURL)
	}
	num, err := strconv.Atoi(m[3])
	if err != nil {
		return PRRef{}, fmt.Errorf("invalid PR number: %s", m[3])
	}
	return PRRef{Owner: m[1], Repo: m[2], Number: num}, nil
}

type IssueRef struct {
	Owner  string
	Repo   string
	Number int
}

func ParseIssueURL(rawURL string) (IssueRef, error) {
	m := issueURLPattern.FindStringSubmatch(rawURL)
	if len(m) != 4 {
		return IssueRef{}, fmt.Errorf("invalid GitHub issue URL: %s", rawURL)
	}
	num, err := strconv.Atoi(m[3])
	if err != nil {
		return IssueRef{}, fmt.Errorf("invalid issue number: %s", m[3])
	}
	return IssueRef{Owner: m[1], Repo: m[2], Number: num}, nil
}

type URLKind int

const (
	KindUnknown URLKind = iota
	KindPR
	KindIssue
)

func ClassifyURL(rawURL string) URLKind {
	switch {
	case prURLPattern.MatchString(rawURL):
		return KindPR
	case issueURLPattern.MatchString(rawURL):
		return KindIssue
	default:
		return KindUnknown
	}
}

func (c *GitHubClient) Fetch(ctx context.Context, ref PRRef) (*PRPayload, error) {
	meta, err := c.fetchMetadata(ctx, ref)
	if err != nil {
		return nil, err
	}
	diff, err := c.fetchDiff(ctx, ref)
	if err != nil {
		return nil, err
	}
	files, err := c.fetchFiles(ctx, ref)
	if err != nil {
		return nil, err
	}
	return &PRPayload{Ref: ref, Metadata: *meta, Diff: diff, Files: files}, nil
}

func (c *GitHubClient) fetchMetadata(ctx context.Context, ref PRRef) (*PRMetadata, error) {
	var raw struct {
		HTMLURL string `json:"html_url"`
		Title   string `json:"title"`
		Body    string `json:"body"`
		User    struct{ Login string `json:"login"` } `json:"user"`
		Base    struct{ Ref string `json:"ref"` } `json:"base"`
		Head    struct{ Ref string `json:"ref"` } `json:"head"`
		State   string `json:"state"`
		ChangedFiles int `json:"changed_files"`
		Additions    int `json:"additions"`
		Deletions    int `json:"deletions"`
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls/%d", ref.Owner, ref.Repo, ref.Number)
	if err := c.getJSON(ctx, url, "application/vnd.github+json", &raw); err != nil {
		return nil, err
	}
	return &PRMetadata{
		URL:          raw.HTMLURL,
		Title:        raw.Title,
		Body:         raw.Body,
		Author:       raw.User.Login,
		BaseBranch:   raw.Base.Ref,
		HeadBranch:   raw.Head.Ref,
		State:        raw.State,
		ChangedFiles: raw.ChangedFiles,
		Additions:    raw.Additions,
		Deletions:    raw.Deletions,
	}, nil
}

func (c *GitHubClient) fetchDiff(ctx context.Context, ref PRRef) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls/%d", ref.Owner, ref.Repo, ref.Number)
	body, err := c.getRaw(ctx, url, "application/vnd.github.v3.diff")
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (c *GitHubClient) fetchFiles(ctx context.Context, ref PRRef) ([]PRFile, error) {
	var files []PRFile
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls/%d/files?per_page=100", ref.Owner, ref.Repo, ref.Number)
	if err := c.getJSON(ctx, url, "application/vnd.github+json", &files); err != nil {
		return nil, err
	}
	return files, nil
}

func (c *GitHubClient) newReq(ctx context.Context, url, accept string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	return req, nil
}

func (c *GitHubClient) do(req *http.Request) ([]byte, error) {
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read github response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("github API %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

func (c *GitHubClient) getJSON(ctx context.Context, url, accept string, out any) error {
	req, err := c.newReq(ctx, url, accept)
	if err != nil {
		return err
	}
	body, err := c.do(req)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

func (c *GitHubClient) getRaw(ctx context.Context, url, accept string) ([]byte, error) {
	req, err := c.newReq(ctx, url, accept)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

type IssueMetadata struct {
	URL    string   `json:"url"`
	Title  string   `json:"title"`
	Body   string   `json:"body"`
	Author string   `json:"author"`
	State  string   `json:"state"`
	Labels []string `json:"labels"`
	Number int      `json:"number"`
}

type IssueComment struct {
	Author string `json:"author"`
	Body   string `json:"body"`
}

type FileContent struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Bytes   int    `json:"bytes"`
}

type IssuePayload struct {
	Ref      IssueRef       `json:"-"`
	Metadata IssueMetadata  `json:"metadata"`
	Comments []IssueComment `json:"comments"`
	Keywords []string       `json:"keywords"`
	Files    []FileContent  `json:"files"`
}

func (c *GitHubClient) FetchIssue(ctx context.Context, ref IssueRef) (*IssuePayload, error) {
	meta, err := c.fetchIssueMetadata(ctx, ref)
	if err != nil {
		return nil, err
	}
	comments, err := c.fetchIssueComments(ctx, ref)
	if err != nil {
		return nil, err
	}
	return &IssuePayload{Ref: ref, Metadata: *meta, Comments: comments}, nil
}

func (c *GitHubClient) fetchIssueMetadata(ctx context.Context, ref IssueRef) (*IssueMetadata, error) {
	var raw struct {
		HTMLURL string `json:"html_url"`
		Title   string `json:"title"`
		Body    string `json:"body"`
		User    struct{ Login string `json:"login"` } `json:"user"`
		State   string `json:"state"`
		Number  int    `json:"number"`
		Labels  []struct{ Name string `json:"name"` } `json:"labels"`
	}
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%d", ref.Owner, ref.Repo, ref.Number)
	if err := c.getJSON(ctx, url, "application/vnd.github+json", &raw); err != nil {
		return nil, err
	}
	labels := make([]string, 0, len(raw.Labels))
	for _, l := range raw.Labels {
		labels = append(labels, l.Name)
	}
	return &IssueMetadata{
		URL:    raw.HTMLURL,
		Title:  raw.Title,
		Body:   raw.Body,
		Author: raw.User.Login,
		State:  raw.State,
		Number: raw.Number,
		Labels: labels,
	}, nil
}

func (c *GitHubClient) fetchIssueComments(ctx context.Context, ref IssueRef) ([]IssueComment, error) {
	var raw []struct {
		User struct{ Login string `json:"login"` } `json:"user"`
		Body string `json:"body"`
	}
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%d/comments?per_page=30", ref.Owner, ref.Repo, ref.Number)
	if err := c.getJSON(ctx, url, "application/vnd.github+json", &raw); err != nil {
		return nil, err
	}
	out := make([]IssueComment, 0, len(raw))
	for _, r := range raw {
		out = append(out, IssueComment{Author: r.User.Login, Body: r.Body})
	}
	return out, nil
}

type CodeSearchResult struct {
	Path       string `json:"path"`
	Repository string `json:"repository"`
	Score      float64 `json:"score"`
}

func (c *GitHubClient) SearchCode(ctx context.Context, ref IssueRef, keywords []string, limit int) ([]CodeSearchResult, error) {
	if len(keywords) == 0 {
		return nil, nil
	}
	terms := make([]string, 0, len(keywords))
	for _, k := range keywords {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		if strings.ContainsAny(k, " \t") {
			terms = append(terms, fmt.Sprintf(`"%s"`, k))
		} else {
			terms = append(terms, k)
		}
	}
	q := strings.Join(terms, " ") + fmt.Sprintf(" repo:%s/%s", ref.Owner, ref.Repo)

	var raw struct {
		Items []struct {
			Path       string  `json:"path"`
			Score      float64 `json:"score"`
			Repository struct{ FullName string `json:"full_name"` } `json:"repository"`
		} `json:"items"`
	}
	endpoint := fmt.Sprintf("https://api.github.com/search/code?q=%s&per_page=%d", url.QueryEscape(q), limit)
	if err := c.getJSON(ctx, endpoint, "application/vnd.github+json", &raw); err != nil {
		return nil, err
	}
	out := make([]CodeSearchResult, 0, len(raw.Items))
	for _, it := range raw.Items {
		out = append(out, CodeSearchResult{Path: it.Path, Repository: it.Repository.FullName, Score: it.Score})
	}
	return out, nil
}

func (c *GitHubClient) FetchFile(ctx context.Context, owner, repo, path string, maxBytes int) (*FileContent, error) {
	var raw struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
		Size     int    `json:"size"`
	}
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s", owner, repo, path)
	if err := c.getJSON(ctx, endpoint, "application/vnd.github+json", &raw); err != nil {
		return nil, err
	}
	if raw.Encoding != "base64" {
		return nil, fmt.Errorf("unsupported file encoding: %s", raw.Encoding)
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(raw.Content, "\n", ""))
	if err != nil {
		return nil, fmt.Errorf("decode file %s: %w", path, err)
	}
	if maxBytes > 0 && len(decoded) > maxBytes {
		decoded = append(decoded[:maxBytes], []byte("\n…[file truncated]")...)
	}
	return &FileContent{Path: path, Content: string(decoded), Bytes: raw.Size}, nil
}
