import { useEffect, useState } from "react";
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  Cell,
} from "recharts";
import api from "@/lib/api";
import ChartSkeleton from "@/components/skeletons/ChartSkeleton";
import EmptyState from "@/components/EmptyState";
import { BarChart3 } from "lucide-react";

interface BucketData {
  range: string;
  count: number;
  min_score: number;
  max_score: number;
}

function getBarColor(minScore: number): string {
  if (minScore >= 70) return "#22c55e"; // green
  if (minScore >= 40) return "#eab308"; // yellow
  return "#ef4444"; // red
}

export default function ScoreDistributionChart() {
  const [data, setData] = useState<BucketData[]>([]);
  const [loading, setLoading] = useState(true);
  const [empty, setEmpty] = useState(false);

  useEffect(() => {
    async function fetch() {
      try {
        const { data: res } = await api.get("/dashboard/score-distribution");
        const buckets = res.buckets ?? res.distribution ?? res ?? [];
        if (Array.isArray(buckets) && buckets.length > 0) {
          setData(buckets);
        } else {
          setEmpty(true);
        }
      } catch {
        setEmpty(true);
      } finally {
        setLoading(false);
      }
    }
    fetch();
  }, []);

  if (loading) return <ChartSkeleton />;

  if (empty) {
    return (
      <div className="rounded-lg border border-gray-200 bg-white p-6 dark:border-gray-700 dark:bg-gray-900">
        <EmptyState
          icon={<BarChart3 className="h-12 w-12" />}
          title="No score data yet"
          description="Health scores will appear here once customers are scored."
        />
      </div>
    );
  }

  return (
    <div className="rounded-lg border border-gray-200 bg-white p-6 dark:border-gray-700 dark:bg-gray-900">
      <h3 className="mb-4 text-sm font-medium text-gray-900 dark:text-gray-100">
        Health Score Distribution
      </h3>
      <ResponsiveContainer width="100%" height={280}>
        <BarChart data={data}>
          <XAxis dataKey="range" tick={{ fontSize: 12 }} stroke="#9ca3af" />
          <YAxis
            tick={{ fontSize: 12 }}
            stroke="#9ca3af"
            allowDecimals={false}
          />
          <Tooltip
            contentStyle={{
              backgroundColor: "var(--color-white, #fff)",
              borderColor: "#e5e7eb",
              borderRadius: 8,
            }}
            formatter={(value) => [value, "Customers"]}
          />
          <Bar dataKey="count" radius={[4, 4, 0, 0]}>
            {data.map((entry, index) => (
              <Cell
                key={index}
                fill={getBarColor(entry.min_score ?? index * 10)}
              />
            ))}
          </Bar>
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}
