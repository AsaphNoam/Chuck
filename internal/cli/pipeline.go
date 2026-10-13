package cli

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/AsaphNoam/Chuck/internal/config"
	"github.com/AsaphNoam/Chuck/internal/pipeline"
)

func newPipelineCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "pipeline", Short: "Manage pipeline templates and durable runs"}
	templates := &cobra.Command{Use: "template", Short: "Inspect and validate pipeline templates"}
	listTemplates := &cobra.Command{Use: "list", Short: "List pipeline templates", Args: cobra.NoArgs, RunE: func(c *cobra.Command, _ []string) error {
		if pipelineOperatorMode(c) {
			return operatorRun(c, http.MethodGet, "/api/pipelines", nil, nil)
		}
		return runPipelineRequest(http.MethodGet, "/api/pipelines", nil)
	}}
	addOperatorFlags(listTemplates)
	templates.AddCommand(listTemplates)
	var templateID, templateFile string
	validate := &cobra.Command{Use: "validate", Short: "Validate a version-2 template JSON file", Args: cobra.NoArgs, RunE: func(c *cobra.Command, _ []string) error {
		var body map[string]any
		var err error
		if c.Flags().Changed("request-file") {
			if c.Flags().Changed("id") || c.Flags().Changed("file") {
				return operatorInputError(fmt.Errorf("use either --request-file or the legacy --id and --file inputs"))
			}
			body, err = operatorRequestBody(c, nil)
			if err == nil {
				err = requirePipelineFields(body, "id", "template")
			}
		} else {
			if templateID == "" || templateFile == "" {
				return operatorInputError(fmt.Errorf("--id and --file are required"))
			}
			data, readErr := operatorReadInput(c, templateFile)
			if readErr != nil {
				return operatorInputError(readErr)
			}
			var template pipeline.Template
			if err = json.Unmarshal(data, &template); err != nil {
				return fmt.Errorf("decode template: %w", err)
			}
			body = map[string]any{"id": templateID, "template": template}
		}
		if err != nil {
			return err
		}
		if pipelineOperatorMode(c) {
			return operatorRun(c, http.MethodPost, "/api/pipelines/validate", body, nil)
		}
		return runPipelineRequest(http.MethodPost, "/api/pipelines/validate", body)
	}}
	validate.Flags().StringVar(&templateID, "id", "", "template id")
	validate.Flags().StringVar(&templateFile, "file", "", "template JSON file")
	addOperatorFlags(validate)
	templates.AddCommand(validate)
	// Full template CRUD uses the existing pipeline routes and accepts the
	// complete nested template shape through --request-file.
	templateShow := pipelineOperatorPath("show <template_id>", http.MethodGet, "/api/pipelines", false)
	documentPipelineCommand(templateShow, "Show a saved template.", "chuck pipeline template show quality")
	templates.AddCommand(templateShow)
	templateCreate := &cobra.Command{Use: "create", Short: "Save a validated pipeline template", Args: cobra.NoArgs, Long: "Save a version-2 template using the existing template request shape {\"id\":\"quality\",\"template\":{...}}. Example: chuck pipeline template create --request-file template.json", RunE: func(c *cobra.Command, _ []string) error {
		if err := requirePipelineRequestFile(c); err != nil {
			return err
		}
		body, err := operatorRequestBody(c, nil)
		if err != nil {
			return err
		}
		if err := requirePipelineFields(body, "id", "template"); err != nil {
			return err
		}
		return operatorRun(c, http.MethodPost, "/api/pipelines", body, nil)
	}}
	addOperatorFlags(templateCreate)
	markPipelineOperator(templateCreate)
	templates.AddCommand(templateCreate)
	templateUpdate := &cobra.Command{Use: "update <template_id>", Short: "Update a saved pipeline template", Args: cobra.ExactArgs(1), Long: "Send the complete version-2 template object in --request-file; the id is the path segment. Example: chuck pipeline template update quality --request-file template.json"}
	addOperatorFlags(templateUpdate)
	markPipelineOperator(templateUpdate)
	templateUpdate.RunE = func(c *cobra.Command, args []string) error {
		if err := requirePipelineRequestFile(c); err != nil {
			return err
		}
		p, err := idPath("/api/pipelines", args)
		if err != nil {
			return err
		}
		b, err := operatorRequestBody(c, nil)
		if err != nil {
			return err
		}
		if _, exists := b["id"]; exists {
			return operatorInputError(fmt.Errorf("template update takes the template object only; its id is the path argument"))
		}
		return operatorRun(c, http.MethodPut, p, b, nil)
	}
	templates.AddCommand(templateUpdate)
	templateDelete := pipelineOperatorPath("delete <template_id>", http.MethodDelete, "/api/pipelines", true)
	documentPipelineCommand(templateDelete, "Delete a saved template after acknowledging the removal.", "chuck pipeline template delete quality --yes")
	templates.AddCommand(templateDelete)
	cmd.AddCommand(templates)

	proposals := &cobra.Command{Use: "proposal", Short: "Inspect and remove pipeline proposals"}
	proposalList := pipelineOperatorPath("list", http.MethodGet, "/api/pipeline-proposals", false)
	documentPipelineCommand(proposalList, "List saved pipeline proposals.", "chuck pipeline proposal list --json")
	proposals.AddCommand(proposalList)
	proposalDecline := pipelineOperatorPath("decline <proposal_id>", http.MethodPost, "/api/pipeline-proposals", false, "decline")
	documentPipelineCommand(proposalDecline, "Decline a pipeline proposal.", "chuck pipeline proposal decline proposal_123 --json")
	proposals.AddCommand(proposalDecline)
	proposalDelete := pipelineOperatorPath("delete <proposal_id>", http.MethodDelete, "/api/pipeline-proposals", true, "")
	documentPipelineCommand(proposalDelete, "Delete a proposal after acknowledging the removal.", "chuck pipeline proposal delete proposal_123 --yes")
	proposals.AddCommand(proposalDelete)
	cmd.AddCommand(proposals)

	runs := &cobra.Command{Use: "run", Short: "Start and control pipeline runs"}
	listRuns := pipelineOperatorPath("list", http.MethodGet, "/api/pipeline-runs", false)
	var runLimit, runOffset int
	listRuns.Flags().IntVar(&runLimit, "limit", 100, "maximum number of runs (1-100)")
	listRuns.Flags().IntVar(&runOffset, "offset", 0, "number of runs to skip")
	listRuns.RunE = func(c *cobra.Command, _ []string) error {
		if runLimit < 1 || runLimit > 100 || runOffset < 0 {
			return operatorInputError(fmt.Errorf("--limit must be 1-100 and --offset must be nonnegative"))
		}
		return operatorRun(c, http.MethodGet, fmt.Sprintf("/api/pipeline-runs?limit=%d&offset=%d", runLimit, runOffset), nil, nil)
	}
	documentPipelineCommand(listRuns, "List a bounded page of pipeline runs using --limit and --offset.", "chuck pipeline run list --limit 20 --offset 40")
	runs.AddCommand(listRuns)
	var startTemplate, startName, startProject, startGoal, startRequestID string
	var orchestratorBackend, orchestratorModel, orchestratorEffort string
	var startInputs, startStages, roomParticipants, roomJudges []string
	var orchestratorFast, acknowledge bool
	start := &cobra.Command{Use: "start", Short: "Start a configured pipeline run", RunE: func(cmd *cobra.Command, _ []string) error {
		if cmd.Flags().Changed("request-file") {
			body, err := pipelineStartRequestFile(cmd)
			if err != nil {
				return err
			}
			if err := ensurePipelineRequestID(body); err != nil {
				return err
			}
			if err := requirePipelineFields(body, "template_id", "project", "goal"); err != nil {
				return err
			}
			return operatorRun(cmd, http.MethodPost, "/api/pipeline-runs", body, nil)
		}
		if startTemplate == "" || startProject == "" || startGoal == "" {
			return fmt.Errorf("template, project and goal are required")
		}
		inputs, err := parseKeyValues(startInputs)
		if err != nil {
			return err
		}
		assignments, err := parseStageAssignments(startStages)
		if err != nil {
			return err
		}
		if _, ok := assignments["standing"]; ok {
			return operatorInputError(fmt.Errorf("use --backend and --model for the standing assignment; --stage cannot replace it"))
		}
		if startRequestID == "" {
			startRequestID = newPipelineRequestID()
		}
		// Keep the legacy assignments projection populated with the standing
		// runtime while the API transitions to the explicit v2 shape. Dedicated
		// stage coordinators are sent separately and never become stage owners.
		standing := pipeline.RuntimeAssignment{Backend: orchestratorBackend, Model: orchestratorModel, Effort: orchestratorEffort, Fast: orchestratorFast}
		assignments["standing"] = standing
		rooms, err := parseRoomAssignments(roomParticipants, roomJudges)
		if err != nil {
			return err
		}
		body := struct {
			pipeline.StartRequest
			Acknowledge bool `json:"acknowledge_shared_workspace"`
		}{StartRequest: pipeline.StartRequest{RequestID: startRequestID, TemplateID: startTemplate, DisplayName: startName, Project: startProject, Goal: startGoal, Inputs: inputs, Assignments: assignments, Orchestrator: standing, DedicatedAssignments: dedicatedAssignments(assignments), ThinkTankAssignments: rooms}, Acknowledge: acknowledge}
		if pipelineOperatorMode(cmd) {
			return operatorRun(cmd, http.MethodPost, "/api/pipeline-runs", body, nil)
		}
		return runPipelineRequest(http.MethodPost, "/api/pipeline-runs", body)
	}}
	start.Flags().StringVar(&startTemplate, "template", "", "saved template id")
	start.Flags().StringVar(&startName, "name", "", "run display name")
	start.Flags().StringVar(&startProject, "project", "", "project id")
	start.Flags().StringVar(&startGoal, "goal", "", "run goal")
	start.Flags().StringVar(&orchestratorBackend, "backend", "", "standing orchestrator backend")
	start.Flags().StringVar(&orchestratorModel, "model", "", "standing orchestrator model")
	start.Flags().StringVar(&orchestratorEffort, "effort", "", "standing orchestrator reasoning effort")
	start.Flags().BoolVar(&orchestratorFast, "fast", false, "use fast mode for the standing orchestrator")
	start.Flags().StringVar(&startRequestID, "request-id", "", "idempotency request id")
	start.Flags().StringSliceVar(&startInputs, "input", nil, "named input as name=value (repeatable)")
	start.Flags().StringSliceVar(&startStages, "stage", nil, "runtime as stage=backend/model (repeatable)")
	start.Flags().StringSliceVar(&roomParticipants, "room-participant", nil, "Think Tank participant runtime as stage.participant=backend/model (repeatable)")
	start.Flags().StringSliceVar(&roomJudges, "room-judge", nil, "Think Tank judge runtime as stage=backend/model (repeatable)")
	start.Flags().BoolVar(&acknowledge, "ack-shared-workspace", false, "acknowledge active work sharing the project directory")
	addOperatorFlags(start)
	start.Long = "Start a configured template with --template, --project, and --goal. Repeat --input name=value for template inputs, --stage stage=backend/model for stage assignments, and use --request-file for the complete request including effort/fast or Think Tank assignments. Example: chuck pipeline run start --template quality --project demo --goal 'Ship it' --backend codex --model gpt"
	// --backend/--model are required by the server only when an ordinary stage
	// needs the standing owner; a Think Tank-only run has none (TS-09.R52).
	runs.AddCommand(start)
	showRun := &cobra.Command{Use: "show <run_id>", Short: "Show a pipeline run", Args: cobra.ExactArgs(1), RunE: func(c *cobra.Command, args []string) error {
		path, err := idPath("/api/pipeline-runs", args)
		if err != nil {
			return err
		}
		if pipelineOperatorMode(c) {
			return operatorRun(c, http.MethodGet, path, nil, nil)
		}
		return runPipelineRequest(http.MethodGet, path, nil)
	}}
	addOperatorFlags(showRun)
	documentPipelineCommand(showRun, "Read the current run state and revision. Example: chuck pipeline run show run_123", "chuck pipeline run show run_123")
	runs.AddCommand(showRun)
	var revision int64
	var continuation string
	continueCmd := &cobra.Command{Use: "continue <run_id>", Short: "Continue a blocked pipeline run", Args: cobra.ExactArgs(1), RunE: func(c *cobra.Command, args []string) error {
		path, err := idPath("/api/pipeline-runs", args)
		if err != nil {
			return err
		}
		path += "/continue"
		body := map[string]any{"revision": revision, "input": continuation}
		if c.Flags().Changed("request-file") {
			body, err = pipelineControlRequestFile(c, map[string]any{"revision": revision, "input": continuation})
			if err != nil {
				return err
			}
		}
		if err := requirePipelineRevision(body); err != nil {
			return err
		}
		if pipelineOperatorMode(c) {
			return operatorRun(c, http.MethodPost, path, body, nil)
		}
		return runPipelineRequest(http.MethodPost, path, body)
	}}
	continueCmd.Flags().Int64Var(&revision, "revision", 0, "current run revision")
	continueCmd.Flags().StringVar(&continuation, "input", "", "new input for a blocked stage")
	addOperatorFlags(continueCmd)
	documentPipelineCommand(continueCmd, "Continue with a caller-supplied positive current revision; stale revisions are refused. --input provides continuation text, or --request-file supplies {revision,input}.", "chuck pipeline run continue run_123 --revision 4 --input 'more context'")
	runs.AddCommand(continueCmd)
	for _, action := range []string{"retry", "stop"} {
		action := action
		var actionRevision int64
		actionCmd := &cobra.Command{Use: action + " <run_id>", Short: strings.ToUpper(action[:1]) + action[1:] + " a pipeline run", Args: cobra.ExactArgs(1), RunE: func(c *cobra.Command, args []string) error {
			path, err := idPath("/api/pipeline-runs", args)
			if err != nil {
				return err
			}
			path += "/" + action
			body := map[string]any{"revision": actionRevision}
			if c.Flags().Changed("request-file") {
				body, err = pipelineControlRequestFile(c, body)
				if err != nil {
					return err
				}
			}
			if err := requirePipelineRevision(body); err != nil {
				return err
			}
			if pipelineOperatorMode(c) {
				return operatorRun(c, http.MethodPost, path, body, nil)
			}
			return runPipelineRequest(http.MethodPost, path, body)
		}}
		actionCmd.Flags().Int64Var(&actionRevision, "revision", 0, "current run revision")
		addOperatorFlags(actionCmd)
		documentPipelineCommand(actionCmd, "Supply the current positive revision; a stale revision is refused. --request-file may supply {revision}.", "chuck pipeline run "+action+" run_123 --revision 4")
		runs.AddCommand(actionCmd)
	}
	for _, operation := range []string{"replace", "repair-cleanup"} {
		operation := operation
		c := pipelineOperatorBodyPath(operation+" <run_id>", http.MethodPost, "/"+operation)
		c.RunE = func(cmd *cobra.Command, args []string) error {
			if err := requirePipelineRequestFile(cmd); err != nil {
				return err
			}
			body, err := operatorRequestBody(cmd, nil)
			if err != nil {
				return err
			}
			if err := requirePipelineRevision(body); err != nil {
				return err
			}
			if operation == "replace" {
				if err := requirePipelineRuntime(body, "orchestrator"); err != nil {
					return err
				}
			}
			p, err := idPath("/api/pipeline-runs", args)
			if err != nil {
				return err
			}
			return operatorRun(cmd, http.MethodPost, p+"/"+operation, body, nil)
		}
		requestShape := "{revision}"
		if operation == "replace" {
			requestShape = "{revision,orchestrator:{backend,model,effort,fast}}"
		}
		documentPipelineCommand(c, "Apply this operation using the supplied current revision and --request-file shape "+requestShape+".", "chuck pipeline run "+operation+" run_123 --request-file request.json")
		runs.AddCommand(c)
	}
	deleteRun := pipelineOperatorBodyPath("delete <run_id>", http.MethodDelete, "", true)
	documentPipelineCommand(deleteRun, "Delete a pipeline run after acknowledging the removal.", "chuck pipeline run delete run_123 --yes")
	runs.AddCommand(deleteRun)
	cmd.AddCommand(runs)
	return cmd
}

