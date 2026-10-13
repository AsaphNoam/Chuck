package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// newOperatorManagementCommands contains the non-conversation operator
// surfaces.  Each command is deliberately a named operation: this is not a
// generic HTTP escape hatch.
func newOperatorManagementCommands() []*cobra.Command {
	return []*cobra.Command{
		newOperatorProjectCmd(), newOperatorGroupCmd(), newOperatorWorktreeCmd(),
		newOperatorArchiveCmd(), newOperatorRoleCmd(), newOperatorBackendCmd(),
		newOperatorConfigCmd(), newOperatorConfigSourceCmd(), newOperatorRemoteCmd(),
	}
}

func operatorRead(cmd *cobra.Command, path string) error {
	return operatorRun(cmd, http.MethodGet, path, nil, nil)
}

func opRaw(s string) json.RawMessage { return json.RawMessage(strconv.Quote(s)) }
func opSetBoolChecked(body map[string]json.RawMessage, key string, value bool) error {
	if _, exists := body[key]; exists {
		return fmt.Errorf("duplicate request field %q", key)
	}
	body[key] = json.RawMessage(strconv.FormatBool(value))
	return nil
}
func opJSONInput(cmd *cobra.Command, body map[string]json.RawMessage) (map[string]json.RawMessage, error) {
	file, err := operatorInput(cmd)
	if err != nil {
		return nil, err
	}
	if file == nil {
		file = map[string]json.RawMessage{}
	}
	for k, v := range body {
		if _, ok := file[k]; !ok {
			file[k] = v
		}
	}
	return file, nil
}
func opNeedYes(yes bool) error {
	if !yes {
		return &operatorError{Code: "invalid_input", Message: "this operation requires --yes", ExitCode: 2}
	}
	return nil
}
func opPath(base, id string) string { return base + "/" + url.PathEscape(id) }
func addManagementFlags(c *cobra.Command) {
	for _, x := range c.Commands() {
		addOperatorFlags(x)
		if x.Args == nil {
			x.Args = cobra.NoArgs
		}
		x.Long = managementHelp(c.Name(), x.Name(), x.Short)
		if x.Example == "" {
			x.Example = operatorExample(c.Use, x.Use)
		}
		old := x.RunE
		x.RunE = func(cmd *cobra.Command, args []string) error {
			request, _ := cmd.Flags().GetString("request-file")
			if request != "" {
				switch cmd.Name() {
				case "list", "show", "status", "search", "devices", "pairing-code", "allow", "decline", "detach", "revoke", "delete":
					return operatorInvalid("this operation uses explicit target flags and does not accept --request-file")
				}
				if c.Name() == "config-source" && cmd.Name() == "refresh" {
					return operatorInvalid("source refresh accepts only its target flag")
				}
			}
			for _, flag := range []string{"project", "id", "backend-id", "group"} {
				if f := cmd.Flags().Lookup(flag); f != nil && cmd.Flags().Changed(flag) {
					value, _ := cmd.Flags().GetString(flag)
					if strings.TrimSpace(value) == "" {
						return operatorInvalid("--" + flag + " must name an exact target")
					}
				}
			}
			return old(cmd, args)
		}
	}
}

func managementHelp(family, action, short string) string {
	shapes := map[string]string{
		"project:create":        "project JSON with required title and cwd; optional project id and project settings",
		"project:edit":          "partial project JSON settings",
		"project:archive":       "optional delete_checkout, dirty_known and dirty fields; dirty requires dirty_known:true; --project selects the target",
		"project:restore":       "optional project restoration JSON; --project selects the target",
		"group:stop":            "{group}; the label is scoped to --project",
		"group:archive":         "{group}; the label is scoped to --project",
		"worktree:fork":         "{title,branch,base?}; --project identifies the source workspace",
		"role:create":           "{role,title,system_prompt?,chuck_aware?,skip_permissions?}",
		"role:edit":             "partial role JSON, such as {title,system_prompt,chuck_aware,skip_permissions}",
		"backend:create":        "{backend_id,name,type,connect_native_configuration?}",
		"backend:replace":       "{version,backends}; --etag supplies the catalog If-Match value",
		"backend:refresh":       "optional {model_id}; --etag supplies the catalog If-Match value",
		"config:edit":           "partial editable configuration; version and port are immutable",
		"config-source:preview": "{provider,root,profile?,mode,claims?,project?}",
		"config-source:bind":    "{preview_token,overrides?,enable_model_sync?}; the token must come from preview",
		"remote:configure":      "partial {enabled,keep_awake} settings",
		"remote:rename":         "{name}; --id selects the paired device",
	}
	text := short + "."
	if shape, ok := shapes[family+":"+action]; ok {
		text += " Read one JSON object with --request-file <path|-> using this shape: " + shape + "."
		text += " Shortcut values cannot conflict with fields in that object. --json preserves the server response, including unknown fields."
	} else {
		text += " Use the explicit target and filter flags shown below; this operation does not accept --request-file."
		if action == "list" || action == "show" || action == "status" || action == "search" || action == "devices" {
			text += " --json preserves the server response, including unknown fields."
		}
	}
	if action == "delete" || action == "archive" || family == "group" || action == "bind" || action == "detach" || action == "revoke" {
		text += " --yes acknowledges this operation's deletion, scoped stop/archive or trust-binding consequence."
	}
	if family == "project" && (action == "delete" || action == "archive") {
		text += " Checkout deletion and dirty-checkout expectations are separate explicit flags; --dirty requires --dirty-known, and --yes does not supply either."
	}
	if family == "role" && action == "delete" {
		text += " Use --force separately when deleting a role with active references."
	}
	return text
}

