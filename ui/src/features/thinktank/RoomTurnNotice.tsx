import { Link } from "react-router-dom";
import { useAgentThinkTanks } from "../../api/thinkTanks";
import { agentConversationPath } from "../../lib/agentConversation";
import { useProjects } from "../../api/config";
import type { ThinkTankSummary } from "../../schemas/thinkTank";
import { phaseLabel, roomTitle, thinkTankPath } from "./roomText";

/** RoomCue is the compact Think Tank identification under an agent's
 *  ordinary header: the room title and whether this agent is taking a room
 *  turn now. Full goal, membership and the Send/Steer explanation live in the
 *  Think Tank tab (FS-03.R69, R71, FS-21.R45). It renders nothing for an
 *  agent in no room. */
export function RoomCue({ agentId }: { agentId: string }) {
  const rooms = useAgentThinkTanks(agentId);
  const list = rooms.data?.rooms ?? [];
  if (list.length === 0) return null;
  const turnIn = list.find((room) => room.active_attempts.some((a) => a.agent_id === agentId));
  const shown = turnIn ?? list.find((room) => room.phase !== "ended") ?? list[0];
  const more = list.length - 1;
  return (
    <p className="room-turn-notice" data-ui="think-tank" data-slot="turn-notice" data-state={turnIn ? "active" : "settled"} role="status">
      <span>Think Tank</span>
      <Link to={thinkTankPath(shown.room_id)}>{roomTitle(shown)}</Link>
      <span>{turnIn ? "Taking a room turn — Steer shapes it privately; Send waits for it to end" : phaseLabel(shown.phase)}</span>
      {more > 0 && <span>+{more} more in the Think Tank tab</span>}
    </p>
  );
}

/** ThinkTankTab is compact navigation for every room this agent belongs to.
 *  Room state, goals and allowances remain on the room and member
 *  destinations (FS-21.R57, FS-03.R71). */
export function ThinkTankTab({ agentId }: { agentId: string }) {
  const rooms = useAgentThinkTanks(agentId);
  if (rooms.isError) return <p className="form-error">Could not load this agent's Think Tanks.</p>;
  if (rooms.isLoading) return <p>Loading Think Tanks…</p>;
  const list = rooms.data?.rooms ?? [];
  if (list.length === 0) return <p>This agent is not in a Think Tank.</p>;
  return (
    <div className="think-tank-tab" data-ui="think-tank" data-slot="membership">
      {list.map((room) => <Membership key={room.room_id} room={room} agentId={agentId} />)}
      {rooms.data?.clipped && <p role="status">Showing the {list.length} most recent Think Tanks; older ones are not listed here.</p>}
    </div>
  );
}

function Membership({ room, agentId }: { room: ThinkTankSummary; agentId: string }) {
  const projects = useProjects();
  const linked = room.roster.filter((member) => member.agent_id !== agentId);
  const duplicateNames = new Set(linked.filter((member, index) => linked.some((other, otherIndex) => otherIndex !== index && other.name === member.name)).map((member) => member.name));
  const labelCounts = new Map<string, number>();
  const projectLabel = (project: string) => projects.data?.[project]?.title ?? project;
  const memberLabel = (member: ThinkTankSummary["roster"][number]) => {
    if (!duplicateNames.has(member.name)) return member.name;
    const base = member.name + " · " + projectLabel(member.project);
    const count = (labelCounts.get(base) ?? 0) + 1;
    labelCounts.set(base, count);
    return count === 1 ? base : base + " · " + member.agent_id;
  };
  const active = new Set(room.active_attempts.map((attempt) => attempt.agent_id));
  return (
    <section className="think-tank-membership" aria-label={roomTitle(room)}>
      <nav className="think-tank-membership-nav" aria-label={roomTitle(room) + " navigation"}>
        <Link className="think-tank-room-link" to={thinkTankPath(room.room_id)}>{roomTitle(room)}</Link>
        <div className="think-tank-member-links">
          {linked.map((member) => {
            const isActive = active.has(member.agent_id);
            const label = memberLabel(member);
            if (!member.exists) return <span className="think-tank-member-unavailable" key={member.agent_id}>{label + " (unavailable)"}</span>;
            return (
              <Link
                className={"ad-button ad-button-secondary think-tank-member-link" + (isActive ? " is-active" : "")}
                data-state={isActive ? "active" : "settled"}
                key={member.agent_id}
                to={agentConversationPath(member.agent_id)}
                aria-label={isActive ? label + ", taking a room turn" : label}
              >
                {label}
              </Link>
            );
          })}
        </div>
      </nav>
    </section>
  );
}
