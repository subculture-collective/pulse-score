import { useEffect, useState } from "react";
import { CheckCircle, XCircle, AlertTriangle, Info, X } from "lucide-react";

interface ToastProps {
  type: "success" | "error" | "warning" | "info";
  message: string;
  onClose: () => void;
}

const styles = {
  success: {
    bg: "bg-green-50 dark:bg-green-950 border-green-200 dark:border-green-800",
    text: "text-green-800 dark:text-green-200",
    Icon: CheckCircle,
  },
  error: {
    bg: "bg-red-50 dark:bg-red-950 border-red-200 dark:border-red-800",
    text: "text-red-800 dark:text-red-200",
    Icon: XCircle,
  },
  warning: {
    bg: "bg-yellow-50 dark:bg-yellow-950 border-yellow-200 dark:border-yellow-800",
    text: "text-yellow-800 dark:text-yellow-200",
    Icon: AlertTriangle,
  },
  info: {
    bg: "bg-blue-50 dark:bg-blue-950 border-blue-200 dark:border-blue-800",
    text: "text-blue-800 dark:text-blue-200",
    Icon: Info,
  },
};

export default function Toast({ type, message, onClose }: ToastProps) {
  const [visible, setVisible] = useState(false);
  const { bg, text, Icon } = styles[type];

  useEffect(() => {
    requestAnimationFrame(() => setVisible(true));
  }, []);

  function handleClose() {
    setVisible(false);
    setTimeout(onClose, 200);
  }

  return (
    <div
      className={`flex w-80 items-start gap-3 rounded-lg border p-4 shadow-lg transition-all duration-200 ${bg} ${
        visible ? "translate-x-0 opacity-100" : "translate-x-full opacity-0"
      }`}
    >
      <Icon className={`h-5 w-5 shrink-0 ${text}`} />
      <p className={`flex-1 text-sm font-medium ${text}`}>{message}</p>
      <button
        onClick={handleClose}
        className={`shrink-0 ${text} hover:opacity-70`}
      >
        <X className="h-4 w-4" />
      </button>
    </div>
  );
}