func mergeManagementFlags(cmd *cobra.Command, body map[string]json.RawMessage, fields map[string]string) error {
	for key, flag := range fields {
		if !cmd.Flags().Changed(flag) {
			continue
		}
		var value any
		if f := cmd.Flags().Lookup(flag); f.Value.Type() == "bool" {
			value, _ = cmd.Flags().GetBool(flag)
		} else {
			value, _ = cmd.Flags().GetString(flag)
		}
		if err := operatorMerge(body, key, value); err != nil {
			return operatorInvalid(err.Error())
		}
	}
	return nil
}

func operatorExample(parent, use string) string {
	parts := strings.Fields(use)
	op := ""
	if len(parts) > 0 {
		op = parts[0]
	}
	switch parent + ":" + op {
	case "project:list":
		return "chuck project list"
	case "project:show":
		return "chuck project show --project demo"
	case "project:create":
		return "chuck project create --request-file project.json"
	case "project:edit":
		return "chuck project edit --project demo --request-file changes.json"
	case "project:delete":
		return "chuck project delete --project demo --yes"
	case "project:archive":
		return "chuck project archive --project demo --yes"
	case "project:restore":
		return "chuck project restore --project demo"
	case "group:stop":
		return "chuck group stop --project demo --group workers --yes"
	case "group:archive":
		return "chuck group archive --project demo --group workers --yes"
	case "worktree:fork":
		return "chuck worktree fork --project demo --title 'Demo fork' --branch demo-fork"
	case "worktree:status":
		return "chuck worktree status --project demo"
	case "archive:search":
		return "chuck archive search --project demo --limit 20"
	case "role:list":
		return "chuck role list"
	case "role:show":
		return "chuck role show --id reviewer"
	case "role:create":
		return "chuck role create --request-file role.json"
	case "role:edit":
		return "chuck role edit --id reviewer --request-file changes.json"
	case "role:delete":
		return "chuck role delete --id reviewer --yes"
	case "backend:list":
		return "chuck backend list"
	case "backend:show":
		return "chuck backend show --id claude"
	case "backend:create":
		return "chuck backend create --id local --name Local --type claude-acp"
	case "backend:replace":
		return "chuck backend replace --request-file backends.json --etag 'W/\"catalog\"'"
	case "backend:refresh":
		return "chuck backend refresh --id claude --model-id claude-sonnet-4-5 --etag 'W/\"catalog\"'"
	case "config:show":
		return "chuck config show"
	case "config:edit":
		return "chuck config edit --request-file settings.json"
	case "config-source:list":
		return "chuck config-source list --project demo"
	case "config-source:preview":
		return "chuck config-source preview --request-file source.json"
	case "config-source:bind":
		return "chuck config-source bind --backend-id claude --request-file bind.json --yes"
	case "config-source:refresh":
		return "chuck config-source refresh --backend-id claude"
	case "config-source:detach":
		return "chuck config-source detach --backend-id claude --yes"
	case "remote:show":
		return "chuck remote show"
	case "remote:configure":
		return "chuck remote configure --enabled"
	case "remote:pairing-code":
		return "chuck remote pairing-code --json"
	case "remote:allow":
		return "chuck remote allow --id pairing-id"
	case "remote:decline":
		return "chuck remote decline --id pairing-id"
	case "remote:devices":
		return "chuck remote devices"
	case "remote:rename":
		return "chuck remote rename --id device-id --name 'Work phone'"
	case "remote:revoke":
		return "chuck remote revoke --id device-id --yes"
	default:
		return "chuck " + parent + " " + use
	}
}

