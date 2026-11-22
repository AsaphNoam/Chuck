import { useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useConfig } from "../../api/config";
import { OnboardingWizard } from "./OnboardingWizard";

interface OnboardingGateProps {
  children: React.ReactNode;
}

export function OnboardingGate({ children }: OnboardingGateProps) {
  const { data: config, isLoading } = useConfig();
  const navigate = useNavigate();
  const [dismissed, setDismissed] = useState(false);
  const wizardOpened = useRef(false);

  if (isLoading) return <div className="gate-loading">Loading…</div>;

  // Only the saved completion flag ends first-run setup (FS-04.R23). A ready
  // backend and a project are progress, not completion: a reload mid-wizard
  // reopens it at the first step the server does not report done.
  if (config && config.onboarding_complete !== true) wizardOpened.current = true;

  // Keep an opened wizard mounted until Launch or Set up later completes so a
  // config poll cannot eject the user mid-flow.
  if (config && !dismissed && wizardOpened.current) {
    return (
      <OnboardingWizard
        steps={config.onboarding.steps}
        onComplete={(launchedAgentId) => {
          setDismissed(true);
          // A completed Launch opens the first agent's chat (FS-04.R20).
          if (launchedAgentId) navigate(`/agent/${launchedAgentId}`);
        }}
      />
    );
  }

  return <>{children}</>;
}
