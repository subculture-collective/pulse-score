interface HealthScoreBadgeProps {
  score: number;
  riskLevel?: "green" | "yellow" | "red";
  size?: "sm" | "md" | "lg";
  showLabel?: boolean;
}

const sizeClasses = {
  sm: "h-6 w-6 text-xs",
  md: "h-8 w-8 text-sm",
  lg: "h-12 w-12 text-lg",
};

const colorClasses = {
  green: "bg-green-500 dark:bg-green-600",
  yellow: "bg-yellow-500 dark:bg-yellow-600",
  red: "bg-red-500 dark:bg-red-600",
};

const riskLabels = {
  green: "Healthy",
  yellow: "At Risk",
  red: "Critical",
};

function deriveRiskLevel(score: number): "green" | "yellow" | "red" {
  if (score >= 70) return "green";
  if (score >= 40) return "yellow";
  return "red";
}

export default function HealthScoreBadge({
  score,
  riskLevel,
  size = "md",
  showLabel = false,
}: HealthScoreBadgeProps) {
  const level = riskLevel ?? deriveRiskLevel(score);
  const label = riskLabels[level];

  return (
    <span
      className="inline-flex items-center gap-2"
      role="img"
      aria-label={`Health score: ${score}, ${label}`}
    >
      <span
        className={`inline-flex items-center justify-center rounded-full font-semibold text-white ${sizeClasses[size]} ${colorClasses[level]}`}
      >
        {score}
      </span>
      {showLabel && (
        <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
          {label}
        </span>
      )}
      {!showLabel && <span className="sr-only">{label}</span>}
    </span>
  );
}