func requireNonemptyFields(body map[string]json.RawMessage, fields ...string) error {
	for _, field := range fields {
		raw, ok := body[field]
		if !ok {
			return fmt.Errorf("%s is required", field)
		}
		var value string
		if json.Unmarshal(raw, &value) != nil || value == "" {
			return fmt.Errorf("%s must be non-empty", field)
		}
	}
	return nil
}

func requireKnownDirtyExpectation(body map[string]json.RawMessage) error {
	if _, ok := body["dirty"]; !ok {
		return nil
	}
	var known bool
	if json.Unmarshal(body["dirty_known"], &known) != nil || !known {
		return fmt.Errorf("dirty requires dirty_known=true")
	}
	return nil
}

func operatorCollectionShow(cmd *cobra.Command, path, id, wrapper string) error {
	client, err := operatorClientFactory()
	if err != nil {
		return err
	}
	ctx, cancel, err := operatorContext(cmd)
	if err != nil {
		return err
	}
	defer cancel()
	data, err := client.Request(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return err
	}
	var values map[string]json.RawMessage
	if err := json.Unmarshal(data, &values); err != nil || values == nil {
		return &operatorError{Code: "invalid_response", Message: "dashboard returned an invalid collection object", ExitCode: 1}
	}
	if wrapper != "" {
		var wrapped map[string]json.RawMessage
		raw, ok := values[wrapper]
		if !ok || json.Unmarshal(raw, &wrapped) != nil || wrapped == nil {
			return &operatorError{Code: "invalid_response", Message: "dashboard returned a collection without " + wrapper, ExitCode: 1}
		}
		values = wrapped
	}
	item, ok := values[id]
	if !ok {
		return &operatorError{Code: "not_found", Message: "no such " + path + " target", HTTPStatus: http.StatusNotFound, ExitCode: 1}
	}
	return operatorOutput(cmd, item)
}

