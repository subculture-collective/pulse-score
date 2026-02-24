import WizardStep from "@/components/wizard/WizardStep";

export interface WelcomeFormValue {
  name: string;
  industry: string;
  company_size: string;
}

interface WelcomeStepProps {
  value: WelcomeFormValue;
  organizationName: string;
  onChange: (value: WelcomeFormValue) => void;
}

const industries = [
  "SaaS",
  "E-commerce",
  "FinTech",
  "Healthcare",
  "Education",
  "Professional Services",
  "Other",
];

const companySizes = ["1-10", "11-50", "51-200", "201-500", "500+"];

export default function WelcomeStep({
  value,
  organizationName,
  onChange,
}: WelcomeStepProps) {
  return (
    <WizardStep
      title="Welcome to PulseScore"
      description="Let’s configure your organization so we can personalize scoring and insights."
    >
      <div className="mb-5 rounded-lg border border-indigo-200 bg-indigo-50 p-4 text-sm text-indigo-700">
        You’re about to connect customer data and generate your first health
        score preview. It usually takes just a few minutes.
      </div>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <label className="block">
          <span className="mb-1 block text-sm font-medium text-gray-700">
            Organization name *
          </span>
          <input
            value={value.name}
            onChange={(e) =>
              onChange({
                ...value,
                name: e.target.value,
              })
            }
            placeholder={organizationName}
            className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm outline-none ring-indigo-500 focus:ring"
          />
        </label>

        <label className="block">
          <span className="mb-1 block text-sm font-medium text-gray-700">
            Industry
          </span>
          <select
            value={value.industry}
            onChange={(e) =>
              onChange({
                ...value,
                industry: e.target.value,
              })
            }
            className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm outline-none ring-indigo-500 focus:ring"
          >
            <option value="">Select an industry</option>
            {industries.map((industry) => (
              <option key={industry} value={industry}>
                {industry}
              </option>
            ))}
          </select>
        </label>

        <label className="block md:col-span-2">
          <span className="mb-1 block text-sm font-medium text-gray-700">
            Company size
          </span>
          <select
            value={value.company_size}
            onChange={(e) =>
              onChange({
                ...value,
                company_size: e.target.value,
              })
            }
            className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm outline-none ring-indigo-500 focus:ring"
          >
            <option value="">Select company size</option>
            {companySizes.map((size) => (
              <option key={size} value={size}>
                {size}
              </option>
            ))}
          </select>
        </label>
      </div>
    </WizardStep>
  );
}
