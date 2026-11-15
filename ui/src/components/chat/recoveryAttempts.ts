const prefix = "chuck-recovery-send:";
const maxAttempts = 20;

function attemptKeys() {
  return Object.keys(localStorage).filter((key) => key.startsWith(prefix));
}

export function saveRecoveryAttempt(agentId: string, requestId: string, attemptId: string) {
  const key = `${prefix}${agentId}:${requestId}`;
  const existing = localStorage.getItem(key);
  if (existing) return existing;
  localStorage.setItem(key, attemptId);
  const older = attemptKeys().filter((candidate) => candidate !== key);
  older.slice(0, Math.max(0, older.length + 1 - maxAttempts)).forEach((candidate) => localStorage.removeItem(candidate));
  return attemptId;
}

export function recoveryAttempts(agentId: string) {
  const owner = `${prefix}${agentId}:`;
  const keys = attemptKeys();
  const excess = Math.max(0, keys.length - maxAttempts);
  keys.slice(0, excess).forEach((key) => localStorage.removeItem(key));
  return keys.slice(excess).filter((key) => key.startsWith(owner))
    .map((key) => [key.slice(owner.length), localStorage.getItem(key)] as const);
}

export function clearRecoveryAttempt(agentId: string, requestId: string, attemptId?: string | null) {
  const key = `${prefix}${agentId}:${requestId}`;
  if (attemptId === undefined || localStorage.getItem(key) === attemptId) localStorage.removeItem(key);
}
