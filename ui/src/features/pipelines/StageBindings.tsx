import type { PipelineTemplate } from "../../schemas/pipeline";
import { newPipelineKey, pipelineStageLabels } from "./pipelineLabels";

type EditTemplate = (change: (draft: PipelineTemplate) => void) => void;

export function StageBindings({ draft, stageIndex, mutate }: { draft: PipelineTemplate; stageIndex: number; mutate: EditTemplate }) {
  const stage = draft.stages[stageIndex];
  const path = `stages.${stageIndex}`;
  const labels = pipelineStageLabels(draft.stages);
  const sources = new Map(draft.inputs.filter(({ name }) => name.trim()).map(({ name }) => [name, {
    name,
    label: `Run input · ${name}`,
  }]));
  for (const producer of draft.stages.slice(0, stageIndex)) {
    for (const output of producer.outputs) {
      if (output.value && output.name.trim()) {
        sources.set(output.value, { name: output.name, label: `${labels.get(producer.id)} · ${output.name}` });
      }
    }
  }

  return (
    <details className="pipeline-disclosure" data-template-field={`${path}.handoffs`}>
      <summary>Structured handoffs{stage.inputs.length + stage.outputs.length > 0 && <span>{stage.inputs.length + stage.outputs.length}</span>}</summary>
      <div className="pipeline-disclosure-body">
        <p>Use handoffs when Chuck needs to check that a named value is available. Stage instructions can refer directly to files and earlier work.</p>
        <div className="pipeline-bindings" data-template-field={`${path}.outputs`}>
          <div className="pipeline-subsection-header">
            <h4>{stage.coordination === "think_tank" ? "Judge’s synthesis" : "Outputs from this stage"}</h4>
            <button type="button" disabled={stage.coordination === "think_tank" && stage.outputs.length === 1} onClick={() => {
              const value = newPipelineKey("value");
              mutate((next) => { next.stages[stageIndex].outputs.push({ name: "", value, description: "" }); });
            }}>Add output</button>
          </div>
          {stage.outputs.map((output, index) => (
            <div className="pipeline-inline-row" key={output.value}>
              <label className="form-field"><span>Output name</span><input value={output.name} data-template-field={`${path}.outputs.${index}.name`} onChange={(event) => mutate((next) => { next.stages[stageIndex].outputs[index].name = event.target.value; })} /></label>
              <label className="form-field"><span>Description</span><input value={output.description} data-template-field={`${path}.outputs.${index}.description`} onChange={(event) => mutate((next) => { next.stages[stageIndex].outputs[index].description = event.target.value; })} /></label>
              <button type="button" className="ad-button-danger" onClick={() => mutate((next) => { next.stages[stageIndex].outputs.splice(index, 1); })}>Remove output</button>
            </div>
          ))}
        </div>
        <div className="pipeline-bindings" data-template-field={`${path}.inputs`}>
          <div className="pipeline-subsection-header">
            <h4>Inputs to this stage</h4>
            <button type="button" disabled={sources.size === 0} onClick={() => mutate((next) => {
              next.stages[stageIndex].inputs.push({ name: "", value: "", required: true });
            })}>Add handoff</button>
          </div>
          {sources.size === 0 && <p>Add a run input or name an output on an earlier stage to use it here.</p>}
          {stage.inputs.map((input, index) => (
            <div className="pipeline-inline-row" key={index}>
              <label className="form-field"><span>Source</span>
                <select value={input.value} data-template-field={`${path}.inputs.${index}.value`} onChange={(event) => {
                  const value = event.target.value;
                  mutate((next) => {
                    const target = next.stages[stageIndex].inputs[index];
                    target.value = value;
                    if (!target.name) target.name = sources.get(value)?.name ?? "";
                  });
                }}>
                  <option value="">Choose source</option>
                  {input.value && !sources.has(input.value) && <option value={input.value}>Unavailable source</option>}
                  {[...sources].map(([key, source]) => <option key={key} value={key}>{source.label}</option>)}
                </select>
              </label>
              <label className="form-field"><span>Input name</span><input value={input.name} data-template-field={`${path}.inputs.${index}.name`} onChange={(event) => mutate((next) => { next.stages[stageIndex].inputs[index].name = event.target.value; })} /></label>
              <label className="pipeline-check"><input type="checkbox" checked={input.required} onChange={(event) => mutate((next) => { next.stages[stageIndex].inputs[index].required = event.target.checked; })} /> Required before entry</label>
              <button type="button" className="ad-button-danger" onClick={() => mutate((next) => { next.stages[stageIndex].inputs.splice(index, 1); })}>Remove handoff</button>
            </div>
          ))}
        </div>
      </div>
    </details>
  );
}
