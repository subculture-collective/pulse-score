interface HealthScoreBadgeProps {
  score: number;
  riskLevel?: "green" | "yellow" | "red";
  size?: "sm" | "md" | "lg";
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

function deriveRiskLevel(score: number): "green" | "yellow" | "red" {
  if (score >= 70) return "green";
  if (score >= 40) return "yellow";
  return "red";
}

export default function HealthScoreBadge({
  score,
  riskLevel,
  size = "md",
}: HealthScoreBadgeProps) {
  const level = riskLevel ?? deriveRiskLevel(score);

  return (
    <span
      className={`inline-flex items-center justify-center rounded-full font-semibold text-white ${sizeClasses[size]} ${colorClasses[level]}`}
      title={`Health score: ${score}`}
    >
      {score}
    </span>
  );
}
