# FS-11 — Project shared resources

**Status:** Partial
**Code:** `internal/config/`, `internal/server/launch.go`, `internal/runtime/`, `ui/src/features/settings/` · **Journeys:** —
**Absorbed:** —

## 1. Purpose

Each Chuck project needs a Chuck-owned place where its agents can leave and reuse working
material: feature specifications, implementation guides, project-specific instructions, research,
test harnesses, and validation results. That material is project-scoped and local to the user's
machine, but intentionally lives outside the project working tree so it cannot become an accidental
repository change or commit.

## 2. Behavior

- **R1** — Every project has one stable shared-resources directory at
  `$CHUCK_HOME/project-resources/{project-id}/`. It is keyed by the immutable project id, not
  by the display title or `cwd`, so renaming a project title or moving its repository leaves the
  resource location unchanged.
- **R2** — Chuck creates the directory when a project is created and also creates
  it lazily for pre-existing projects before their first launch. The directory is empty by default;
  Chuck does not seed, scan, index, synchronize, or otherwise interpret its contents.
- **R3** — Every new agent launch receives the canonical absolute resource-directory
  path as `CHUCK_PROJECT_RESOURCES` and an explicit composed instruction that the directory is
  the project’s shared place for agent-created material, is outside the repository, and may be read
  or written by project agents. It is also included once in the agent's additional accessible
  directories. The agent’s working directory remains the configured project `cwd`.
- **R4** — The project Settings view shows the resource-directory path as a copyable,
  read-only value and explains that it is outside the repository. It does not expose a control for
  choosing a different location in this change.
- **R5** — Deleting a project definition never deletes its shared-resources directory.
  Chuck reports the retained path in the successful delete response/UI confirmation so the user
  can remove it deliberately if desired. Recreating a project with a different id creates a new,
  empty directory; it never adopts a directory based only on title or `cwd`.

## 3. States & transitions

- **R6** — Project creation validates and ensures the corresponding resource directory
  before committing the project definition. A failure to create the directory fails the create
  request without leaving a project definition behind. Launch similarly ensures the directory before
  process start; failure prevents the launch and names the path.
- **R7** — The resource path is included in the frozen additional-directory and prompt
  composition of a running or archived agent. Editing project metadata cannot redirect an existing
  agent; every future launch uses the stable path for its project id.

## 4. Edge cases & errors

- **R8** — Project ids pass the existing slug validation before a resource path is
  constructed. A resource-path component cannot contain separators, dots, or another project’s id.
- **R9** — An existing directory is reused. A missing, non-directory, unreadable, or
  unwritable target produces an actionable error and Chuck must not launch an agent that was not
  told a usable shared location.
- **R10** — Resource contents are opaque user data: they are never returned by the
  dashboard API, emitted over SSE, included in transcripts, or logged merely because the directory
  exists. Agents remain responsible for any secrets they choose to place there under the ordinary
  local-machine trust boundary.

### Project file browser

- **R11 (planned) — Open shared files from the project dashboard.** A visible **Files** button
  opens a modal from a scoped desktop project dashboard, including before its first agent starts.
  The modal identifies the project and starts with files from all folders under its shared-resources
  root (R1). It lists contents
  on demand, independent of agent status or file-tracking history. The checkout, other additional
  directories, and other projects are outside the browser's scope. This adds an explicit read to
  R2/R10 while preserving resource location, agent access, and retention (R1–R7).
- **R12 (planned) — Scan the newest files and narrow the list.** A compact table shows **Name**,
  **Path** (parent folder relative to the shared root), and **Last modified**, newest first. Equal
  timestamps sort by name then path. **All folders** includes files at every depth, including hidden
  files. The folder selector lists directories, including empty ones, and choosing a folder scopes
  the table to that folder and its descendants. The same columns and order apply while filtering.
  **Search files** matches names and relative paths case-insensitively within the selected scope.
  An attached type picker lists the file types in the project. Selections appear as removable
  badges inside the search box with readable labels such as **Markdown**, **TS**, and **CSV**.
  Multiple selected types match any selected type and combine with the text query and folder.
  Files without an extension can be selected as a group. Badges can be removed by pointer or
  keyboard. Clearing text keeps the selected types and folder. Search excludes file contents and
  distinguishes matches, no matches, and incomplete results.
