import { useEffect, useState } from "react";
import api from "@/lib/api";
import { useToast } from "@/contexts/ToastContext";
import { Loader2 } from "lucide-react";

interface Organization {
  id: string;
  name: string;
  slug: string;
  plan?: string;
}

export default function OrganizationTab() {
  const [org, setOrg] = useState<Organization | null>(null);
  const [loading, setLoading] = useState(true);
  const toast = useToast();

  useEffect(() => {
    async function fetch() {
      try {
        const { data } = await api.get<Organization>("/organizations/current");
        setOrg(data);
      } catch {
        toast.error("Failed to load organization info");
      } finally {
        setLoading(false);
      }
    }
    fetch();
  }, [toast]);

  if (loading) {
    return (
      <div className="flex justify-center py-8">
        <Loader2 className="h-6 w-6 animate-spin text-gray-400" />
      </div>
    );
  }

  if (!org) return null;

  return (
    <div className="max-w-lg space-y-4">
      <div>
        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
          Organization Name
        </label>
        <p className="mt-1 text-sm text-gray-900 dark:text-gray-100">
          {org.name}
        </p>
      </div>
      <div>
        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
          Slug
        </label>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {org.slug}
        </p>
      </div>
      {org.plan && (
        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
            Plan
          </label>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {org.plan}
          </p>
        </div>
      )}
    </div>
  );
}
