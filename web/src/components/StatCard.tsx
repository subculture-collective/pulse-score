import { TrendingUp, TrendingDown, Minus } from "lucide-react";
import type { ReactNode } from "react";

interface StatCardProps {
  title: string;
  value: string | number;
  trend?: number;
  trendDirection?: "up" | "down" | "neutral";
  icon?: ReactNode;
}

const trendLabels = {
  up: "increased",
  down: "decreased",
  neutral: "unchanged",
};

export default function StatCard({
  title,
  value,
  trend,
  trendDirection = "neutral",
  icon,
}: StatCardProps) {
  const trendColor =
    trendDirection === "up"
      ? "text-green-600 dark:text-green-400"
      : trendDirection === "down"
        ? "text-red-600 dark:text-red-400"
        : "text-gray-500 dark:text-gray-400";

  const TrendIcon =
    trendDirection === "up"
      ? TrendingUp
      : trendDirection === "down"
        ? TrendingDown
        : Minus;

  return (
    <div className="rounded-lg border border-gray-200 bg-white p-6 dark:border-gray-700 dark:bg-gray-900">
      <div className="flex items-center justify-between">
        <p className="text-sm font-medium text-gray-500 dark:text-gray-400">
          {title}
        </p>
        {icon && (
          <span className="text-gray-400 dark:text-gray-500" aria-hidden="true">
            {icon}
          </span>
        )}
      </div>
      <p className="mt-2 text-3xl font-bold text-gray-900 dark:text-gray-100">
        {value}
      </p>
      {trend !== undefined && (
        <div className={`mt-2 flex items-center gap-1 text-sm ${trendColor}`}>
          <TrendIcon className="h-4 w-4" aria-hidden="true" />
          <span>{Math.abs(trend)}%</span>
          <span className="text-gray-500 dark:text-gray-400">vs last period</span>
          <span className="sr-only">{trendLabels[trendDirection]}</span>
        </div>
      )}
    </div>
  );
}
