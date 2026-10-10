import { describe, expect, it } from "vitest";
import { isCanonicalNoPersona, personaLabel, personaIdentityLabel, resolvedChuckAware, visiblePersonaEntries } from "./personas";
import { roleResponseSchema } from "../schemas/role";

// FS-04.A36: only the canonical definition is hidden and labelled No persona.
describe("persona presentation", () => {
  it("labels only the canonical empty default as No persona", () => {
    const empty = { title: "Default", system_prompt: "", skip_permissions: null, chuck_aware: true };
    expect(isCanonicalNoPersona("default", empty)).toBe(true);
    expect(personaLabel("default", empty)).toBe("No persona");
    expect(isCanonicalNoPersona("default", { ...empty, system_prompt: "custom" })).toBe(false);
    expect(isCanonicalNoPersona("default", { ...empty, chuck_aware: false })).toBe(false);
    expect(isCanonicalNoPersona("default", { ...empty, skip_permissions: false })).toBe(false);
    expect(personaIdentityLabel("default")).toBe("default");
    expect(personaIdentityLabel("default", empty)).toBe("No persona");
    expect(personaIdentityLabel("default", { ...empty, title:"Custom", system_prompt:"custom" })).toBe("Custom");
  });

  // FS-04.A36: decoder preserves false and rejects non-boolean values.
  it("decodes true, false and legacy role choices without truthy coercion", () => {
    const base = { role:"custom", title:"Custom", system_prompt:"" };
    for (const value of [true, false, null, undefined]) {
      const decoded = roleResponseSchema.parse({ ...base, chuck_aware:value });
      expect(decoded.chuck_aware).toBe(value);
    }
    for (const value of ["false", 0, [], {}]) {
      expect(roleResponseSchema.safeParse({ ...base, chuck_aware:value }).success).toBe(false);
    }
  });

  it("hides the canonical default from persona management and defaults awareness on", () => {
    const roles = {
      default: { title: "Default", system_prompt: "", skip_permissions: null, chuck_aware: true },
      reviewer: { title: "Reviewer", system_prompt: "review", skip_permissions: null },
    };
    expect(visiblePersonaEntries(roles).map(([id]) => id)).toEqual(["reviewer"]);
    expect(resolvedChuckAware({})).toBe(true);
    expect(resolvedChuckAware({ chuck_aware: false })).toBe(false);
  });
});
