import type { RoleResponse } from "../schemas/role";

export type PersonaEntry = readonly [id: string, role: Omit<RoleResponse, "role">];

/** The empty, aware internal role is presented as the ordinary No persona choice. */
export function isCanonicalNoPersona(id: string, role?: Partial<RoleResponse>): boolean {
  return id === "default" && role?.system_prompt === "" && role?.skip_permissions == null && role?.chuck_aware !== false;
}

export function personaLabel(id: string, role?: Partial<RoleResponse>): string {
  return isCanonicalNoPersona(id, role) ? "No persona" : role?.title || id;
}

export function visiblePersonaEntries<T extends Omit<RoleResponse, "role">>(roles: Record<string, T>): [string, T][] {
  return Object.entries(roles).filter(([id, role]) => !isCanonicalNoPersona(id, role));
}

export function resolvedChuckAware(role?: Pick<RoleResponse, "chuck_aware"> | null): boolean {
  return role?.chuck_aware !== false;
}

export function personaIdentityLabel(id: string, role?: Partial<RoleResponse>): string {
  // Keep established ID labels for named roles; the internal empty choice is
  // labelled only after its definition is known, so custom content stays honest.
  return id === "default" && role ? personaLabel(id, role) : id;
}
