import { useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { openAgentConversation, registerConversationNavigator } from "../../lib/agentConversation";
import { OUTCOME_TOAST_MS, useUiStore, type ToastItem } from "../../store/uiStore";

// Toast owns its own auto-dismiss timer (6s unless the toast names a shorter
// life), keyed to its id, so a newly pushed toast doesn't restart the timers of
// older ones (the previous single effect depended on the whole toasts array,
// resetting every timer on each new toast so older toasts lingered).
function Toast({ toast, dismiss }: { toast: ToastItem; dismiss: (id: string) => void }) {
  const life = toast.durationMs ?? 6_000;
  useEffect(() => {
    const timer = window.setTimeout(() => dismiss(toast.id), life);
    return () => window.clearTimeout(timer);
  }, [toast.id, dismiss, life]);

  // An agent toast opens that agent's conversation (FS-02.R64); an error toast
  // names no agent and only dismisses.
  const open = () => {
    if (toast.agentId) openAgentConversation(toast.agentId);
    dismiss(toast.id);
  };

  return (
    <div className={`toast ${toast.type}`} data-ui="toast" data-state={toast.type}
      data-fade={toast.durationMs === OUTCOME_TOAST_MS ? "" : undefined}>
      <button type="button" data-slot="open" onClick={open}>
        <strong data-slot="title">{toast.title}</strong>
        {toast.body && <span data-slot="body">{toast.body}</span>}
      </button>
      <button type="button" className="ad-button-icon" data-slot="close" aria-label="Dismiss notification" onClick={() => dismiss(toast.id)}>
        ×
      </button>
    </div>
  );
}

export function NotificationCenter() {
  const toasts = useUiStore((state) => state.toasts);
  const dismiss = useUiStore((state) => state.dismissToast);
  const navigate = useNavigate();

  // Desktop notifications fire from the SSE client, outside the router tree.
  useEffect(() => {
    registerConversationNavigator(navigate);
    return () => registerConversationNavigator(null);
  }, [navigate]);

  if (toasts.length === 0) return null;
  return (
    <div className="toast-stack" role="status" aria-live="polite">
      {toasts.map((toast) => (
        <Toast key={toast.id} toast={toast} dismiss={dismiss} />
      ))}
    </div>
  );
}
