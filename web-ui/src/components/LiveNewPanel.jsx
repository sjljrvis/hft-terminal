import { useEffect, useState, useRef, useCallback } from "react";
import axios from "axios";
import ChartPanel from "./ChartPanel";
import EquityCurve from "./EquityCurve";
import ActivityPanel from "./ActivityPanel";
import PositionWidget from "./PositionWidget";
import RiskGauge from "./RiskGauge";
import StrategyControls from "./StrategyControls";
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
          <div className="ln__kv" style={{ marginTop: 4 }}>
            <span>{metrics.totalTrades} trades</span>
            <span>{winRate.toFixed(0)}% win</span>
          </div>
        </section>

        <PositionWidget />

        {broker && <RiskGauge broker={broker} />}

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

        {/* <StrategyControls /> */}
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

export default LiveNewPanel;
