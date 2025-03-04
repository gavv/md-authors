package backend

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/Jeffail/gabs/v2"
	"github.com/gofri/go-github-pagination/githubpagination"
	"github.com/gofri/go-github-ratelimit/github_ratelimit"
	"github.com/texttheater/golang-levenshtein/levenshtein"

	"github.com/gavv/md-authors/src/cache"
	"github.com/gavv/md-authors/src/defs"
	"github.com/gavv/md-authors/src/logs"
	"github.com/gavv/md-authors/src/match"
)

var (
	noreplyRx = regexp.MustCompile(`^([0-9]+\+)?([^@]+)@users\.noreply\.github\.com$`)
	spaceRx   = regexp.MustCompile(`\s`)
)

func githubPopulate(author defs.Author, conf defs.Config) (defs.Author, error) {
	project := conf.Project
	if project == "" {
		project = githubProject()
	}

	gitName := author.Name
	gitEmail := author.Email

	if m := noreplyRx.FindStringSubmatch(author.Email); m != nil {
		author.Login = m[2]
		author.Email = ""
	} else {
		author.Login = ""
		author.Email = gitEmail
	}

	if author.Login == "" {
		author.Login = githubLogin(project, gitName, gitEmail)
	}

	if author.Login != "" {
		author.Profile = fmt.Sprintf("https://github.com/%s", author.Login)
	}

	if author.Email == "" && author.Login != "" {
		author.Email = githubEmail(project, author.Login, gitName)
	}

	if author.Name == "" || !spaceRx.MatchString(author.Name) {
		author.Name = githubName(project, author.Login, author.Name)
	}

	return author, nil
}

func githubLogin(project, gitName, gitEmail string) (login string) {
	if gitName == "" || gitEmail == "" {
		return ""
	}

	defer func() {
		cache.DiskStore([]string{"github", "n2l", gitName}, login)
		cache.DiskStore([]string{"github", "e2l", gitEmail}, login)
	}()

	var found bool

	login, found = cache.DiskLoad([]string{"github", "n2l", gitName})
	if found {
		return login
	}

	login, found = cache.DiskLoad([]string{"github", "e2l", gitEmail})
	if found {
		return login
	}

	var candidates []string

	if users := githubRequest("/search/users", false, "q", gitEmail+" in:email"); users != nil {
		for _, user := range users.Path("items").Children() {
			if userLogin := user.Path("login").Data().(string); userLogin != "" {
				if !slices.Contains(candidates, userLogin) {
					candidates = append(candidates, userLogin)
				}
			}
		}
	}

	// shortcut: often it's enough to check just first candidate
	for _, userLogin := range candidates {
		userEmail := githubEmail(project, userLogin, gitName)
		if strings.EqualFold(userEmail, gitEmail) {
			return userLogin
		}
	}

	if users := githubRequest("/search/users", false, "q", gitName+" in:name"); users != nil {
		for _, user := range users.Path("items").Children() {
			if userLogin := user.Path("login").Data().(string); userLogin != "" {
				if !slices.Contains(candidates, userLogin) {
					candidates = append(candidates, userLogin)
					// if search by name gives lots of results, ignore them,
					// search by commits will do the job
					if len(candidates) > 3 {
						break
					}
				}
			}
		}
	}

	if !strings.Contains(gitName, " ") {
		if !slices.Contains(candidates, gitName) {
			candidates = append(candidates, gitName)
		}
	}

	// shortcut: often it's enough to check just search results, without
	// loading contributors and commits
	for _, userLogin := range candidates {
		userEmail := githubEmail(project, userLogin, gitName)
		if strings.EqualFold(userEmail, gitEmail) {
			return userLogin
		}
	}

	// add all contributors to candidate list
	if project != "" {
		contributors := githubContributors(project)

		// shortcut: this sorting doesn't affect end result, but it allows to check
		// more probable candidates first and hence improves performance
		sort.Slice(contributors, func(i, j int) bool {
			iLogin := contributors[i]
			jLogin := contributors[j]

			iDist := levenshtein.DistanceForStrings([]rune(iLogin), []rune(gitName),
				levenshtein.DefaultOptions)
			jDist := levenshtein.DistanceForStrings([]rune(jLogin), []rune(gitName),
				levenshtein.DefaultOptions)

			return iDist < jDist
		})

		for _, userLogin := range contributors {
			if !slices.Contains(candidates, userLogin) {
				candidates = append(candidates, userLogin)
			}
		}
	}

	// match candidates by email
	for _, userLogin := range candidates {
		userEmail := githubEmail(project, userLogin, gitName)
		if strings.EqualFold(userEmail, gitEmail) {
			return userLogin
		}

		if project != "" {
			for _, commit := range githubContribCommits(project, userLogin) {
				if strings.EqualFold(commit.Email, gitEmail) {
					return userLogin
				}
			}
		}
	}

	return ""
}