func newOperatorProjectCmd() *cobra.Command {
	c := &cobra.Command{Use: "project", Short: "Manage projects"}
	list := &cobra.Command{Use: "list", Short: "List projects", RunE: func(cmd *cobra.Command, _ []string) error { return operatorRead(cmd, "/api/projects") }}
	c.AddCommand(list)
	id := new(string)
	show := &cobra.Command{Use: "show", Short: "Show one project", RunE: func(cmd *cobra.Command, _ []string) error {
		return operatorCollectionShow(cmd, "/api/projects", *id, "")
	}}
	show.Flags().StringVar(id, "project", "", "project id")
	_ = show.MarkFlagRequired("project")
	c.AddCommand(show)
	create := &cobra.Command{Use: "create", Short: "Create a project", RunE: func(cmd *cobra.Command, _ []string) error {
		b, err := opJSONInput(cmd, map[string]json.RawMessage{})
		if err != nil {
			return err
		}
		if err := mergeManagementFlags(cmd, b, map[string]string{"project": "project"}); err != nil {
			return err
		}
		if err := requireNonemptyFields(b, "title", "cwd"); err != nil {
			return operatorInvalid(err.Error())
		}
		return operatorRun(cmd, http.MethodPost, "/api/projects", b, nil)
	}}
	create.Flags().StringVar(id, "project", "", "project id")
	c.AddCommand(create)
	edit := &cobra.Command{Use: "edit", Short: "Edit a project", RunE: func(cmd *cobra.Command, _ []string) error {
		b, err := opJSONInput(cmd, map[string]json.RawMessage{})
		if err != nil {
			return err
		}
		return operatorRun(cmd, http.MethodPut, opPath("/api/projects", *id), b, nil)
	}}
	edit.Flags().StringVar(id, "project", "", "project id")
	_ = edit.MarkFlagRequired("project")
	c.AddCommand(edit)
	force, checkout, dirtyKnown, dirty, yes := false, false, false, false, false
	del := &cobra.Command{Use: "delete", Short: "Delete a project", RunE: func(cmd *cobra.Command, _ []string) error {
		if err := opNeedYes(yes); err != nil {
			return err
		}
		if cmd.Flags().Changed("dirty") && !dirtyKnown {
			return operatorInvalid("--dirty requires --dirty-known")
		}
		q := url.Values{}
		if force {
			q.Set("force", "true")
		}
		if checkout {
			q.Set("delete_checkout", "true")
		}
		if dirtyKnown {
			q.Set("dirty_known", "true")
			if dirty {
				q.Set("dirty", "true")
			}
		}
		p := opPath("/api/projects", *id)
		if len(q) > 0 {
			p += "?" + q.Encode()
		}
		return operatorRun(cmd, http.MethodDelete, p, nil, nil)
	}}
	del.Flags().StringVar(id, "project", "", "project id")
	del.Flags().BoolVar(&force, "force", false, "force deletion")
	del.Flags().BoolVar(&checkout, "delete-checkout", false, "delete owned checkout")
	del.Flags().BoolVar(&dirtyKnown, "dirty-known", false, "provide dirty expectation")
	del.Flags().BoolVar(&dirty, "dirty", false, "expected dirty state")
	del.Flags().BoolVar(&yes, "yes", false, "confirm")
	_ = del.MarkFlagRequired("project")
	c.AddCommand(del)
	for _, action := range []string{"archive", "restore"} {
		a := action
		var deleteCheckout, dirtyKnown, dirty bool
		x := &cobra.Command{Use: a, Short: a + " a project", RunE: func(cmd *cobra.Command, _ []string) error {
			if a == "archive" {
				if err := opNeedYes(yes); err != nil {
					return err
				}
				if cmd.Flags().Changed("dirty") && !dirtyKnown {
					return operatorInvalid("--dirty requires --dirty-known")
				}
			}
			body, err := opJSONInput(cmd, map[string]json.RawMessage{})
			if err != nil {
				return err
			}
			if a == "archive" {
				if cmd.Flags().Changed("delete-checkout") {
					if err = opSetBoolChecked(body, "delete_checkout", deleteCheckout); err != nil {
						return err
					}
				}
				if cmd.Flags().Changed("dirty-known") {
					if err = opSetBoolChecked(body, "dirty_known", dirtyKnown); err != nil {
						return err
					}
				}
				if cmd.Flags().Changed("dirty") {
					if err = opSetBoolChecked(body, "dirty", dirty); err != nil {
						return err
					}
				}
				if err = requireKnownDirtyExpectation(body); err != nil {
					return operatorInvalid(err.Error())
				}
			}
			return operatorRun(cmd, http.MethodPost, opPath("/api/projects", *id)+"/"+a, body, nil)
		}}
		x.Flags().StringVar(id, "project", "", "project id")
		_ = x.MarkFlagRequired("project")
		x.Flags().BoolVar(&yes, "yes", false, "confirm")
		x.Flags().BoolVar(&deleteCheckout, "delete-checkout", false, "delete owned checkout")
		x.Flags().BoolVar(&dirtyKnown, "dirty-known", false, "provide dirty expectation")
		x.Flags().BoolVar(&dirty, "dirty", false, "expected dirty state")
		c.AddCommand(x)
	}
	addManagementFlags(c)
	return c
}

func newOperatorGroupCmd() *cobra.Command {
	c := &cobra.Command{Use: "group", Short: "Control project groups"}
	for _, a := range []string{"stop", "archive"} {
		a := a
		var project, group string
		var yes bool
		x := &cobra.Command{Use: a, Short: a + " a named group", RunE: func(cmd *cobra.Command, _ []string) error {
			if err := opNeedYes(yes); err != nil {
				return err
			}
			body, err := operatorInput(cmd)
			if err != nil {
				return err
			}
			if err = mergeManagementFlags(cmd, body, map[string]string{"group": "group"}); err != nil {
				return err
			}
			if err = requireNonemptyFields(body, "group"); err != nil {
				return operatorInvalid(err.Error())
			}
			return operatorRun(cmd, http.MethodPost, opPath("/api/projects", project)+"/groups/"+a, body, nil)
		}}
		x.Flags().StringVar(&project, "project", "", "project id")
		x.Flags().StringVar(&group, "group", "", "group label")
		_ = x.MarkFlagRequired("project")
		x.Flags().BoolVar(&yes, "yes", false, "confirm")
		c.AddCommand(x)
	}
	addManagementFlags(c)
	return c
}

