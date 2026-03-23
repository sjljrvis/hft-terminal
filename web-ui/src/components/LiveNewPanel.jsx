import { useEffect, useState, useRef, useCallback } from "react";
import axios from "axios";
import ChartPanel from "./ChartPanel";
import EquityCurve from "./EquityCurve";
import ActivityPanel from "./ActivityPanel";
import { useAppDispatch, useAppSelector } from "../store/hooks";
import {
  fetchLiveTrades,
  selectLiveTrades,
  selectLiveTotalPnL,
  selectLiveTradeMetrics,
} from "../store/slices/tradesSlice";
import { Crosshair } from "phosphor-react";

function LiveNewPanel() {
  const [status, setStatus] = useState(null);
  const [activeTab, setActiveTab] = useState("equity"); // "chart" | "equity"
  const [tradesHeight, setTradesHeight] = useState(250);
  const dragging = useRef(false);
  const startY = useRef(0);
  const startH = useRef(0);

  const onDragStart = useCallback((e) => {
    e.preventDefault();
    dragging.current = true;
    startY.current = e.clientY;
    startH.current = tradesHeight;
    document.body.style.cursor = "row-resize";
    document.body.style.userSelect = "none";

    const onMove = (ev) => {
      if (!dragging.current) return;
      const delta = startY.current - ev.clientY;
      setTradesHeight(Math.max(80, Math.min(startH.current + delta, window.innerHeight * 0.7)));
    };
    const onUp = () => {
      dragging.current = false;
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
      window.removeEventListener("mousemove", onMove);
      window.removeEventListener("mouseup", onUp);
    };
    window.addEventListener("mousemove", onMove);
    window.addEventListener("mouseup", onUp);
  }, [tradesHeight]);
  const dispatch = useAppDispatch();
  const liveTrades = useAppSelector(selectLiveTrades);
  const positionPnL = useAppSelector(selectLiveTotalPnL);
  const metrics = useAppSelector(selectLiveTradeMetrics);

  useEffect(() => {
    const controller = new AbortController();
    const fetchStatus = async () => {
      try {
        const { data } = await axios.get("http://localhost:5001/hft/status", {
          signal: controller.signal,
        });
        setStatus(data);
      } catch {}
    };
    fetchStatus();
    const interval = setInterval(fetchStatus, 5000);
    return () => {
      controller.abort();
      clearInterval(interval);
    };
  }, []);

  useEffect(() => {
    dispatch(fetchLiveTrades());
    const interval = setInterval(() => dispatch(fetchLiveTrades()), 5000);
    return () => clearInterval(interval);
  }, [dispatch]);

  const pnl = positionPnL ?? 0;
  const winRate = metrics.totalTrades > 0 ? metrics.winRate : 0;
  const avgProfit =
    metrics.totalTrades > 0 ? metrics.totalPnl / metrics.totalTrades : 0;
  const broker = status?.hft?.Broker;

  const formatPrice = (v) =>
    Number(v).toLocaleString("en-IN", {
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    });

  const formatTime = (value) => {
    if (!value) return "-";
    const d = new Date(value);
    if (isNaN(d.getTime())) return "-";
    return d.toLocaleString("en-GB", {
      day: "2-digit",
      month: "short",
      hour: "2-digit",
      minute: "2-digit",
      hour12: false,
    });
  };

  const recentTrades = [...(liveTrades || [])]
    .sort((a, b) => new Date(b.exitTime) - new Date(a.exitTime))
    .slice(0, 50);

  return (
    <div className="ln">
      {/* ── LEFT COLUMN ── */}
      <aside className="ln__left">
        <section className="ln__section">
          <div className="ln__section-label">P&L</div>
          <div className={`ln__pnl-value ${pnl >= 0 ? "is-up" : "is-down"}`}>
            {pnl >= 0 ? "+" : ""}
            {formatPrice(pnl)}
          </div>
          <div className="ln__pnl-sub">
            <span>Session</span>
            <span className={pnl >= 0 ? "is-up" : "is-down"}>
              {metrics.totalPnl >= 0 ? "+" : ""}
              {formatPrice(metrics.totalPnl)}
            </span>
          </div>
        </section>

        {broker && (
          <section className="ln__section">
            <div className="ln__section-label">Broker</div>
            <div className="ln__kv">
              <span>Margin</span>
              <span>{formatPrice(broker.AvailableMargin)}</span>
            </div>
            <div className="ln__kv">
              <span>Utilized</span>
              <span>{formatPrice(broker.UtilizedMargin)}</span>
            </div>
            <div className="ln__kv">
              <span>Equity</span>
              <span>{formatPrice(broker.Equity)}</span>
            </div>
          </section>
        )}

        <section className="ln__section">
          <div className="ln__section-label">System</div>
          <div className="ln__kv">
            <span>Mode</span>
            <span className="ln__tag">LIVE</span>
          </div>
          <div className="ln__kv">
            <span>Status</span>
            <span className={`ln__tag ${status?.hft?.Status === "connected" ? "ln__tag--green" : ""}`}>
              {status?.hft?.Status?.toUpperCase() || "OFFLINE"}
            </span>
          </div>
        </section>

        <section className="ln__section">
          <div className="ln__section-label">Performance</div>
          <div className="ln__perf-grid">
            <div className="ln__perf-ring">
              <svg viewBox="0 0 36 36" className="ln__ring-svg">
                <circle cx="18" cy="18" r="15.9" fill="none" stroke="var(--border-color)" strokeWidth="2" />
                <circle
                  cx="18"
                  cy="18"
                  r="15.9"
                  fill="none"
                  stroke="var(--green-candle)"
                  strokeWidth="2"
                  strokeDasharray={`${winRate} ${100 - winRate}`}
                  strokeDashoffset="25"
                  strokeLinecap="round"
                />
              </svg>
              <span className="ln__ring-label">{winRate.toFixed(0)}%</span>
              <span className="ln__ring-sub">Win</span>
            </div>
            <div className="ln__perf-ring">
              <svg viewBox="0 0 36 36" className="ln__ring-svg">
                <circle cx="18" cy="18" r="15.9" fill="none" stroke="var(--border-color)" strokeWidth="2" />
                <circle
                  cx="18"
                  cy="18"
                  r="15.9"
                  fill="none"
                  stroke={avgProfit >= 0 ? "var(--green-candle)" : "var(--red-candle)"}
                  strokeWidth="2"
                  strokeDasharray={`${Math.min(Math.abs(avgProfit) * 2, 100)} ${100 - Math.min(Math.abs(avgProfit) * 2, 100)}`}
                  strokeDashoffset="25"
                  strokeLinecap="round"
                />
              </svg>
              <span className="ln__ring-label">
                {avgProfit >= 0 ? "+" : ""}
                {avgProfit.toFixed(1)}
              </span>
              <span className="ln__ring-sub">Avg</span>
            </div>
          </div>
          <div className="ln__stat-row">
            <div className="ln__stat">
              <span className="ln__stat-val">{metrics.totalTrades}</span>
              <span className="ln__stat-label">Trades</span>
            </div>
            <div className="ln__stat">
              <span className="ln__stat-val">{formatPrice(metrics.maxDrawdown)}</span>
              <span className="ln__stat-label">Max DD</span>
            </div>
          </div>
          <div className="ln__stat-row">
            <div className="ln__stat">
              <span className="ln__stat-val">{metrics.profitFactor === Infinity ? "INF" : metrics.profitFactor?.toFixed(2)}</span>
              <span className="ln__stat-label">Profit Factor</span>
            </div>
            <div className="ln__stat">
              <span className="ln__stat-val">
                {metrics.winCount}/{metrics.totalTrades}
              </span>
              <span className="ln__stat-label">W/L</span>
            </div>
          </div>
        </section>

        <section className="ln__section">
          <div className="ln__section-label">Exit Breakdown</div>
          <ExitBar label="Target" count={metrics.profitTargetCount} total={metrics.totalTrades} color="var(--green-candle)" />
          <ExitBar label="Trail" count={metrics.trailingStopCount} total={metrics.totalTrades} color="var(--accent)" />
          <ExitBar label="SL" count={metrics.stopLossCount} total={metrics.totalTrades} color="var(--red-candle)" />
          <ExitBar label="Signal" count={metrics.signalCount} total={metrics.totalTrades} color="var(--muted-color)" />
        </section>
      </aside>

      {/* ── CENTER COLUMN ── */}
      <div className="ln__center">
        <div className="ln__center-tabs">
          <button
            className={`ln__tab ${activeTab === "chart" ? "is-active" : ""}`}
            onClick={() => setActiveTab("chart")}
          >
            Chart
          </button>
          <button
            className={`ln__tab ${activeTab === "equity" ? "is-active" : ""}`}
            onClick={() => setActiveTab("equity")}
          >
            Equity Curve
          </button>
        </div>
        <div className="ln__center-chart">
          {activeTab === "chart" ? (
            <ChartPanel apiEndpoint="http://localhost:5001/live/ticks" />
          ) : (
            <EquityCurve trades={liveTrades} height={400} />
          )}
        </div>
        <div className="ln__resize-handle" onMouseDown={onDragStart} />
        <div className="ln__trades" style={{ height: tradesHeight }}>
          <div className="ln__trades-header">
            <span className="ln__section-label">Market Executions</span>
            <span className="ln__trades-count">{recentTrades.length} trades</span>
          </div>
          <div className="ln__trades-body">
            <table className="ln__trades-table">
              <thead>
                <tr>
                  <th>Time</th>
                  <th>Side</th>
                  <th>Entry</th>
                  <th>Exit</th>
                  <th>P&L</th>
                  <th>Reason</th>
                </tr>
              </thead>
              <tbody>
                {recentTrades.map((t, i) => {
                  const profit = Number(t.profit) || 0;
                  return (
                    <tr key={i}>
                      <td>
                        <Crosshair
                          size={12}
                          weight="regular"
                          className="ln__crosshair"
                          onClick={() =>
                            window.dispatchEvent(
                              new CustomEvent("onDateChange", {
                                detail: { date: t.entryTime },
                              })
                            )
                          }
                        />
                        {formatTime(t.exitTime)}
                      </td>
                      <td className={t.type === "BUY" ? "is-up" : "is-down"}>
                        {t.type}
                      </td>
                      <td>{formatPrice(t.entryPrice)}</td>
                      <td>{formatPrice(t.exitPrice)}</td>
                      <td className={profit >= 0 ? "is-up" : "is-down"}>
                        {profit >= 0 ? "+" : ""}
                        {profit.toFixed(2)}
                      </td>
                      <td>
                        <span className={`ln__reason ln__reason--${(t.reason || "").toLowerCase().replace("_", "-")}`}>
                          {t.reason === "PROFIT_TARGET"
                            ? "Target"
                            : t.reason === "STOP_LOSS"
                            ? "SL"
                            : t.reason === "TRAILING_STOP"
                            ? "Trail"
                            : t.reason === "SIGNAL"
                            ? "Signal"
                            : t.reason || "-"}
                        </span>
                      </td>
                    </tr>
                  );
                })}
                {recentTrades.length === 0 && (
                  <tr>
                    <td colSpan={6} className="ln__trades-empty">
                      No trades yet
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      </div>

      {/* ── RIGHT COLUMN ── */}
      <ActivityPanel />
    </div>
  );
}

function ExitBar({ label, count, total, color }) {
  const pct = total > 0 ? (count / total) * 100 : 0;
  return (
    <div className="ln__exit-bar">
      <div className="ln__exit-bar-label">
        <span>{label}</span>
        <span>
          {count} <span className="ln__exit-bar-pct">({pct.toFixed(0)}%)</span>
        </span>
      </div>
      <div className="ln__exit-bar-track">
        <div
          className="ln__exit-bar-fill"
          style={{ width: `${pct}%`, background: color }}
        />
      </div>
    </div>
  );
}

export default LiveNewPanel;
