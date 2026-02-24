import {
  createContext,
  useContext,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import type { OnboardingStatus, OnboardingStepId } from "@/lib/api";
import {
  ONBOARDING_STEPS,
  clampStepIndex,
  stepIdToIndex,
} from "@/contexts/onboarding/constants";

interface OnboardingContextValue {
  status: OnboardingStatus | null;
  currentStepIndex: number;
  setCurrentStepIndex: (index: number) => void;
  hydrateFromStatus: (
    status: OnboardingStatus,
    preferredStepIndex?: number,
  ) => void;
  markCompleted: (stepId: OnboardingStepId) => void;
  markSkipped: (stepId: OnboardingStepId) => void;
  setStepPayload: (
    stepId: OnboardingStepId,
    payload: Record<string, unknown>,
  ) => void;
  setCompletedAt: (completedAt: string | null) => void;
}

const OnboardingContext = createContext<OnboardingContextValue | null>(null);

function dedupeSteps(
  steps: OnboardingStepId[],
  target: OnboardingStepId,
): OnboardingStepId[] {
  if (steps.includes(target)) return steps;
  return [...steps, target];
}

function removeStep(
  steps: OnboardingStepId[],
  target: OnboardingStepId,
): OnboardingStepId[] {
  return steps.filter((step) => step !== target);
}

export function OnboardingProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<OnboardingStatus | null>(null);
  const [currentStepIndex, setCurrentStepIndexState] = useState(0);

  function setCurrentStepIndex(index: number) {
    setCurrentStepIndexState(clampStepIndex(index));
  }

  function hydrateFromStatus(
    nextStatus: OnboardingStatus,
    preferredStepIndex?: number,
  ) {
    setStatus(nextStatus);
    if (typeof preferredStepIndex === "number") {
      setCurrentStepIndexState(clampStepIndex(preferredStepIndex));
      return;
    }
    setCurrentStepIndexState(stepIdToIndex(nextStatus.current_step));
  }

  function markCompleted(stepId: OnboardingStepId) {
    setStatus((prev) => {
      if (!prev) return prev;
      return {
        ...prev,
        completed_steps: dedupeSteps(prev.completed_steps, stepId),
        skipped_steps: removeStep(prev.skipped_steps, stepId),
      };
    });
  }

  function markSkipped(stepId: OnboardingStepId) {
    setStatus((prev) => {
      if (!prev) return prev;
      return {
        ...prev,
        skipped_steps: dedupeSteps(prev.skipped_steps, stepId),
        completed_steps: removeStep(prev.completed_steps, stepId),
      };
    });
  }

  function setStepPayload(
    stepId: OnboardingStepId,
    payload: Record<string, unknown>,
  ) {
    setStatus((prev) => {
      if (!prev) return prev;
      return {
        ...prev,
        step_payloads: {
          ...(prev.step_payloads ?? {}),
          [stepId]: payload,
        },
      };
    });
  }

  function setCompletedAt(completedAt: string | null) {
    setStatus((prev) => {
      if (!prev) return prev;
      return { ...prev, completed_at: completedAt };
    });
  }

  const value = useMemo<OnboardingContextValue>(
    () => ({
      status,
      currentStepIndex,
      setCurrentStepIndex,
      hydrateFromStatus,
      markCompleted,
      markSkipped,
      setStepPayload,
      setCompletedAt,
    }),
    [status, currentStepIndex],
  );

  return (
    <OnboardingContext.Provider value={value}>
      {children}
    </OnboardingContext.Provider>
  );
}

export function useOnboarding() {
  const ctx = useContext(OnboardingContext);
  if (!ctx) {
    throw new Error("useOnboarding must be used inside OnboardingProvider");
  }
  return ctx;
}

export function isFinalOnboardingStep(stepId: OnboardingStepId) {
  return stepId === ONBOARDING_STEPS[ONBOARDING_STEPS.length - 1];
}