- **R13 (planned) — Preview in place and return to the same list.** Selecting a regular file shows
  the existing viewer in the same modal, with its relative path and supported read-only previews
  and refusals (FS-03.R52, R55 and the viewer's current format support). Close and Escape dismiss
  the modal. **Back to files** restores the prior folder, query, type filters, scroll position, and focus to the
  selected row when it still exists. If it has disappeared, focus returns to **Search files**.
  Opening Files focuses search, keyboard users can activate rows, and dismissing the modal restores
  focus to **Files**. This entry adds no editing, file management,
  paired-phone access, or new agent command.
- **R14 (planned) — Reflect changes and distinguish errors.** Opening the browser reads the current
  directory. **Refresh** updates the active folder or search without clearing its context. Show
  loading, an empty folder, no matches, and a failed read with Retry as distinct states. A legacy
  project whose shared folder has yet to be created shows an empty root without creating it. If a
  folder disappears, explain what happened and offer a return to the root. If a file disappears or
  cannot be previewed, show the reason in the viewer and keep **Back to files** available.
- **R15 (planned) — Preserve the shared-root boundary.** List symbolic links as unavailable entries
  without traversing them or disclosing their targets. Special filesystem entries are also
  unavailable to preview. Directory navigation, search, file actions and links inside a preview
  cannot disclose contents beyond the selected project's shared root. Browsing adds no persistent
  search index, copied file, or change to resource retention.
- **R16 (planned) — Use the shared file menu.** Right-click or the keyboard context-menu command
  opens the file menu on a file row. **Copy path** copies the absolute path of the selected file.
  **Open in Finder** reveals the selected file on the local Mac (FS-03.R119). **Open in new tab**
  is available for HTML files and opens a separate tab in the current browser with the existing
  sandboxed HTML viewer (FS-03.R109). The modal stays on its current list state. Failed clipboard,
  preview, or Finder actions report the failure and preserve the list. The project viewer also
  offers the shared CSV display (FS-03.R120).

## 5. Acceptance criteria

- **A1** — Creating a project produces an owner-only empty directory under a test
  Chuck home, and launching a pre-existing project creates the same stable directory lazily.
  *Verified by:* config/server integration tests.
- **A2** — A launched fake agent sees the canonical path in
  `CHUCK_PROJECT_RESOURCES` and in its composed instruction while its `cwd` remains the project
  working directory. *Verified by:* launch-composition tests.
- **A3** — Project title/`cwd` edits do not change the resource path; deleting the
  project leaves contents intact and a new project id does not reuse them. *Verified by:* project
  CRUD/resource-lifecycle tests.
- **A4** — Invalid ids and unusable filesystem targets fail before process launch; the
  directory and its files remain absent from list/API/SSE/log payloads. *Verified by:* adversarial
  path/mode and server-response tests.
- **A5** — Settings displays and copies the project resource path but offers no mutable
  location field. *Verified by:* Settings UI tests and a usability journey.
- **A6 (planned)** — From a project with no running agents, open **Files** and verify newest-first
  ordering across root and nested files. Search for mixed-case text shared by similarly named
  Markdown, TS and CSV files. Distinguish duplicate names by path, select two types, remove one
  badge by keyboard, and scope to a subfolder. Locate a hidden entry and an empty folder. A file
  existing only in another project or the checkout never appears. *Verify:* project resource
  integration tests and a rendered desktop journey.
- **A7 (planned)** — Open a result in the modal viewer with the keyboard and inspect a rendered
  preview or its Source. Use **Back to files** to restore the query, badges, row, and scroll position,
  then close the modal and continue from **Files**. Long names and parent paths remain distinguishable
  at the supported desktop width. *Verify:* rendered desktop journey including keyboard and narrow
  desktop checks.
- **A8 (planned)** — Distinguish an empty root, an empty nested folder, no search matches, and an
  unreadable directory. Create a nested file externally and find it after Refresh. Remove a file
  after listing it and verify the viewer explains the failed read and still returns to the list.
  *Verify:* filesystem integration tests and a rendered recovery journey.
- **A9 (planned)** — Place links to another project and back into the same directory tree, plus an
  unsupported file. Browse and search without following links or exposing targets, and see the
  viewer's refusal for the unsupported file. Attempts to read outside the root and from a paired
  phone are refused. *Verify:* server boundary tests and a rendered preview refusal check.
- **A10 (planned)** — Open the context menu on HTML and CSV rows. Copy the exact path, reveal the
  selected file in Finder, and open the HTML sandboxed preview in a second browser tab while the
  original retains its query and filter state. Remove the selected file before invoking Finder
  and confirm that the action reports failure and keeps the list. Open the CSV row and see its
  table. *Verify:* focused action tests, a rendered multi-tab journey and a local Mac Finder check.

## 6. Deviations & open decisions

- Projects created before this shipped have no frozen resource path in their existing session
  snapshots, so a resume of such an agent still receives the `CHUCK_PROJECT_RESOURCES` env var
  but not the path in its frozen add_dirs/prompt. New launches compose all three.
- Retention after project deletion is intentional: project resources can contain useful work and
  are not safe to erase as a side effect of removing configuration. A future explicit cleanup tool
  may be designed separately.
- Repository-resident directories, cloud synchronization, full-text search, access control beyond
  the current same-user model, and configurable locations are out of scope.
- R11–R16/A6–A10 are a feature draft. Confirm the all-folders default with a folder selector,
  sandboxed HTML in a new browser tab, and desktop-only shared-folder access without symbolic-link
  traversal before technical design.

## 7. Traceability

- Resource helper: `internal/config/resources.go` (`EnsureProjectResources`, `ProjectResourcesPath`).
- Launch/resume/switch composition: `internal/server/launch.go` (`ensureProjectResources`,
  `projectResourcesEnv`, `projectResourcesInstruction`, `appendResourceDir`), `resume.go`, `switch.go`.
- Project create + `resource_dir` responses: `internal/server/config_handlers.go`, `handlers.go`.
- Settings surface: `ui/src/features/settings/ProjectForm.tsx`, `ProjectsEditor.tsx`.
- Regression anchors: `TestEnsureProjectResourcesCreatesOwnerOnly`,
  `TestEnsureProjectResourcesRejectsSymlink`, `TestLaunchComposesProjectResources`,
  `TestResumeAndSwitchCarryProjectResources`, `TestProjectCreateFailsWhenResourcesUnusable`,
  `TestProjectResponsesExposePathNotContents`.
- Project identity/configuration: FS-00.R6, FS-04.R5–R7 and FS-04.R25–R28.
- Persistence and filesystem safeguards: TS-02.R3/R13 and TS-05.R5–R7/R13.
