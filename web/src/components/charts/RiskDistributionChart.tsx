import { useEffect, useState } from "react";
import {
  PieChart,
  Pie,
  Cell,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from "recharts";
import api from "@/lib/api";
import ChartSkeleton from "@/components/skeletons/ChartSkeleton";
import EmptyState from "@/components/EmptyState";
import { PieChart as PieChartIcon } from "lucide-react";

interface BucketData {
  range: string;
  count: number;
  min_score: number;
  max_score: number;
}

interface RiskSegment {
  name: string;
  value: number;
  color: string;
}

function aggregateRisk(buckets: BucketData[]): RiskSegment[] {
  let healthy = 0;
  let atRisk = 0;
  let critical = 0;

  for (const b of buckets) {
    if (b.min_score >= 70) {
      healthy += b.count;
    } else if (b.min_score >= 40) {
      atRisk += b.count;
    } else {
      critical += b.count;
    }
  }

  return [
    { name: "Healthy", value: healthy, color: "#22c55e" },
    { name: "At Risk", value: atRisk, color: "#eab308" },
    { name: "Critical", value: critical, color: "#ef4444" },
  ].filter((s) => s.value > 0);
}

export default function RiskDistributionChart() {
  const [segments, setSegments] = useState<RiskSegment[]>([]);
  const [loading, setLoading] = useState(true);
  const [empty, setEmpty] = useState(false);

  useEffect(() => {
    async function fetch() {
      try {
        const { data: res } = await api.get("/dashboard/score-distribution");
        const buckets = res.buckets ?? res.distribution ?? res ?? [];
        if (Array.isArray(buckets) && buckets.length > 0) {
          const agg = aggregateRisk(buckets);
          if (agg.length > 0) {
            setSegments(agg);
          } else {
            setEmpty(true);
          }
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
          icon={<PieChartIcon className="h-12 w-12" />}
          title="No risk data yet"
          description="Risk distribution will appear once customers are scored."
        />
      </div>
    );
  }

  const total = segments.reduce((sum, s) => sum + s.value, 0);

  return (
    <div className="rounded-lg border border-gray-200 bg-white p-6 dark:border-gray-700 dark:bg-gray-900">
      <h3 className="mb-4 text-sm font-medium text-gray-900 dark:text-gray-100">
        Risk Distribution
      </h3>
      <ResponsiveContainer width="100%" height={280}>
        <PieChart>
          <Pie
            data={segments}
            cx="50%"
            cy="50%"
            innerRadius={60}
            outerRadius={100}
            dataKey="value"
            label={({ name, value }) =>
              `${name}: ${value} (${Math.round((value / total) * 100)}%)`
            }
          >
            {segments.map((s) => (
              <Cell key={s.name} fill={s.color} />
            ))}
          </Pie>
          <Tooltip
            contentStyle={{
              backgroundColor: "var(--chart-tooltip-bg)",
              borderColor: "var(--chart-tooltip-border)",
              borderRadius: 8,
              color: "var(--chart-tooltip-text)",
            }}
            formatter={(value) => [
              `${value} (${Math.round(((value as number) / total) * 100)}%)`,
              "Customers",
            ]}
          />
          <Legend />
        </PieChart>
      </ResponsiveContainer>
    </div>
  );
}