func newOperatorWorktreeCmd() *cobra.Command {
	c := &cobra.Command{Use: "worktree", Short: "Manage project worktrees"}
	var project, title, branch, base string
	fork := &cobra.Command{Use: "fork", Short: "Fork a project worktree", RunE: func(cmd *cobra.Command, _ []string) error {
		b, err := opJSONInput(cmd, map[string]json.RawMessage{})
		if err != nil {
			return err
		}
		if err := mergeManagementFlags(cmd, b, map[string]string{"title": "title", "branch": "branch", "base": "base"}); err != nil {
			return err
		}
		if err := requireNonemptyFields(b, "title", "branch"); err != nil {
			return err
		}
		return operatorRun(cmd, http.MethodPost, opPath("/api/projects", project)+"/worktree-fork", b, nil)
	}}
	fork.Flags().StringVar(&project, "project", "", "project id")
	fork.Flags().StringVar(&title, "title", "", "title")
	fork.Flags().StringVar(&branch, "branch", "", "branch")
	fork.Flags().StringVar(&base, "base", "", "base revision")
	_ = fork.MarkFlagRequired("project")
	c.AddCommand(fork)
	status := &cobra.Command{Use: "status", Short: "Show worktree status", RunE: func(cmd *cobra.Command, _ []string) error {
		return operatorRead(cmd, opPath("/api/projects", project)+"/worktree")
	}}
	status.Flags().StringVar(&project, "project", "", "project id")
	_ = status.MarkFlagRequired("project")
	c.AddCommand(status)
	addManagementFlags(c)
	return c
}

func newOperatorArchiveCmd() *cobra.Command {
	c := &cobra.Command{Use: "archive", Short: "Search retained archive"}
	var q, project, active string
	var limit, offset int
	list := &cobra.Command{Use: "search", Short: "Search archive", RunE: func(cmd *cobra.Command, _ []string) error {
		if limit < 1 || limit > 200 || offset < 0 || (active != "" && active != "true" && active != "false") {
			return operatorInvalid("--limit must be 1–200, --offset nonnegative and --active true|false")
		}
		v := url.Values{}
		if q != "" {
			v.Set("q", q)
		}
		if active != "" {
			v.Set("active", active)
		}
		v.Set("limit", strconv.Itoa(limit))
		v.Set("offset", strconv.Itoa(offset))
		if project != "" {
			return operatorRead(cmd, opPath("/api/archive/projects", project)+"?"+v.Encode())
		}
		return operatorRead(cmd, "/api/archive?"+v.Encode())
	}}
	list.Flags().StringVar(&q, "query", "", "query")
	list.Flags().StringVar(&project, "project", "", "project id")
	list.Flags().StringVar(&active, "active", "", "true or false")
	list.Flags().IntVar(&limit, "limit", 50, "limit")
	list.Flags().IntVar(&offset, "offset", 0, "offset")
	c.AddCommand(list)
	addManagementFlags(c)
	return c
}

