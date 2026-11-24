export function AnnotationSourceCaption({ label, kind }: { label: string; kind?: "file" | "event" }) {
  const prefix = kind === "file" ? "File " : kind === "event" ? "Event " : "";
  return <>{prefix}<code>{label.slice(prefix.length)}</code></>;
}