func runPipelineRequest(method, path string, body any) error {
	store, err := config.New()
	if err != nil {
		return err
	}
	status, data, err := pipelineAPIRequest(dashboardPort(store), method, path, body)
	if err != nil {
		return fmt.Errorf("could not reach dashboard; run `chuck dashboard start` first: %w", err)
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("pipeline request failed (%d): %s", status, strings.TrimSpace(string(data)))
	}
	if len(bytes.TrimSpace(data)) > 0 {
		var pretty bytes.Buffer
		if json.Indent(&pretty, data, "", "  ") == nil {
			fmt.Println(pretty.String())
		} else {
			fmt.Println(string(data))
		}
	}
	return nil
}

func pipelineAPIRequest(port int, method, path string, body any) (int, []byte, error) {
	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
	}
	request, err := http.NewRequest(method, fmt.Sprintf("http://127.0.0.1:%d%s", port, path), bytes.NewReader(payload))
	if err != nil {
		return 0, nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := (&http.Client{Timeout: 30 * time.Second}).Do(request)
	if err != nil {
		return 0, nil, err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	return response.StatusCode, data, err
}

func parseKeyValues(values []string) (map[string]string, error) {
	out := map[string]string{}
	for _, value := range values {
		name, text, ok := strings.Cut(value, "=")
		if !ok || name == "" {
			return nil, fmt.Errorf("invalid name=value input %q", value)
		}
		if _, exists := out[name]; exists {
			return nil, fmt.Errorf("duplicate input name %q", name)
		}
		out[name] = text
	}
	return out, nil
}

func parseStageAssignments(values []string) (map[string]pipeline.RuntimeAssignment, error) {
	out := map[string]pipeline.RuntimeAssignment{}
	for _, value := range values {
		stage, runtimeValue, ok := strings.Cut(value, "=")
		backend, model, runtimeOK := strings.Cut(runtimeValue, "/")
		if !ok || !runtimeOK || stage == "" || backend == "" || model == "" {
			return nil, fmt.Errorf("invalid stage=backend/model assignment %q", value)
		}
		if _, exists := out[stage]; exists {
			return nil, fmt.Errorf("duplicate stage assignment %q", stage)
		}
		out[stage] = pipeline.RuntimeAssignment{Backend: backend, Model: model}
	}
	return out, nil
}

// parseRoomAssignments builds think_tank_assignments from
// --room-participant stage.participant=backend/model and
// --room-judge stage=backend/model (TS-09.R51).
func parseRoomAssignments(participants, judges []string) (map[string]pipeline.ThinkTankAssignment, error) {
	out := map[string]pipeline.ThinkTankAssignment{}
	slot := func(stage string) pipeline.ThinkTankAssignment {
		room, ok := out[stage]
		if !ok {
			room = pipeline.ThinkTankAssignment{Participants: map[string]pipeline.RuntimeAssignment{}}
		}
		return room
	}
	parsed, err := parseStageAssignments(participants)
	if err != nil {
		return nil, err
	}
	for key, runtime := range parsed {
		stage, participant, ok := strings.Cut(key, ".")
		if !ok || stage == "" || participant == "" {
			return nil, fmt.Errorf("invalid stage.participant=backend/model room participant %q", key)
		}
		room := slot(stage)
		room.Participants[participant] = runtime
		out[stage] = room
	}
	parsed, err = parseStageAssignments(judges)
	if err != nil {
		return nil, err
	}
	for stage, runtime := range parsed {
		room := slot(stage)
		room.Judge = runtime
		out[stage] = room
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func dedicatedAssignments(values map[string]pipeline.RuntimeAssignment) map[string]pipeline.RuntimeAssignment {
	out := make(map[string]pipeline.RuntimeAssignment)
	for stage, assignment := range values {
		if stage != "standing" {
			out[stage] = assignment
		}
	}
	return out
}

func newPipelineRequestID() string {
	var value [8]byte
	if _, err := rand.Read(value[:]); err != nil {
		return fmt.Sprintf("request-%d", time.Now().UnixNano())
	}
	return "request-" + hex.EncodeToString(value[:])
}

func pipelineOperatorPath(use, method, base string, destructive bool, suffix ...string) *cobra.Command {
	cmd := &cobra.Command{Use: use, Short: pipelineCommandShort(use, method), Args: cobra.ArbitraryArgs}
	addOperatorFlags(cmd)
	markPipelineOperator(cmd)
	if destructive {
		cmd.Flags().Bool("yes", false, "acknowledge deletion")
	}
	cmd.RunE = func(c *cobra.Command, args []string) error {
		if destructive {
			yes, _ := c.Flags().GetBool("yes")
			if !yes {
				return &operatorError{Code: "confirmation_required", Message: "this operation deletes data; pass --yes", ExitCode: 2}
			}
		}
		path := base
		if use != "list" {
			if len(args) != 1 {
				return fmt.Errorf("an id is required")
			}
			path += "/" + url.PathEscape(args[0])
		}
		if len(suffix) > 0 && suffix[0] != "" {
			path += "/" + suffix[0]
		}
		return operatorRun(c, method, path, nil, nil)
	}
	if use == "list" {
		cmd.Args = cobra.NoArgs
	} else {
		cmd.Args = cobra.ExactArgs(1)
	}
	return cmd
}

func pipelineOperatorBodyPath(use, method, suffix string, destructive ...bool) *cobra.Command {
	del := len(destructive) > 0 && destructive[0]
	cmd := &cobra.Command{Use: use, Short: pipelineCommandShort(use, method), Args: cobra.ExactArgs(1)}
	addOperatorFlags(cmd)
	markPipelineOperator(cmd)
	if del {
		cmd.Flags().Bool("yes", false, "acknowledge deletion")
	}
	cmd.RunE = func(c *cobra.Command, args []string) error {
		if del {
			yes, _ := c.Flags().GetBool("yes")
			if !yes {
				return &operatorError{Code: "confirmation_required", Message: "this operation deletes data; pass --yes", ExitCode: 2}
			}
		}
		p := "/api/pipeline-runs/" + url.PathEscape(args[0])
		if suffix != "" {
			p += suffix
		}
		if method == http.MethodDelete {
			return operatorRun(c, method, p, nil, nil)
		}
		b, e := operatorRequestBody(c, nil)
		if e != nil {
			return e
		}
		return operatorRun(c, method, p, b, nil)
	}
	return cmd
}

func pipelineCommandShort(use, method string) string {
	verb := "Inspect"
	if method == http.MethodDelete {
		verb = "Delete"
	}
	if method == http.MethodPost || method == http.MethodPut {
		verb = "Apply"
	}
	if use == "list" {
		return "List pipeline records"
	}
	if strings.HasPrefix(use, "show ") {
		return "Show one pipeline record"
	}
	if strings.HasPrefix(use, "decline ") {
		return "Decline a pipeline proposal"
	}
	if strings.HasPrefix(use, "create") {
		return "Create a pipeline record"
	}
	if strings.HasPrefix(use, "update ") {
		return "Update a pipeline template"
	}
	if strings.HasPrefix(use, "delete ") {
		return "Delete a pipeline record"
	}
	return verb + " a pipeline record"
}

func pipelineOperatorMode(cmd *cobra.Command) bool {
	return operatorMode(cmd) || cmd.Flags().Changed("timeout")
}

func markPipelineOperator(cmd *cobra.Command) {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations["operator"] = "true"
}

func documentPipelineCommand(cmd *cobra.Command, help, example string) {
	cmd.Long = help + " Example: " + example + "."
}

func requirePipelineRequestFile(cmd *cobra.Command) error {
	if !cmd.Flags().Changed("request-file") || cmdFlagString(cmd, "request-file") == "" {
		return operatorInputError(fmt.Errorf("--request-file is required"))
	}
	return nil
}

func requirePipelineFields(body map[string]any, fields ...string) error {
	for _, field := range fields {
		value, ok := body[field]
		if !ok || value == nil {
			return operatorInputError(fmt.Errorf("request field %q is required", field))
		}
		switch raw := value.(type) {
		case json.RawMessage:
			trimmed := bytes.TrimSpace(raw)
			if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
				return operatorInputError(fmt.Errorf("request field %q is required", field))
			}
			if field == "id" || field == "template_id" || field == "project" || field == "goal" {
				var text string
				if json.Unmarshal(raw, &text) != nil || strings.TrimSpace(text) == "" {
					return operatorInputError(fmt.Errorf("request field %q must be a nonempty string", field))
				}
			}
		case string:
			if strings.TrimSpace(raw) == "" {
				return operatorInputError(fmt.Errorf("request field %q must be a nonempty string", field))
			}
		}
	}
	return nil
}

func requirePipelineRuntime(body map[string]any, field string) error {
	value, ok := body[field]
	if !ok {
		return operatorInputError(fmt.Errorf("request field %q is required", field))
	}
	var runtime struct {
		Backend string `json:"backend"`
		Model   string `json:"model"`
	}
	switch raw := value.(type) {
	case json.RawMessage:
		if err := json.Unmarshal(raw, &runtime); err != nil {
			return operatorInputError(fmt.Errorf("request field %q must be a runtime object", field))
		}
	case map[string]any:
		encoded, err := json.Marshal(raw)
		if err != nil || json.Unmarshal(encoded, &runtime) != nil {
			return operatorInputError(fmt.Errorf("request field %q must be a runtime object", field))
		}
	default:
		return operatorInputError(fmt.Errorf("request field %q must be a runtime object", field))
	}
	if strings.TrimSpace(runtime.Backend) == "" || strings.TrimSpace(runtime.Model) == "" {
		return operatorInputError(fmt.Errorf("request field %q requires backend and model", field))
	}
	return nil
}

// pipelineStartRequestFile keeps the compact start flags as shortcuts while
// retaining every nested field supplied in a request file. A duplicate source
// is a local error so an operator never has to guess which value won.
func pipelineStartRequestFile(cmd *cobra.Command) (map[string]any, error) {
	raw, err := operatorInput(cmd)
	if err != nil {
		return nil, err
	}
	body := make(map[string]any, len(raw)+12)
	for key, value := range raw {
		body[key] = value
	}
	merge := func(flag, key string, value any) error {
		if !cmd.Flags().Changed(flag) {
			return nil
		}
		if _, exists := raw[key]; exists {
			return operatorInputError(fmt.Errorf("field %q supplied by both --%s and --request-file", key, flag))
		}
		body[key] = value
		return nil
	}
	for _, entry := range []struct{ flag, key, value string }{{"template", "template_id", cmdFlagString(cmd, "template")}, {"name", "display_name", cmdFlagString(cmd, "name")}, {"project", "project", cmdFlagString(cmd, "project")}, {"goal", "goal", cmdFlagString(cmd, "goal")}, {"request-id", "request_id", cmdFlagString(cmd, "request-id")}} {
		if err := merge(entry.flag, entry.key, entry.value); err != nil {
			return nil, err
		}
	}
	inputs, err := parseKeyValues(cmdFlagStrings(cmd, "input"))
	if err != nil {
		return nil, operatorInputError(err)
	}
	if len(inputs) > 0 {
		if err := merge("input", "inputs", inputs); err != nil {
			return nil, err
		}
	}
	assignments, err := parseStageAssignments(cmdFlagStrings(cmd, "stage"))
	if err != nil {
		return nil, operatorInputError(err)
	}
	if len(assignments) > 0 {
		if _, ok := assignments["standing"]; ok {
			return nil, operatorInputError(fmt.Errorf("use --backend and --model for the standing assignment; --stage cannot replace it"))
		}
		if err := merge("stage", "assignments", assignments); err != nil {
			return nil, err
		}
		dedicated := dedicatedAssignments(assignments)
		if err := merge("stage", "dedicated_assignments", dedicated); err != nil {
			return nil, err
		}
	}
	rooms, err := parseRoomAssignments(cmdFlagStrings(cmd, "room-participant"), cmdFlagStrings(cmd, "room-judge"))
	if err != nil {
		return nil, operatorInputError(err)
	}
	if len(rooms) > 0 {
		if _, exists := raw["think_tank_assignments"]; exists {
			return nil, operatorInputError(fmt.Errorf("field %q supplied by both room shortcut flags and --request-file", "think_tank_assignments"))
		}
		body["think_tank_assignments"] = rooms
	}
	standingKeys := []string{"backend", "model", "effort", "fast"}
	standingTouched := false
	standing := map[string]any{}
	for _, flag := range standingKeys {
		if cmd.Flags().Changed(flag) {
			standingTouched = true
		}
	}
	if standingTouched {
		if _, exists := raw["orchestrator"]; exists {
			return nil, operatorInputError(fmt.Errorf("field %q supplied by both shortcut flags and --request-file", "orchestrator"))
		}
		if cmd.Flags().Changed("backend") {
			standing["backend"] = cmdFlagString(cmd, "backend")
		}
		if cmd.Flags().Changed("model") {
			standing["model"] = cmdFlagString(cmd, "model")
		}
		if cmd.Flags().Changed("effort") {
			standing["effort"] = cmdFlagString(cmd, "effort")
		}
		if cmd.Flags().Changed("fast") {
			standing["fast"] = cmdFlagBool(cmd, "fast")
		}
		body["orchestrator"] = standing
	}
	if cmd.Flags().Changed("ack-shared-workspace") {
		if err := merge("ack-shared-workspace", "acknowledge_shared_workspace", cmdFlagBool(cmd, "ack-shared-workspace")); err != nil {
			return nil, err
		}
	}
	return body, nil
}

func pipelineControlRequestFile(cmd *cobra.Command, defaults map[string]any) (map[string]any, error) {
	raw, err := operatorInput(cmd)
	if err != nil {
		return nil, err
	}
	body := make(map[string]any, len(raw)+len(defaults))
	for k, v := range raw {
		body[k] = v
	}
	for _, key := range []string{"revision", "input"} {
		flag := key
		if key == "revision" {
			flag = "revision"
		}
		if !cmd.Flags().Changed(flag) {
			continue
		}
		if _, exists := raw[key]; exists {
			return nil, operatorInputError(fmt.Errorf("field %q supplied by both --%s and --request-file", key, flag))
		}
		if value, ok := defaults[key]; ok {
			body[key] = value
		}
	}
	return body, nil
}

func ensurePipelineRequestID(body map[string]any) error {
	if raw, ok := body["request_id"]; ok {
		var id string
		switch value := raw.(type) {
		case json.RawMessage:
			_ = json.Unmarshal(value, &id)
		case string:
			id = value
		}
		if strings.TrimSpace(id) == "" {
			return operatorInputError(fmt.Errorf("request_id must be a nonempty string"))
		}
		return nil
	}
	body["request_id"] = newPipelineRequestID()
	return nil
}

func requirePipelineRevision(body map[string]any) error {
	raw, ok := body["revision"]
	if !ok {
		return operatorInputError(fmt.Errorf("a positive --revision or request-file revision is required"))
	}
	var revision int64
	switch value := raw.(type) {
	case int:
		revision = int64(value)
	case int64:
		revision = value
	case json.RawMessage:
		if err := json.Unmarshal(value, &revision); err != nil {
			return operatorInputError(fmt.Errorf("revision must be a positive integer"))
		}
	}
	if revision <= 0 {
		return operatorInputError(fmt.Errorf("revision must be a positive current run revision"))
	}
	return nil
}

func cmdFlagString(cmd *cobra.Command, name string) string {
	value, _ := cmd.Flags().GetString(name)
	return value
}
func cmdFlagBool(cmd *cobra.Command, name string) bool {
	value, _ := cmd.Flags().GetBool(name)
	return value
}
func cmdFlagStrings(cmd *cobra.Command, name string) []string {
	value, _ := cmd.Flags().GetStringSlice(name)
	return value
}