func newOperatorRoleCmd() *cobra.Command {
	c := &cobra.Command{Use: "role", Short: "Manage roles"}
	c.AddCommand(operatorCollectionCommands("role", "/api/roles", true)...)
	return c
}
func newOperatorBackendCmd() *cobra.Command {
	c := &cobra.Command{Use: "backend", Short: "Manage backends"}
	list := &cobra.Command{Use: "list", Short: "List backends and optionally save the response ETag", RunE: func(cmd *cobra.Command, _ []string) error {
		output, _ := cmd.Flags().GetString("etag-output")
		if output == "" {
			return operatorRead(cmd, "/api/backends")
		}
		if _, err := os.Lstat(output); err == nil {
			return operatorInvalid("ETag destination exists; choose a new path")
		} else if !os.IsNotExist(err) {
			return operatorInvalid("cannot inspect ETag destination")
		}
		ctx, cancel, err := operatorContext(cmd)
		if err != nil {
			return err
		}
		defer cancel()
		client, err := operatorClientFactory()
		if err != nil {
			return err
		}
		data, err := client.Request(ctx, "GET", "/api/backends", nil, nil)
		if err != nil {
			return err
		}
		etag := client.responseHeaders.Get("ETag")
		if etag == "" {
			return &operatorError{Code: "invalid_response", Message: "backend response did not include its catalog ETag", ExitCode: 1}
		}
		file, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return operatorInvalid("cannot create ETag destination")
		}
		_, err = file.WriteString(etag + "\n")
		if err == nil {
			err = file.Sync()
		}
		closeErr := file.Close()
		if err == nil {
			err = closeErr
		}
		if err != nil {
			_ = os.Remove(output)
			return &operatorError{Code: "output_failed", Message: "could not save catalog ETag", ExitCode: 1}
		}
		return operatorOutput(cmd, data)
	}}
	list.Flags().String("etag-output", "", "save the response ETag to a new owner-only file without changing stdout")
	c.AddCommand(list)
	var id, model, etag, name, typ string
	show := &cobra.Command{Use: "show", Short: "Show one backend", RunE: func(cmd *cobra.Command, _ []string) error {
		return operatorCollectionShow(cmd, "/api/backends", id, "backends")
	}}
	show.Flags().StringVar(&id, "id", "", "backend id")
	_ = show.MarkFlagRequired("id")
	c.AddCommand(show)
	create := &cobra.Command{Use: "create", Short: "Create a backend", RunE: func(cmd *cobra.Command, _ []string) error {
		b, err := opJSONInput(cmd, map[string]json.RawMessage{})
		if err != nil {
			return err
		}
		if err = mergeManagementFlags(cmd, b, map[string]string{"backend_id": "id", "name": "name", "type": "type"}); err != nil {
			return err
		}
		if err = requireNonemptyFields(b, "backend_id", "name", "type"); err != nil {
			return err
		}
		return operatorRun(cmd, http.MethodPost, "/api/backends", b, nil)
	}}
	create.Flags().StringVar(&id, "id", "", "backend id")
	create.Flags().StringVar(&name, "name", "", "display name")
	create.Flags().StringVar(&typ, "type", "", "backend type")
	c.AddCommand(create)
	refresh := &cobra.Command{Use: "refresh", Short: "Refresh provider catalog", RunE: func(cmd *cobra.Command, _ []string) error {
		if etag == "" {
			return operatorInvalid("--etag from backend list is required")
		}
		h := http.Header{}
		if etag != "" {
			h.Set("If-Match", etag)
		}
		b, err := operatorInput(cmd)
		if err != nil {
			return err
		}
		if err = mergeManagementFlags(cmd, b, map[string]string{"model_id": "model-id"}); err != nil {
			return err
		}
		return operatorRun(cmd, http.MethodPost, opPath("/api/backends", id)+"/refresh-provider", b, h)
	}}
	refresh.Flags().StringVar(&id, "id", "", "backend id")
	refresh.Flags().StringVar(&model, "model-id", "", "model id")
	refresh.Flags().StringVar(&etag, "etag", "", "ETag")
	_ = refresh.MarkFlagRequired("id")
	c.AddCommand(refresh)
	replace := &cobra.Command{Use: "replace", Short: "Replace backend catalog", RunE: func(cmd *cobra.Command, _ []string) error {
		if etag == "" {
			return operatorInvalid("--etag from backend list is required")
		}
		b, err := operatorInput(cmd)
		if err != nil {
			return err
		}
		h := http.Header{}
		if etag != "" {
			h.Set("If-Match", etag)
		}
		return operatorRun(cmd, http.MethodPut, "/api/backends", b, h)
	}}
	replace.Flags().StringVar(&etag, "etag", "", "ETag")
	c.AddCommand(replace)
	addManagementFlags(c)
	return c
}

