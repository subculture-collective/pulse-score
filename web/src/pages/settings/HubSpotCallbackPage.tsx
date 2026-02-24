import { useEffect, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { hubspotApi } from "@/lib/hubspot";
import BaseLayout from "@/components/BaseLayout";

export default function HubSpotCallbackPage() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const [error, setError] = useState("");

  useEffect(() => {
    const code = searchParams.get("code");
    const state = searchParams.get("state");
    const errParam = searchParams.get("error");

    if (errParam) {
      const desc = searchParams.get("error_description") || errParam;
      setError(`HubSpot connection failed: ${desc}`);
      return;
    }

    if (!code || !state) {
      setError("Invalid callback parameters.");
      return;
    }

    hubspotApi
      .callback(code, state)
      .then(() => {
        navigate("/settings", { replace: true });
      })
      .catch(() => {
        setError("Failed to complete HubSpot connection.");
      });
  }, [searchParams, navigate]);

  if (error) {
    return (
      <BaseLayout>
        <div className="mx-auto max-w-md text-center">
          <div className="rounded-md bg-red-50 p-6">
            <h2 className="text-lg font-semibold text-red-800">
              Connection Failed
            </h2>
            <p className="mt-2 text-sm text-red-700">{error}</p>
            <button
              onClick={() => navigate("/settings")}
              className="mt-4 rounded-md bg-orange-600 px-4 py-2 text-sm font-medium text-white hover:bg-orange-700"
            >
              Back to Settings
            </button>
          </div>
        </div>
      </BaseLayout>
    );
  }

  return (
    <BaseLayout>
      <div className="mx-auto max-w-md text-center">
        <p className="text-gray-600">Connecting HubSpot...</p>
      </div>
    </BaseLayout>
  );
}
