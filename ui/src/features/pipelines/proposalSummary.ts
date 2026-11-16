import { pipelineProposalSchema, type PipelineListedProposal, type PipelineProposal, type PipelineTemplateRecord, type PipelineWorkspaceSelection } from "../../schemas/pipeline";
import type { PipelineWorkspacePreview } from "../../api/pipelines";

// One draft must never push the template library or the run ledger off the
// surface, so every summarized value is clipped here rather than at each call
// site (FS-14.R51).
export const proposalSummaryLimit = 120;

export function clipProposalValue(value: string, limit = proposalSummaryLimit) {
  const chars = [...value];
  if (chars.length <= limit) return value;
  return chars.slice(0, limit - 1).join("") + "…";
}

export interface ProposalSummaryField {
  label: string;
  value: string;
}

// asPipelineProposal is the one place a listed record becomes a typed proposal.
// A payload that does not match its kind cannot be summarized or rendered for
// exact approval, but the record still lists with its kind and proposal id
// rather than disappearing (FS-14.R51).
export function asPipelineProposal(entry: PipelineListedProposal): PipelineProposal | null {
  const parsed = pipelineProposalSchema.safeParse(entry);
  return parsed.success ? parsed.data : null;
}

// describeWorkspace states the run's checkout choice for review and proposal
// summaries; an absent selection is the project checkout (FS-14.R91). Review
// passes the server's plan so it shows the resolved checkout (FS-14.R113).
export function describeWorkspace(workspace: PipelineWorkspaceSelection | undefined, plan?: PipelineWorkspacePreview) {
  if (plan?.mode === "project") return `Project checkout · ${plan.cwd}${plan.branch ? ` on ${plan.branch}` : ""}`;
  if (plan?.mode === "new_worktree") return `New worktree · branch ${plan.branch} from ${plan.base}`;
  if (!workspace || workspace.mode !== "new_worktree") return "Project checkout";
  return `New worktree · branch ${workspace.branch ?? ""}${workspace.base ? ` from ${workspace.base}` : ""}`;
}

export function proposalKindLabel(kind: string) {
  if (kind === "save_template") return "Save template";
  if (kind === "start_run") return "Start run";
  return kind;
}

// summarizeProposal states what the offer asks for. A start proposal's durable
// payload names its template by id alone, so the title is resolved from that
// template as it stands now: a rename follows, and a template deleted since the
// proposal was made says so instead of hiding the offer (FS-14.R51).
export function summarizeProposal(
  proposal: PipelineProposal,
  templates: PipelineTemplateRecord[] | undefined,
): ProposalSummaryField[] {
  if (proposal.kind === "save_template") {
    const stages = proposal.payload.template.stages.length;
    return [
      { label: "Template", value: clipProposalValue(proposal.payload.template.title || "Untitled pipeline") },
      { label: "Stages", value: `${stages} stage${stages === 1 ? "" : "s"}` },
    ];
  }
  return [
    { label: "Template", value: resolveTemplateTitle(proposal.payload.template_id, templates) },
    { label: "Run", value: clipProposalValue(proposal.payload.display_name) },
    { label: "Goal", value: clipProposalValue(proposal.payload.goal) },
    ...(proposal.payload.workspace ? [{ label: "Workspace", value: clipProposalValue(describeWorkspace(proposal.payload.workspace)) }] : []),
  ];
}

function resolveTemplateTitle(templateID: string, templates: PipelineTemplateRecord[] | undefined) {
  // Until the library has loaded, claiming the template is gone would be worse than a neutral label.
  if (!templates) return "Pipeline template";
  const record = templates.find((entry) => entry.id === templateID);
  if (!record) return "Template is gone";
  return clipProposalValue(record.template.title || "Untitled pipeline");
}
