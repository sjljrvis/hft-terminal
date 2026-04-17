import { useEffect, useState } from "react";
import axios from "axios";

function PositionWidget() {
  const [position, setPosition] = useState(null);
  const [lastPrice, setLastPrice] = useState(null);

  useEffect(() => {
    const fetchPosition = async () => {
      try {
        const { data } = await axios.get("http://localhost:5001/live/position");
        setPosition(data);
      } catch {}
    };
    const fetchPrice = async () => {
      try {
        const { data } = await axios.get("http://localhost:5001/live/ticks");
        if (data?.length > 0) {
          setLastPrice(data[data.length - 1].close);
        }
      } catch {}
    };
    fetchPosition();
    fetchPrice();
    const interval = setInterval(() => {
      fetchPosition();
      fetchPrice();
    }, 3000);
    return () => clearInterval(interval);
  }, []);

  if (!position) {
    return (
      <section className="ln__section pw">
        <div className="ln__section-label">Position</div>
        <div className="pw__flat">FLAT</div>
      </section>
    );
  }

  const entry = position.entryPrice || 0;
  const current = lastPrice || entry;
  const unrealizedPnL =
    position.kind === "BUY" ? current - entry : entry - current;
  const pnlPct = entry > 0 ? (unrealizedPnL / entry) * 100 : 0;

  const entryTime = position.entryTime ? new Date(position.entryTime) : null;
  const elapsed = entryTime
    ? formatElapsed(Date.now() - entryTime.getTime())
    : "-";

  const peakProfit = position.peakProfit || 0;
  const peakLoss = position.peakLoss || 0;

  return (
    <section className="ln__section pw">
      <div className="ln__section-label">Position</div>
      <div className="pw__header">
        <span className={`pw__side pw__side--${position.kind?.toLowerCase()}`}>
          {position.kind}
        </span>
        <span className="pw__elapsed">{elapsed}</span>
      </div>
      <div className="pw__kv">
        <span>Entry</span>
        <span>{formatPrice(entry)}</span>
      </div>
      <div className="pw__kv">
        <span>Current</span>
        <span>{formatPrice(current)}</span>
      </div>
      <div className="pw__kv">
        <span>Unrealized</span>
        <span className={unrealizedPnL >= 0 ? "is-up" : "is-down"}>
          {unrealizedPnL >= 0 ? "+" : ""}
          {unrealizedPnL.toFixed(2)} ({pnlPct.toFixed(2)}%)
        </span>
      </div>
      <div className="pw__kv">
        <span>Peak</span>
        <span className="is-up">+{peakProfit.toFixed(2)}</span>
      </div>
      <div className="pw__kv">
        <span>Trough</span>
        <span className="is-down">{peakLoss.toFixed(2)}</span>
      </div>
    </section>
  );
}

function formatPrice(v) {
  return Number(v).toLocaleString("en-IN", {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });
}

function formatElapsed(ms) {
  const totalSec = Math.floor(ms / 1000);
  const h = Math.floor(totalSec / 3600);
  const m = Math.floor((totalSec % 3600) / 60);
  const s = totalSec % 60;
  if (h > 0) return `${h}h ${m}m`;
  if (m > 0) return `${m}m ${s}s`;
  return `${s}s`;
}

export default PositionWidget;
