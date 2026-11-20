import type { SVGProps } from "react";

// Inline action icons (FS-02.R67, TS-08.R89). They inherit `currentColor` and are decorative:
// the button that holds one carries the accessible name and tooltip.
function Icon({ children, ...props }: SVGProps<SVGSVGElement>) {
  return (
    <svg className="ad-icon" viewBox="0 0 16 16" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true" focusable="false" {...props}>
      {children}
    </svg>
  );
}

export function SendIcon() {
  return <Icon><path d="M8 13V3M3.5 7.5 8 3l4.5 4.5" /></Icon>;
}

export function BackIcon() {
  return <Icon><path d="M13 8H3m4-4L3 8l4 4" /></Icon>;
}

export function StopIcon() {
  return <Icon><rect x="4" y="4" width="8" height="8" rx="1" fill="currentColor" stroke="none" /></Icon>;
}

export function CollapseIcon() {
  return <Icon><path d="M4 10 8 6l4 4" /></Icon>;
}

export function ExpandIcon() {
  return <Icon><path d="M4 6l4 4 4-4" /></Icon>;
}

export function GripIcon() {
  return <Icon fill="currentColor" stroke="none"><circle cx="6" cy="4" r="1.1" /><circle cx="10" cy="4" r="1.1" /><circle cx="6" cy="8" r="1.1" /><circle cx="10" cy="8" r="1.1" /><circle cx="6" cy="12" r="1.1" /><circle cx="10" cy="12" r="1.1" /></Icon>;
}

export function FileIcon() {
  return <Icon><path d="M9.5 2H4v12h8V4.5zM9.5 2v3H12M6 9h4m-4 2.5h4" /></Icon>;
}

export function CommandIcon() {
  return <Icon><path d="m3.5 5 3 3-3 3m5 0h4" /></Icon>;
}

export function PauseIcon() {
  return <Icon><path d="M6 3.5v9m4-9v9" /></Icon>;
}

export function RoomIcon() {
  return <Icon><path d="M2.5 2.5h11v8.5H6l-3.5 3z" /></Icon>;
}

export function CollapseAllIcon() {
  return <Icon><path d="M4 8.5 8 4.5l4 4M4 12.5l4-4 4 4" /></Icon>;
}

export function CloseIcon() {
  return <Icon><path d="m4.5 4.5 7 7m0-7-7 7" /></Icon>;
}

export function CopyIcon() {
  return <Icon><rect x="5.5" y="5.5" width="8" height="8" rx="1.5" /><path d="M10.5 3.5v-.5a1 1 0 0 0-1-1h-6a1 1 0 0 0-1 1v6a1 1 0 0 0 1 1h.5" /></Icon>;
}

export function CheckIcon() {
  return <Icon><path d="m3.5 8.5 3 3 6-7" /></Icon>;
}

export function ForkIcon() {
  return <Icon><circle cx="4.5" cy="3.5" r="1.5" /><circle cx="11.5" cy="3.5" r="1.5" /><circle cx="4.5" cy="12.5" r="1.5" /><path d="M4.5 5v6M11.5 5v1a2.5 2.5 0 0 1-2.5 2.5H7A2.5 2.5 0 0 0 4.5 11" /></Icon>;
}

export function MoveLeftIcon() {
  return <Icon><path d="M10 4 6 8l4 4" /></Icon>;
}

export function MoveRightIcon() {
  return <Icon><path d="m6 4 4 4-4 4" /></Icon>;
}
