# Browse project files

**State:** Waiting to start
**Why:** The operator requested a **Files** button for the project’s agent-shared space and confirmed
the newest-first table, folder scope, type badges, modal preview, HTML new-tab action, Finder and CSV.
**Relevant requirements:** FS-11.R11–R16/A6–A10, FS-03.R119–R120/A94–A95, TS-02.R13,
TS-03.R93–R94, TS-05.R30–R31, TS-08.R152/R172–R174, INV §2/§7/§8/§10/§11/§13/§14/§16/§17

## Outcome

An operator can find the latest agent-generated file in a project's shared space, open it, and
return to the same results without leaving the project dashboard.

## Included work

- Add the desktop **Files** dialog with Name, Path, Last modified, folder selection and removable
  type badges in the name/path search field. Reuse the file viewer inside the dialog.
- Add **Open in new tab** for project HTML through the sandboxed viewer and **Open in Finder**
  for project and chat files. Add CSV **Table** and **Source** to the shared viewer.
- Use the existing `ProjectResourcesPath` owner. `fileread.go` already provides bounded snapshot
  decoding, but the chat loader has broader filesystem authority. Reuse it below authorization,
  with the descriptor-relative no-follow pattern in `attachments.go` for project paths.
  `useFileReader` already accepts an injected loader. The viewer's `HtmlPreview` provides the sandbox for
  the new tab, while the existing chat **Open in browser** action launches the host browser.
- Scope covers the configured project's shared folder. Editing, file management, content search,
  symbolic-link traversal, and paired-phone browsing are excluded. No data migration is needed.

## How we will know it works

- Exercise the project header, sort order, filter combinations, duplicate paths, folders, modal
  return, and recovery at supported desktop widths in the real browser (FS-11.A6–A10).
- Verify root and traversal boundaries with temporary files, including replacement during access,
  partial scans, phone denial, local Host/Origin rejection, and read-only previews.
- Open a sandboxed HTML preview in a second browser tab, verify accurate CSV Table/Source, and
  check that Finder selects the exact file from both project Files and chat on a Mac (FS-03.A94–A95).
