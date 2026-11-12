import { useState } from "react";
import { useRetainedCommand, useSaveThinkTankTemplate } from "../../api/thinkTanks";
import { Button, ConfirmDialog } from "../../components/ui";

/** RoomSaveSetup saves an active or ended room's participants and discussion
 *  settings as a template, including raised ceilings and without its title,
 *  goal, files or history (FS-21.R60). The room stays open; a failed save
 *  keeps the name for retry. */
export function RoomSaveSetup({ roomID, title }: { roomID: string; title: string }) {
  const save = useSaveThinkTankTemplate();
  const command = useRetainedCommand();
  const [open, setOpen] = useState(false);
  const [name, setName] = useState("");
  const [error, setError] = useState("");
  const [note, setNote] = useState("");
  const submit = () => {
    setError("");
    if (!name.trim()) {
      setError("Name the template.");
      return;
    }
    save.mutate({ command_id: command.idFor({ name, roomID }), name, room_id: roomID }, {
      onSuccess: (saved) => { command.acknowledge(); setOpen(false); setNote(`Saved “${saved.name}”.`); },
      onError: (err) => setError(err instanceof Error ? `The setup could not be saved: ${err.message}` : "The setup could not be saved."),
    });
  };
  return (
    <>
      <Button type="button" onClick={() => { setOpen(true); setNote(""); setError(""); setName((n) => n || title); }}>Save setup</Button>
      {note && <span className="think-tank-count" role="status">{note}</span>}
      <ConfirmDialog
        open={open}
        title="Save this setup as a template"
        confirmLabel="Save template"
        pending={save.isPending}
        onCancel={() => setOpen(false)}
        onConfirm={submit}
      >
        <div className="form-field">
          <label htmlFor="think-tank-save-name">Template name</label>
          <input id="think-tank-save-name" value={name} maxLength={120} onChange={(e) => setName(e.target.value)}
            onKeyDown={(e) => { if (e.key === "Enter") { e.preventDefault(); submit(); } }} />
        </div>
        <p className="form-hint">Saves participants' launch settings, turn limits, instructions and discussion settings. The title, goal, files and discussion stay with this room.</p>
        {error && <p className="form-error" role="alert">{error}</p>}
      </ConfirmDialog>
    </>
  );
}
