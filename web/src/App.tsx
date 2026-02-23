import BaseLayout from "@/components/BaseLayout";

function App() {
  return (
    <BaseLayout>
      <div className="text-center">
        <h2 className="text-2xl font-semibold text-gray-900">Dashboard</h2>
        <p className="mt-2 text-gray-600">
          Welcome to PulseScore. Connect your integrations to get started.
        </p>
      </div>
    </BaseLayout>
  );
}

export default App;
