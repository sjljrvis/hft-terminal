function RiskGauge({ broker }) {
  const margin = broker?.Margin || broker?.AvailableMargin || 0;
  const utilized = broker?.UtilizedMargin || 0;
  const total = margin + utilized;
  const pct = total > 0 ? (utilized / total) * 100 : 0;

  // Color thresholds: green < 40%, yellow 40-70%, red > 70%
  const color =
    pct > 70
      ? "var(--red-candle)"
      : pct > 40
      ? "#e8a735"
      : "var(--green-candle)";

  const label =
    pct > 70 ? "HIGH RISK" : pct > 40 ? "MODERATE" : "LOW RISK";

  // SVG arc for semi-circle gauge
  const radius = 40;
  const circumference = Math.PI * radius; // half circle
  const filled = (pct / 100) * circumference;

  return (
    <section className="ln__section rg">
      <div className="ln__section-label">Margin Risk</div>
      <div className="rg__gauge">
        <svg viewBox="0 0 100 55" className="rg__svg">
          <path
            d="M 10 50 A 40 40 0 0 1 90 50"
            fill="none"
            stroke="var(--border-color)"
            strokeWidth="6"
            strokeLinecap="round"
          />
          <path
            d="M 10 50 A 40 40 0 0 1 90 50"
            fill="none"
            stroke={color}
            strokeWidth="6"
            strokeLinecap="round"
            strokeDasharray={`${filled} ${circumference}`}
          />
        </svg>
        <div className="rg__value" style={{ color }}>
          {pct.toFixed(0)}%
        </div>
        <div className="rg__label" style={{ color }}>
          {label}
        </div>
      </div>
      <div className="rg__details">
        <div className="pw__kv">
          <span>Available</span>
          <span>{formatPrice(margin)}</span>
        </div>
        <div className="pw__kv">
          <span>Utilized</span>
          <span style={{ color: utilized > 0 ? color : undefined }}>
            {formatPrice(utilized)}
          </span>
        </div>
        <div className="pw__kv">
          <span>Total</span>
          <span>{formatPrice(total)}</span>
        </div>
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

export default RiskGauge;
