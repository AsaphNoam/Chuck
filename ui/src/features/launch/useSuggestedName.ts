import { useEffect, useRef, useState } from "react";

function capitalize(s: string): string {
  if (!s) return s;
  return s.charAt(0).toUpperCase() + s.slice(1);
}

function suggest(role: string): string {
  // The default agent name is just the role (FS-01.R1 only requires that the
  // modal auto-suggests some name; the user can always override it).
  // Empty persona creation uses the server's ordinary curated suggestion,
  // rather than exposing its internal role id as the agent's name.
  if (!role || role === "default") return "";
  return capitalize(role);
}

export function useSuggestedName(
  role: string,
  /** A prefilled name (an edited launch draft) is kept like a typed one. */
  initial?: string,
): [string, (v: string) => void] {
  const [name, setName] = useState(() => initial ?? suggest(role));
  const dirtyRef = useRef(initial !== undefined);

  useEffect(() => {
    if (!dirtyRef.current) {
      setName(suggest(role));
    }
  }, [role]);

  const handleChange = (v: string) => {
    dirtyRef.current = true;
    setName(v);
  };

  return [name, handleChange];
}