func operatorCollectionCommands(kind, path string, mutate bool) []*cobra.Command {
	list := &cobra.Command{Use: "list", Short: "List " + kind + "s", RunE: func(cmd *cobra.Command, _ []string) error { return operatorRead(cmd, path) }}
	show := &cobra.Command{Use: "show", Short: "Show one " + kind, RunE: func(cmd *cobra.Command, _ []string) error {
		id, _ := cmd.Flags().GetString("id")
		if strings.TrimSpace(id) == "" {
			return operatorInvalid("--id must name an exact role")
		}
		return operatorCollectionShow(cmd, path, id, "")
	}}
	show.Flags().String("id", "", "id")
	_ = show.MarkFlagRequired("id")
	create := &cobra.Command{Use: "create", Short: "Create " + kind, RunE: func(cmd *cobra.Command, _ []string) error {
		b, e := operatorInput(cmd)
		if e != nil {
			return e
		}
		if err := requireNonemptyFields(b, "role", "title"); err != nil {
			return operatorInvalid(err.Error())
		}
		return operatorRun(cmd, http.MethodPost, path, b, nil)
	}}
	edit := &cobra.Command{Use: "edit", Short: "Edit " + kind, RunE: func(cmd *cobra.Command, _ []string) error {
		b, e := operatorInput(cmd)
		if e != nil {
			return e
		}
		id, _ := cmd.Flags().GetString("id")
		if strings.TrimSpace(id) == "" {
			return operatorInvalid("--id must name an exact role")
		}
		return operatorRun(cmd, http.MethodPut, opPath(path, id), b, nil)
	}}
	del := &cobra.Command{Use: "delete", Short: "Delete " + kind, RunE: func(cmd *cobra.Command, _ []string) error {
		yes, _ := cmd.Flags().GetBool("yes")
		if e := opNeedYes(yes); e != nil {
			return e
		}
		id, _ := cmd.Flags().GetString("id")
		if strings.TrimSpace(id) == "" {
			return operatorInvalid("--id must name an exact role")
		}
		force, _ := cmd.Flags().GetBool("force")
		query := ""
		if force {
			query = "?force=true"
		}
		return operatorRun(cmd, http.MethodDelete, opPath(path, id)+query, nil, nil)
	}}
	for _, x := range []*cobra.Command{list, show, create, edit, del} {
		addOperatorFlags(x)
		if x.Args == nil {
			x.Args = cobra.NoArgs
		}
		x.Long = managementHelp(kind, x.Name(), x.Short)
		x.Example = operatorExample(kind, x.Use)
		if x.Name() == "list" || x.Name() == "show" || x.Name() == "delete" {
			oldRun := x.RunE
			x.RunE = func(cmd *cobra.Command, args []string) error {
				request, _ := cmd.Flags().GetString("request-file")
				if request != "" {
					return operatorInvalid("this operation uses explicit target flags and does not accept --request-file")
				}
				return oldRun(cmd, args)
			}
		}
	}
	edit.Flags().String("id", "", "id")
	_ = edit.MarkFlagRequired("id")
	del.Flags().String("id", "", "id")
	del.Flags().Bool("yes", false, "confirm")
	del.Flags().Bool("force", false, "explicitly delete despite active references")
	_ = del.MarkFlagRequired("id")
	return []*cobra.Command{list, show, create, edit, del}
}