func githubName(project, login, gitName string) (name string) {
	if login == "" {
		return ""
	}

	defer func() {
		cache.DiskStore([]string{"github", "l2n", login}, name)
	}()

	var found bool

	name, found = cache.DiskLoad([]string{"github", "l2n", login})
	if found {
		return name
	}

	commitName := ""

	if project != "" {
		author := githubCommitAuthor(githubContribCommits(project, login), "")
		if author != nil {
			commitName = author.Name
		}

		if commitName == "" {
			author = githubCommitAuthor(githubPullreqCommits(project, login), "")
			if author != nil {
				commitName = author.Name
			}
		}
	}

	if commitName == "" {
		author := githubCommitAuthor(githubEventCommits(login), "")
		if author != nil {
			commitName = author.Email
		}
	}

	profileName := ""

	profile := githubRequest("/users/"+login, false)
	if profile != nil {
		profileName, _ = profile.Path("name").Data().(string)
	}

	switch {
	case spaceRx.MatchString(commitName) ||
		(commitName != "" && !spaceRx.MatchString(profileName) && !spaceRx.MatchString(gitName)):
		return commitName

	case spaceRx.MatchString(profileName) ||
		(profileName != "" && !spaceRx.MatchString(gitName)):
		return profileName

	default:
		return gitName
	}
}

func githubEmail(project, login, nameHint string) (email string) {
	if login == "" || nameHint == "" {
		return ""
	}

	defer func() {
		cache.DiskStore([]string{"github", "l2e", login}, email)
	}()

	var found bool

	email, found = cache.DiskLoad([]string{"github", "l2e", login})
	if found {
		return email
	}

	if project != "" {
		author := githubCommitAuthor(githubContribCommits(project, login), nameHint)
		if author != nil {
			return author.Email
		}

		author = githubCommitAuthor(githubPullreqCommits(project, login), nameHint)
		if author != nil {
			return author.Email
		}
	}

	author := githubCommitAuthor(githubEventCommits(login), nameHint)
	if author != nil {
		return author.Email
	}

	return ""
}

type githubCommit struct {
	Email string `json:"e"`
	Name  string `json:"n"`
}

func githubCommitAuthor(commits []githubCommit, nameHint string) *githubCommit {
	if commits == nil {
		return nil
	}

	if len(commits) == 1 && !noreplyRx.MatchString(commits[0].Email) {
		return &commits[0]
	}

	if nameHint != "" {
		for _, commit := range commits {
			if !noreplyRx.MatchString(commit.Email) && match.LooksAlike(commit.Name, nameHint) {
				return &commit
			}
		}
	}

	return nil
}

func githubPullreqCommits(project, login string) (commits []githubCommit) {
	if project == "" || login == "" {
		return nil
	}

	defer func() {
		cache.DiskStore([]string{"github", "pc", project, login},
			cache.Serialize(commits))
	}()

	data, found := cache.DiskLoad([]string{"github", "pc", project, login})
	if found {
		cache.Deserialize(data, &commits)
		return
	}

	ghPullreqs := githubRequest("/search/issues", false, "q",
		fmt.Sprintf("type:pr repo:%s author:%s", project, login))

	if ghPullreqs != nil {
		for _, prChild := range ghPullreqs.Path("items").Children() {
			prNo, _ := prChild.Path("number").Data().(float64)
			if prNo == 0 {
				continue
			}

			ghState := githubRequest(
				fmt.Sprintf("/repos/%s/pulls/%d", project, int(prNo)), false)

			merged, _ := ghState.Path("merged").Data().(bool)
			if !merged {
				continue
			}

			ghCommits := githubRequest(
				fmt.Sprintf("/repos/%s/pulls/%d/commits", project, int(prNo)), true)

			if ghCommits != nil {
				for _, commitChild := range ghCommits.Children() {
					var commit githubCommit

					commit.Email, _ = commitChild.Path("commit.author.email").Data().(string)
					commit.Name, _ = commitChild.Path("commit.author.name").Data().(string)

					if commit.Email == "" || commit.Name == "" {
						continue
					}
					if slices.Contains(commits, commit) {
						continue
					}

					commits = append(commits, commit)
				}
			}
		}
	}

	return
}

func githubContribCommits(project, login string) (commits []githubCommit) {
	if project == "" || login == "" {
		return nil
	}

	defer func() {
		cache.DiskStore([]string{"github", "cc", project, login},
			cache.Serialize(commits))
	}()

	data, found := cache.DiskLoad([]string{"github", "cc", project, login})
	if found {
		cache.Deserialize(data, &commits)
		return
	}

	ghCommits := githubRequest("/repos/"+project+"/commits", false, "author", login)

	if ghCommits != nil {
		for _, child := range ghCommits.Children() {
			var commit githubCommit

			commit.Email, _ = child.Path("commit.author.email").Data().(string)
			commit.Name, _ = child.Path("commit.author.name").Data().(string)

			if commit.Email == "" || commit.Name == "" {
				continue
			}
			if slices.Contains(commits, commit) {
				continue
			}

			commits = append(commits, commit)
		}
	}

	return
}

