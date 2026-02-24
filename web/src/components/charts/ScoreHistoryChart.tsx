import { useEffect, useState } from "react";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  Tooltip,
  ReferenceArea,
  ResponsiveContainer,
} from "recharts";
import api from "@/lib/api";
import ChartSkeleton from "@/components/skeletons/ChartSkeleton";
import EmptyState from "@/components/EmptyState";
import { TrendingUp } from "lucide-react";

interface ScorePoint {
  date: string;
  score: number;
}

function formatDate(dateStr: string): string {
  const d = new Date(dateStr);
  return d.toLocaleDateString("en-US", { month: "short", day: "numeric" });
}

export default function ScoreHistoryChart({
  customerId,
}: {
  customerId: string;
}) {
  const [data, setData] = useState<ScorePoint[]>([]);
  const [loading, setLoading] = useState(true);
  const [empty, setEmpty] = useState(false);

  useEffect(() => {
    async function fetch() {
      try {
        const { data: res } = await api.get(
          `/customers/${customerId}/score-history`,
        );
        const points = res.history ?? res.data ?? res ?? [];
        if (Array.isArray(points) && points.length > 0) {
          setData(points);
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
  }, [customerId]);

  if (loading) return <ChartSkeleton />;

  if (empty) {
    return (
      <div className="rounded-lg border border-gray-200 bg-white p-6 dark:border-gray-700 dark:bg-gray-900">
        <EmptyState
          icon={<TrendingUp className="h-12 w-12" />}
          title="No score history"
          description="Score history will appear as health scores are calculated over time."
        />
      </div>
    );
  }

  return (
    <div className="rounded-lg border border-gray-200 bg-white p-6 dark:border-gray-700 dark:bg-gray-900">
      <h3 className="mb-4 text-sm font-medium text-gray-900 dark:text-gray-100">
        Health Score History
      </h3>
      <ResponsiveContainer width="100%" height={280}>
        <LineChart data={data}>
          <ReferenceArea y1={70} y2={100} fill="#22c55e" fillOpacity={0.08} />
          <ReferenceArea y1={40} y2={70} fill="#eab308" fillOpacity={0.08} />
          <ReferenceArea y1={0} y2={40} fill="#ef4444" fillOpacity={0.08} />
          <XAxis
            dataKey="date"
            tickFormatter={formatDate}
            tick={{ fontSize: 12 }}
            stroke="#9ca3af"
          />
          <YAxis domain={[0, 100]} tick={{ fontSize: 12 }} stroke="#9ca3af" />
          <Tooltip
            formatter={(value) => [value, "Score"]}
            labelFormatter={(label) => formatDate(String(label))}
            contentStyle={{
              backgroundColor: "var(--color-white, #fff)",
              borderColor: "#e5e7eb",
              borderRadius: 8,
            }}
          />
          <Line
            type="monotone"
            dataKey="score"
            stroke="#6366f1"
            strokeWidth={2}
            dot={{ r: data.length <= 10 ? 4 : 2 }}
            activeDot={{ r: 6 }}
          />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}
