import { useEffect, useState, useRef } from "react";
import { Warning } from "phosphor-react";

const DRAWDOWN_THRESHOLD = 200; // points

function DrawdownAlert({ trades }) {
  const [alert, setAlert] = useState(null);
  const [dismissed, setDismissed] = useState(false);
  const audioRef = useRef(null);

  useEffect(() => {
    if (!trades || trades.length === 0) {
      setAlert(null);
      return;
    }

    let cumulative = 0;
    let peak = 0;
    let maxDD = 0;

    trades.forEach((t) => {
      cumulative += Number(t.profit) || 0;
      if (cumulative > peak) peak = cumulative;
      const dd = peak - cumulative;
      if (dd > maxDD) maxDD = dd;
    });

    if (maxDD >= DRAWDOWN_THRESHOLD) {
      setAlert({ drawdown: maxDD, pct: peak > 0 ? ((maxDD / peak) * 100).toFixed(1) : "N/A" });
      setDismissed(false);
    } else {
      setAlert(null);
    }
  }, [trades]);

  if (!alert || dismissed) return null;

  return (
    <div className="drawdown-alert">
      <Warning size={14} weight="bold" />
      <span>
        Drawdown alert: {alert.drawdown.toFixed(2)} pts
        {alert.pct !== "N/A" && ` (${alert.pct}% from peak)`}
      </span>
      <button className="drawdown-alert__dismiss" onClick={() => setDismissed(true)}>
        Dismiss
      </button>
    </div>
  );
}

export default DrawdownAlert;