func githubEventCommits(login string) (commits []githubCommit) {
	if login == "" {
		return nil
	}

	defer func() {
		cache.DiskStore([]string{"github", "ec", login},
			cache.Serialize(commits))
	}()

	data, found := cache.DiskLoad([]string{"github", "ec", login})
	if found {
		cache.Deserialize(data, &commits)
		return
	}

	ghEvents := githubRequest("/users/"+login+"/events/public", false)

	if ghEvents != nil {
		for _, child := range ghEvents.Path("payload.commits").Children() {
			var commit githubCommit

			commit.Email, _ = child.Path("author.email").Data().(string)
			commit.Name, _ = child.Path("author.name").Data().(string)

			if commit.Email == "" || commit.Name == "" {
				continue
			}
			if slices.Contains(commits, commit) {
				continue
			}

			commits = append(commits, commit)
		}
	}

	return
}

func githubContributors(project string) (contributors []string) {
	if project == "" {
		return nil
	}

	defer func() {
		cache.MemStore([]string{"github", "contrib", project},
			cache.Serialize(contributors))
	}()

	data, found := cache.MemLoad([]string{"github", "contrib", project})
	if found {
		cache.Deserialize(data, &contributors)
		return
	}

	ghContribs := githubRequest("/repos/"+project+"/contributors", true)

	if ghContribs != nil {
		for _, child := range ghContribs.Children() {
			if login, _ := child.Path("login").Data().(string); login != "" {
				contributors = append(contributors, login)
			}
		}
	}

	return contributors
}

var githubClient = &http.Client{
	Transport: func() http.RoundTripper {
		rateLimiter, _ := github_ratelimit.NewRateLimitWaiter(nil)
		return rateLimiter
	}(),
}

var githubPaginatingClient = &http.Client{
	Transport: func() http.RoundTripper {
		rateLimiter, _ := github_ratelimit.NewRateLimitWaiter(nil)
		return githubpagination.New(rateLimiter)
	}(),
}

func githubRequest(endpoint string, paginate bool, queryArgs ...string) *gabs.Container {
	req, _ := http.NewRequest("GET", "https://api.github.com"+endpoint, nil)

	req.Header.Add("accept", "application/vnd.github.v3+json")

	q := req.URL.Query()
	for i := 0; i < len(queryArgs); i += 2 {
		q.Add(queryArgs[i], queryArgs[i+1])
	}
	req.URL.RawQuery = q.Encode()

	// If 'gh' is eavailable, use it instead of direct HTTP request.
	// While direct request will also work as we only access publically available
	// data, if 'gh' tool is authenticated, it has higher rate limits.
	if _, err := exec.LookPath("gh"); err == nil {
		cmdArgs := []string{"gh", "api"}
		if paginate {
			cmdArgs = append(cmdArgs, "--paginate")
		}
		cmdArgs = append(cmdArgs, req.URL.String())

		logs.Debugf("running: %s", strings.Join(cmdArgs, " "))

		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)

		var out bytes.Buffer
		cmd.Stdout = &out
		err := cmd.Run()

		if err == nil {
			js, err := gabs.ParseJSON(out.Bytes())
			if err == nil {
				return js
			}
		}
	}

	logs.Debugf("sending: %s %s", req.Method, req.URL.String())

	client := githubClient
	if paginate {
		client = githubPaginatingClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	js, err := gabs.ParseJSON(body)
	if err != nil {
		return nil
	}

	return js
}

func githubProject() string {
	cmdArgs := []string{"git", "remote", "-v"}

	logs.Debugf("running: %s", strings.Join(cmdArgs, " "))

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	remoteMap := make(map[string]string)
	remoteList := make([]string, 0)

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "(fetch)") {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}

			remote := fields[0]
			uri := fields[1]

			switch {
			case strings.HasPrefix(uri, "git@github.com:"):
				uri = strings.TrimPrefix(uri, "git@github.com:")
				uri = strings.TrimSuffix(uri, ".git")

			case strings.HasPrefix(uri, "https://github.com/"):
				uri = strings.TrimPrefix(uri, "https://github.com/")
				uri = strings.TrimSuffix(uri, ".git")

			default:
				continue
			}

			if strings.Count(uri, "/") != 1 {
				continue
			}

			remoteMap[remote] = uri
			remoteList = append(remoteList, uri)
		}
	}

	project, _ := remoteMap["origin"]
	if project == "" && len(remoteList) > 0 {
		project = remoteList[0]
	}

	if project != "" {
		logs.Debugf("auto-detected github project %q", project)
	}

	return project
}
