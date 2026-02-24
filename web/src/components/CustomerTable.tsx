import { useNavigate } from "react-router-dom";
import HealthScoreBadge from "@/components/HealthScoreBadge";
import { ChevronUp, ChevronDown, ChevronsUpDown } from "lucide-react";

export interface Customer {
  id: string;
  name: string;
  email: string;
  company: string;
  mrr: number;
  health_score: number;
  risk_level: "green" | "yellow" | "red";
  last_seen_at: string;
}

interface CustomerTableProps {
  customers: Customer[];
  sort: string;
  order: "asc" | "desc";
  onSort: (field: string) => void;
}

function formatCurrency(cents: number): string {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
    minimumFractionDigits: 0,
  }).format(cents / 100);
}

function relativeTime(dateStr: string): string {
  if (!dateStr) return "—";
  const diff = Date.now() - new Date(dateStr).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return "Just now";
  if (mins < 60) return `${mins}m ago`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  if (days < 30) return `${days}d ago`;
  return new Date(dateStr).toLocaleDateString();
}

const columns = [
  { key: "name", label: "Name" },
  { key: "company", label: "Company" },
  { key: "mrr", label: "MRR" },
  { key: "health_score", label: "Health Score" },
  { key: "risk_level", label: "Risk" },
  { key: "last_seen_at", label: "Last Seen" },
];

function SortIcon({
  field,
  sort,
  order,
}: {
  field: string;
  sort: string;
  order: string;
}) {
  if (sort !== field)
    return <ChevronsUpDown className="h-3 w-3 text-gray-400" />;
  return order === "asc" ? (
    <ChevronUp className="h-3 w-3" />
  ) : (
    <ChevronDown className="h-3 w-3" />
  );
}

export default function CustomerTable({
  customers,
  sort,
  order,
  onSort,
}: CustomerTableProps) {
  const navigate = useNavigate();

  return (
    <div className="overflow-x-auto rounded-lg border border-gray-200 dark:border-gray-700">
      <table className="w-full text-left text-sm">
        <thead className="border-b border-gray-200 bg-gray-50 text-xs uppercase text-gray-500 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-400">
          <tr>
            {columns.map((col) => (
              <th
                key={col.key}
                className="cursor-pointer px-6 py-3 hover:text-gray-700 dark:hover:text-gray-200"
                onClick={() => onSort(col.key)}
              >
                <span className="flex items-center gap-1">
                  {col.label}
                  <SortIcon field={col.key} sort={sort} order={order} />
                </span>
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {customers.map((c) => (
            <tr
              key={c.id}
              onClick={() => navigate(`/customers/${c.id}`)}
              className="cursor-pointer border-b border-gray-100 bg-white hover:bg-gray-50 dark:border-gray-800 dark:bg-gray-900 dark:hover:bg-gray-800"
            >
              <td className="px-6 py-4 font-medium text-gray-900 dark:text-gray-100">
                {c.name}
                <div className="text-xs text-gray-500 dark:text-gray-400">
                  {c.email}
                </div>
              </td>
              <td className="px-6 py-4 text-gray-700 dark:text-gray-300">
                {c.company}
              </td>
              <td className="px-6 py-4 text-gray-700 dark:text-gray-300">
                {formatCurrency(c.mrr)}
              </td>
              <td className="px-6 py-4">
                <HealthScoreBadge
                  score={c.health_score}
                  riskLevel={c.risk_level}
                  size="sm"
                />
              </td>
              <td className="px-6 py-4">
                <span
                  className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${
                    c.risk_level === "green"
                      ? "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"
                      : c.risk_level === "yellow"
                        ? "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200"
                        : "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200"
                  }`}
                >
                  {c.risk_level}
                </span>
              </td>
              <td className="px-6 py-4 text-gray-500 dark:text-gray-400">
                {relativeTime(c.last_seen_at)}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
