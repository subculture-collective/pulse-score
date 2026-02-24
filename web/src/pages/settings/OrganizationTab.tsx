import { useEffect, useState } from "react";
import api, { onboardingApi } from "@/lib/api";
import { useToast } from "@/contexts/ToastContext";
import { useNavigate } from "react-router-dom";
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
  const [resetting, setResetting] = useState(false);
  const toast = useToast();
  const navigate = useNavigate();

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

  async function rerunOnboarding() {
    setResetting(true);
    try {
      await onboardingApi.reset();
      toast.success("Onboarding reset. Let’s run setup again.");
      navigate("/onboarding?step=welcome");
    } catch {
      toast.error("Failed to reset onboarding.");
    } finally {
      setResetting(false);
    }
  }

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

      <div className="rounded-lg border border-indigo-200 bg-indigo-50 p-4">
        <h3 className="text-sm font-semibold text-indigo-800">Onboarding</h3>
        <p className="mt-1 text-sm text-indigo-700">
          Need to revisit setup? You can restart the onboarding wizard anytime.
        </p>
        <button
          onClick={rerunOnboarding}
          disabled={resetting}
          className="mt-3 rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
        >
          {resetting ? "Resetting..." : "Re-run onboarding"}
        </button>
      </div>
    </div>
  );
}
