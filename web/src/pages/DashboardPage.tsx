import { useCallback, useEffect, useState } from "react";
import api from "@/lib/api";
import { useToast } from "@/contexts/ToastContext";
import StatCard from "@/components/StatCard";
import CardSkeleton from "@/components/skeletons/CardSkeleton";
import ScoreDistributionChart from "@/components/charts/ScoreDistributionChart";
import MRRTrendChart from "@/components/charts/MRRTrendChart";
import { Users, AlertTriangle, DollarSign, Activity } from "lucide-react";

interface DashboardSummary {
  total_customers: number;
  at_risk_customers: number;
  total_mrr: number;
  average_health_score: number;
}

function formatCurrency(cents: number): string {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(cents / 100);
}

export default function DashboardPage() {
  const [summary, setSummary] = useState<DashboardSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);
  const toast = useToast();

  const fetchSummary = useCallback(async () => {
    try {
      const { data } = await api.get<DashboardSummary>("/dashboard/summary");
      setSummary(data);
      setError(false);
    } catch {
      setError(true);
      toast.error("Failed to load dashboard summary");
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => {
    fetchSummary();
    const interval = setInterval(fetchSummary, 5 * 60 * 1000);
    return () => clearInterval(interval);
  }, [fetchSummary]);

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
        Dashboard
      </h1>

      {/* Stat cards */}
      {loading ? (
        <div className="grid grid-cols-1 gap-6 md:grid-cols-2 xl:grid-cols-4">
          {[...Array(4)].map((_, i) => (
            <CardSkeleton key={i} />
          ))}
        </div>
      ) : error ? (
        <div className="rounded-lg border border-red-200 bg-red-50 p-6 text-center dark:border-red-800 dark:bg-red-950">
          <p className="text-sm text-red-700 dark:text-red-300">
            Failed to load dashboard data.
          </p>
          <button
            onClick={fetchSummary}
            className="mt-2 text-sm font-medium text-indigo-600 hover:underline dark:text-indigo-400"
          >
            Retry
          </button>
        </div>
      ) : summary ? (
        <div className="grid grid-cols-1 gap-6 md:grid-cols-2 xl:grid-cols-4">
          <StatCard
            title="Total Customers"
            value={summary.total_customers}
            icon={<Users className="h-5 w-5" />}
          />
          <StatCard
            title="At-Risk Customers"
            value={summary.at_risk_customers}
            icon={<AlertTriangle className="h-5 w-5" />}
          />
          <StatCard
            title="Total MRR"
            value={formatCurrency(summary.total_mrr)}
            icon={<DollarSign className="h-5 w-5" />}
          />
          <StatCard
            title="Avg Health Score"
            value={Math.round(summary.average_health_score)}
            icon={<Activity className="h-5 w-5" />}
          />
        </div>
      ) : null}

      {/* Charts */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <ScoreDistributionChart />
        <MRRTrendChart />
      </div>
    </div>
  );
}
