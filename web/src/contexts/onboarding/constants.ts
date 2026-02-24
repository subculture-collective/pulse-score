import type { OnboardingStepId } from "@/lib/api";

export const ONBOARDING_STEPS: OnboardingStepId[] = [
  "welcome",
  "stripe",
  "hubspot",
  "intercom",
  "preview",
];

export const ONBOARDING_RESUME_STEP_STORAGE_KEY = "onboarding:resume-step";

export function stepIdToIndex(stepId: OnboardingStepId | string): number {
  const idx = ONBOARDING_STEPS.findIndex((step) => step === stepId);
  return idx >= 0 ? idx : 0;
}

export function stepIndexToId(index: number): OnboardingStepId {
  if (index < 0) return ONBOARDING_STEPS[0];
  if (index >= ONBOARDING_STEPS.length) {
    return ONBOARDING_STEPS[ONBOARDING_STEPS.length - 1];
  }
  return ONBOARDING_STEPS[index];
}

export function clampStepIndex(index: number): number {
  if (Number.isNaN(index) || !Number.isFinite(index)) return 0;
  return Math.max(0, Math.min(index, ONBOARDING_STEPS.length - 1));
}