func newOperatorConfigCmd() *cobra.Command {
	c := &cobra.Command{Use: "config", Short: "Inspect and edit configuration"}
	c.AddCommand(&cobra.Command{Use: "show", Short: "Show config", RunE: func(cmd *cobra.Command, _ []string) error { return operatorRead(cmd, "/api/config") }})
	edit := &cobra.Command{Use: "edit", Short: "Edit config", RunE: func(cmd *cobra.Command, _ []string) error {
		b, e := operatorInput(cmd)
		if e != nil {
			return e
		}
		for _, k := range []string{"version", "port"} {
			if _, ok := b[k]; ok {
				return fmt.Errorf("%s is immutable", k)
			}
		}
		return operatorRun(cmd, http.MethodPut, "/api/config", b, nil)
	}}
	c.AddCommand(edit)
	addManagementFlags(c)
	return c
}
func newOperatorConfigSourceCmd() *cobra.Command {
	c := &cobra.Command{Use: "config-source", Short: "Manage native configuration federation"}
	list := &cobra.Command{Use: "list", Short: "List bindings and candidates", RunE: func(cmd *cobra.Command, _ []string) error {
		project, _ := cmd.Flags().GetString("project")
		path := "/api/config-sources"
		if project != "" {
			path += "?" + url.Values{"project": {project}}.Encode()
		}
		return operatorRead(cmd, path)
	}}
	list.Flags().String("project", "", "project context for federation candidates")
	c.AddCommand(list)
	var backend string
	var yes bool
	bind := &cobra.Command{Use: "bind", Short: "Bind a previewed source", RunE: func(cmd *cobra.Command, _ []string) error {
		if err := opNeedYes(yes); err != nil {
			return err
		}
		b, e := operatorInput(cmd)
		if e != nil {
			return e
		}
		if e = requireNonemptyFields(b, "preview_token"); e != nil {
			return operatorInvalid(e.Error())
		}
		return operatorRun(cmd, http.MethodPut, opPath("/api/config-sources", backend), b, nil)
	}}
	bind.Flags().StringVar(&backend, "backend-id", "", "backend id")
	bind.Flags().BoolVar(&yes, "yes", false, "confirm")
	_ = bind.MarkFlagRequired("backend-id")
	c.AddCommand(bind)
	detach := &cobra.Command{Use: "detach", Short: "Detach a source binding", RunE: func(cmd *cobra.Command, _ []string) error {
		if err := opNeedYes(yes); err != nil {
			return err
		}
		return operatorRun(cmd, http.MethodDelete, opPath("/api/config-sources", backend), nil, nil)
	}}
	detach.Flags().StringVar(&backend, "backend-id", "", "backend id")
	detach.Flags().BoolVar(&yes, "yes", false, "confirm")
	_ = detach.MarkFlagRequired("backend-id")
	c.AddCommand(detach)
	c.AddCommand(&cobra.Command{Use: "preview", Short: "Preview a source", RunE: func(cmd *cobra.Command, _ []string) error {
		b, e := operatorInput(cmd)
		if e != nil {
			return e
		}
		return operatorRun(cmd, http.MethodPost, "/api/config-sources/preview", b, nil)
	}})
	refresh := &cobra.Command{Use: "refresh", Short: "Refresh a binding", RunE: func(cmd *cobra.Command, _ []string) error {
		b, _ := cmd.Flags().GetString("backend-id")
		return operatorRun(cmd, http.MethodPost, opPath("/api/config-sources", b)+"/refresh", nil, nil)
	}}
	refresh.Flags().String("backend-id", "", "backend id")
	_ = refresh.MarkFlagRequired("backend-id")
	c.AddCommand(refresh)
	addManagementFlags(c)
	return c
}
func newOperatorRemoteCmd() *cobra.Command {
	c := &cobra.Command{Use: "remote", Short: "Manage remote administration"}
	var id, name string
	c.AddCommand(&cobra.Command{Use: "show", Short: "Show remote settings", RunE: func(cmd *cobra.Command, _ []string) error { return operatorRead(cmd, "/api/remote") }})
	cfg := &cobra.Command{Use: "configure", Short: "Configure remote settings", RunE: func(cmd *cobra.Command, _ []string) error {
		b, e := operatorInput(cmd)
		if e != nil {
			return e
		}
		if err := mergeManagementFlags(cmd, b, map[string]string{"enabled": "enabled", "keep_awake": "keep-awake"}); err != nil {
			return err
		}
		if len(b) == 0 {
			return operatorInvalid("supply enabled or keep_awake settings")
		}
		return operatorRun(cmd, http.MethodPut, "/api/remote", b, nil)
	}}
	cfg.Flags().Bool("enabled", false, "enable or disable remote access")
	cfg.Flags().Bool("keep-awake", false, "enable or disable keep-awake")
	c.AddCommand(cfg)
	c.AddCommand(&cobra.Command{Use: "pairing-code", Short: "Create pairing code", RunE: func(cmd *cobra.Command, _ []string) error {
		return operatorRun(cmd, http.MethodPost, "/api/remote/pairings", nil, nil)
	}})
	for _, action := range []string{"allow", "decline"} {
		a := action
		x := &cobra.Command{Use: a, Short: a + " a pending pairing", RunE: func(cmd *cobra.Command, _ []string) error {
			return operatorRun(cmd, http.MethodPost, opPath("/api/remote/pairings", id)+"/"+a, nil, nil)
		}}
		x.Flags().StringVar(&id, "id", "", "pairing id")
		_ = x.MarkFlagRequired("id")
		c.AddCommand(x)
	}
	c.AddCommand(&cobra.Command{Use: "devices", Short: "List paired devices", RunE: func(cmd *cobra.Command, _ []string) error { return operatorRead(cmd, "/api/remote/devices") }})
	rename := &cobra.Command{Use: "rename", Short: "Rename a device", RunE: func(cmd *cobra.Command, _ []string) error {
		b, err := operatorInput(cmd)
		if err != nil {
			return err
		}
		if err = mergeManagementFlags(cmd, b, map[string]string{"name": "name"}); err != nil {
			return err
		}
		if err = requireNonemptyFields(b, "name"); err != nil {
			return operatorInvalid(err.Error())
		}
		return operatorRun(cmd, http.MethodPatch, opPath("/api/remote/devices", id), b, nil)
	}}
	rename.Flags().StringVar(&id, "id", "", "device id")
	rename.Flags().StringVar(&name, "name", "", "name")
	_ = rename.MarkFlagRequired("id")
	c.AddCommand(rename)
	var yes bool
	revoke := &cobra.Command{Use: "revoke", Short: "Revoke a device", RunE: func(cmd *cobra.Command, _ []string) error {
		if e := opNeedYes(yes); e != nil {
			return e
		}
		return operatorRun(cmd, http.MethodDelete, opPath("/api/remote/devices", id), nil, nil)
	}}
	revoke.Flags().StringVar(&id, "id", "", "device id")
	revoke.Flags().BoolVar(&yes, "yes", false, "confirm")
	_ = revoke.MarkFlagRequired("id")
	c.AddCommand(revoke)
	addManagementFlags(c)
	return c
}
