import { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import api from "@/lib/api";
import { formatCurrency, relativeTime } from "@/lib/format";
import HealthScoreBadge from "@/components/HealthScoreBadge";
import { Loader2 } from "lucide-react";

interface AtRiskCustomer {
  id: string;
  name: string;
  health_score: number;
  risk_level: "green" | "yellow" | "red";
  mrr: number;
  last_seen_at: string;
}

export default function AtRiskCustomersTable() {
  const [customers, setCustomers] = useState<AtRiskCustomer[]>([]);
  const [loading, setLoading] = useState(true);
  const navigate = useNavigate();

  useEffect(() => {
    async function fetch() {
      try {
        const { data } = await api.get("/customers", {
          params: {
            sort: "health_score",
            order: "asc",
            per_page: 5,
          },
        });
        const list = data.customers ?? data.data ?? data ?? [];
        setCustomers(Array.isArray(list) ? list : []);
      } catch {
        setCustomers([]);
      } finally {
        setLoading(false);
      }
    }
    fetch();
  }, []);

  return (
    <div className="rounded-lg border border-gray-200 bg-white p-6 dark:border-gray-700 dark:bg-gray-900">
      <div className="mb-4 flex items-center justify-between">
        <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100">
          At-Risk Customers
        </h3>
        <Link
          to="/customers?sort=health_score&order=asc"
          className="text-xs font-medium text-indigo-600 hover:underline dark:text-indigo-400"
        >
          View all
        </Link>
      </div>

      {loading ? (
        <div className="flex justify-center py-8">
          <Loader2 className="h-5 w-5 animate-spin text-gray-400" />
        </div>
      ) : customers.length === 0 ? (
        <p className="py-8 text-center text-sm text-gray-500 dark:text-gray-400">
          No at-risk customers
        </p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead className="border-b border-gray-200 text-xs uppercase text-gray-500 dark:border-gray-700 dark:text-gray-400">
              <tr>
                <th className="pb-2 pr-4">Name</th>
                <th className="pb-2 pr-4">Score</th>
                <th className="pb-2 pr-4">MRR</th>
                <th className="pb-2">Last Seen</th>
              </tr>
            </thead>
            <tbody>
              {customers.map((c) => (
                <tr
                  key={c.id}
                  onClick={() => navigate(`/customers/${c.id}`)}
                  className="cursor-pointer border-b border-gray-100 last:border-0 hover:bg-gray-50 dark:border-gray-800 dark:hover:bg-gray-800"
                >
                  <td className="py-3 pr-4 font-medium text-gray-900 dark:text-gray-100">
                    {c.name}
                  </td>
                  <td className="py-3 pr-4">
                    <HealthScoreBadge
                      score={c.health_score}
                      riskLevel={c.risk_level}
                      size="sm"
                      showLabel
                    />
                  </td>
                  <td className="py-3 pr-4 text-gray-700 dark:text-gray-300">
                    {formatCurrency(c.mrr)}
                  </td>
                  <td className="py-3 text-gray-500 dark:text-gray-400">
                    {relativeTime(c.last_seen_at)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
