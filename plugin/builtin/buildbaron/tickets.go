package buildbaron

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"text/template"

	"github.com/evergreen-ci/evergreen/model/host"
	"github.com/evergreen-ci/evergreen/model/task"
	"github.com/evergreen-ci/evergreen/plugin"
	"github.com/evergreen-ci/evergreen/thirdparty"
	"github.com/evergreen-ci/evergreen/util"
	"github.com/mongodb/grip"
)

const (
	FailingTasksField     = "customfield_12950"
	FailingVariantField   = "customfield_14277"
	EvergreenProjectField = "customfield_14278"
	UIRoot                = "https://evergreen.mongodb.com"
)

const DescriptionTemplateString = `
h2. [{{.Task.DisplayName}} failed on {{.Task.BuildVariant}}|` + UIRoot + `/task/{{.Task.Id}}/{{.Task.Execution}}]

Host: [{{.Host.Host}}|` + UIRoot + `/host/{{.Host.Id}}]
Project: [{{.Task.Project}}|` + UIRoot + `/waterfall/{{.Task.Project}}]

{{range .Tests}}*{{.Name}}* - [Logs|{{.URL}}] | [History|{{.HistoryURL}}]

{{end}}


~BF Ticket Generated by [~{{.UserId}}]~
`

var DescriptionTemplate = template.Must(template.New("Desc").Parse(DescriptionTemplateString))

// jiraTestFailure contains the required fields for generating a failure report.
type jiraTestFailure struct {
	Name       string
	URL        string
	HistoryURL string
}

// fileTicket creates a JIRA ticket for a task with the given test failures.
func (bbp *BuildBaronPlugin) fileTicket(w http.ResponseWriter, r *http.Request) {
	var input struct {
		TaskId  string   `json:"task"`
		TestIds []string `json:"tests"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		util.WriteJSON(w, http.StatusInternalServerError, err.Error())
	}

	// grab the task and user info to fill out the ticket
	u := plugin.GetUser(r)
	if u == nil {
		util.WriteJSON(w, http.StatusUnauthorized, "must be logged in to file a ticket")
		return
	}
	// Find information about the task
	t, err := task.FindOne(task.ById(input.TaskId))
	if err != nil {
		util.WriteJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	if t == nil {
		util.WriteJSON(w, http.StatusNotFound, fmt.Sprintf("task not found for id %v", input.TaskId))
		return
	}
	// Find the host the task ran on
	h, err := host.FindOne(host.ById(t.HostId))
	if err != nil {
		util.WriteJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	if h == nil {
		util.WriteJSON(w, http.StatusInternalServerError, fmt.Sprintf("host not found for task id %v with host id: %v", input.TaskId, t.HostId))
		return
	}

	// build a list of all failed tests to include
	testIds := map[string]bool{}
	for _, testId := range input.TestIds {
		testIds[testId] = true
	}
	tests := []jiraTestFailure{}
	for _, test := range t.TestResults {
		if testIds[test.TestFile] {
			tests = append(tests, jiraTestFailure{
				Name:       cleanTestName(test.TestFile),
				URL:        test.URL,
				HistoryURL: historyURL(t, cleanTestName(test.TestFile)),
			})
		}
	}

	//lay out the JIRA API request
	request := map[string]interface{}{}
	request["project"] = map[string]string{"key": bbp.opts.Projects[t.Project].TicketCreateProject}
	request["summary"] = getSummary(t.DisplayName, tests)
	request[FailingTasksField] = []string{t.DisplayName}
	request[FailingVariantField] = []string{t.BuildVariant}
	request[EvergreenProjectField] = []string{t.Project}
	request["issuetype"] = map[string]string{"name": "Build Failure"}
	request["assignee"] = map[string]string{"name": u.Id}
	request["reporter"] = map[string]string{"name": u.Id}
	request["description"], err = getDescription(t, h, u.Id, tests)

	if err != nil {
		util.WriteJSON(
			w, http.StatusBadRequest, fmt.Sprintf("error creating description: %v", err))
		return
	}

	grip.Infoln("Creating JIRA ticket for user", u.Id)

	jiraHandler := thirdparty.NewJiraHandler(
		bbp.opts.Host,
		bbp.opts.Username,
		bbp.opts.Password,
	)
	result, err := jiraHandler.CreateTicket(request)
	if err != nil {
		msg := fmt.Sprintf("error creating JIRA ticket: %v", err)
		grip.Error(msg)
		util.WriteJSON(w, http.StatusBadRequest, msg)
		return
	}
	grip.Infof("Ticket %s successfully created", result.Key)
	util.WriteJSON(w, http.StatusOK, result)
}

func cleanTestName(path string) string {
	if unixIdx := strings.LastIndex(path, "/"); unixIdx != -1 {
		// if the path ends in a slash, remove it and try again
		if unixIdx == len(path)-1 {
			return cleanTestName(path[:len(path)-1])
		}
		return path[unixIdx+1:]
	}
	if windowsIdx := strings.LastIndex(path, `\`); windowsIdx != -1 {
		// if the path ends in a slash, remove it and try again
		if windowsIdx == len(path)-1 {
			return cleanTestName(path[:len(path)-1])
		}
		return path[windowsIdx+1:]
	}
	return path
}

func historyURL(t *task.Task, testName string) string {
	return fmt.Sprintf("%v/task_history/%v/%v#%v=fail",
		UIRoot, t.Project, t.DisplayName, testName)
}

func getSummary(taskName string, tests []jiraTestFailure) string {
	switch {
	case len(tests) == 0:
		// this is likely a compile failure
		return fmt.Sprintf("%v failure", taskName)
	case len(tests) > 4:
		// if there are many failures, just squish the summary
		return fmt.Sprintf("%v failures", taskName)
	default:
		names := []string{}
		for _, t := range tests {
			names = append(names, t.Name)
		}
		return strings.Join(names, ", ")
	}
}

func getDescription(t *task.Task, h *host.Host, userId string, tests []jiraTestFailure) (string, error) {
	args := struct {
		Task   *task.Task
		Host   *host.Host
		UserId string
		Tests  []jiraTestFailure
	}{t, h, userId, tests}
	buf := &bytes.Buffer{}
	if err := DescriptionTemplate.Execute(buf, args); err != nil {
		return "", err
	}
	return buf.String(), nil
}
