import { useEffect, useState } from "react";
import axios from "axios";

function PositionCard() {
  const [position, setPosition] = useState(null);
  const [lastPrice, setLastPrice] = useState(null);

  useEffect(() => {
    const controller = new AbortController();
    const poll = async () => {
      try {
        const { data } = await axios.get("http://localhost:5001/live/position", {
          signal: controller.signal,
        });
        setPosition(data);
      } catch (err) {
        if (!controller.signal.aborted) setPosition(null);
      }
      try {
        const { data: ticks } = await axios.get("http://localhost:5001/live/ticks", {
          signal: controller.signal,
        });
        if (ticks && ticks.length > 0) {
          setLastPrice(ticks[ticks.length - 1].close);
        }
      } catch {}
    };

    poll();
    const interval = setInterval(poll, 3000);
    return () => {
      controller.abort();
      clearInterval(interval);
    };
  }, []);

  if (!position) {
    return (
      <div className="position-card position-card--flat">
        <span className="position-card__label">Position</span>
        <span className="position-card__flat-text">FLAT</span>
      </div>
    );
  }

  const entryPrice = position.entryPrice || 0;
  const unrealizedPnl =
    lastPrice != null
      ? position.kind === "BUY"
        ? lastPrice - entryPrice
        : entryPrice - lastPrice
      : null;

  const entryTime = new Date(position.entryTime);
  const elapsed = Math.floor((Date.now() - entryTime.getTime()) / 1000);
  const elapsedMin = Math.floor(elapsed / 60);
  const elapsedSec = elapsed % 60;

  const formatPrice = (v) =>
    Number(v).toLocaleString("en-IN", { minimumFractionDigits: 2, maximumFractionDigits: 2 });

  return (
    <div className={`position-card position-card--${position.kind === "BUY" ? "long" : "short"}`}>
      <div className="position-card__row">
        <span className="position-card__label">Position</span>
        <span className={`position-card__side position-card__side--${position.kind?.toLowerCase()}`}>
          {position.kind}
        </span>
      </div>
      <div className="position-card__row">
        <span className="position-card__label">Entry</span>
        <span className="position-card__value">{formatPrice(entryPrice)}</span>
      </div>
      {lastPrice != null && (
        <div className="position-card__row">
          <span className="position-card__label">LTP</span>
          <span className="position-card__value">{formatPrice(lastPrice)}</span>
        </div>
      )}
      {unrealizedPnl != null && (
        <div className="position-card__row">
          <span className="position-card__label">Unrealized</span>
          <span
            className={`position-card__value ${unrealizedPnl >= 0 ? "profit-positive" : "profit-negative"}`}
          >
            {unrealizedPnl >= 0 ? "+" : ""}
            {unrealizedPnl.toFixed(2)} pts
          </span>
        </div>
      )}
      <div className="position-card__row">
        <span className="position-card__label">Duration</span>
        <span className="position-card__value">
          {elapsedMin}m {elapsedSec}s
        </span>
      </div>
    </div>
  );
}

export default PositionCard;
