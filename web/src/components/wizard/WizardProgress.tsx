import type { OnboardingStepId } from "@/lib/api";

interface WizardProgressProps {
  steps: Array<{ id: OnboardingStepId; label: string }>;
  currentStepIndex: number;
  completedSteps: OnboardingStepId[];
  skippedSteps: OnboardingStepId[];
}

function statusFor(
  index: number,
  stepId: OnboardingStepId,
  currentStepIndex: number,
  completedSteps: OnboardingStepId[],
  skippedSteps: OnboardingStepId[],
) {
  if (completedSteps.includes(stepId)) return "completed";
  if (skippedSteps.includes(stepId)) return "skipped";
  if (index === currentStepIndex) return "active";
  if (index < currentStepIndex) return "past";
  return "upcoming";
}

export default function WizardProgress({
  steps,
  currentStepIndex,
  completedSteps,
  skippedSteps,
}: WizardProgressProps) {
  return (
    <ol className="mb-6 grid grid-cols-1 gap-3 sm:grid-cols-5">
      {steps.map((step, idx) => {
        const status = statusFor(
          idx,
          step.id,
          currentStepIndex,
          completedSteps,
          skippedSteps,
        );

        const tone =
          status === "completed"
            ? "border-green-300 bg-green-50 text-green-700"
            : status === "skipped"
              ? "border-yellow-300 bg-yellow-50 text-yellow-700"
              : status === "active"
                ? "border-indigo-300 bg-indigo-50 text-indigo-700"
                : status === "past"
                  ? "border-gray-300 bg-gray-50 text-gray-600"
                  : "border-gray-200 bg-white text-gray-500";

        return (
          <li
            key={step.id}
            className={`rounded-lg border px-3 py-2 text-xs font-medium sm:text-sm ${tone}`}
          >
            <span className="mr-2 inline-flex h-5 w-5 items-center justify-center rounded-full border border-current text-[11px]">
              {idx + 1}
            </span>
            {step.label}
          </li>
        );
      })}
    </ol>
  );
}
