import { useState, useMemo } from "react";
import { X, Crosshair } from "phosphor-react";
import { useAppDispatch, useAppSelector } from "../store/hooks";
import {
  selectEquityDrawerOpen,
  toggleEquityDrawer,
  selectActivePanel,
} from "../store/slices/uiSlice";
import {
  selectTrades,
  selectLiveTrades,
  selectTradeMetrics,
  selectLiveTradeMetrics,
} from "../store/slices/tradesSlice";
import EquityCurve from "./EquityCurve";

function EquityDrawer() {
  const dispatch = useAppDispatch();
  const open = useAppSelector(selectEquityDrawerOpen);
  const activePanel = useAppSelector(selectActivePanel);
  const liveTrades = useAppSelector(selectLiveTrades);
  const backtestTrades = useAppSelector(selectTrades);
  const trades = activePanel === "live" ? liveTrades : backtestTrades;
  const baseMetrics = useAppSelector(
    activePanel === "live" ? selectLiveTradeMetrics : selectTradeMetrics
  );

  const [tab, setTab] = useState("equity");
  const [sortField, setSortField] = useState("");
  const [sortDir, setSortDir] = useState("asc");

  const toggleSort = (field) => {
    if (sortField === field) {
      setSortDir((d) => (d === "asc" ? "desc" : "asc"));
    } else {
      setSortField(field);
      setSortDir("asc");
    }
  };

  const sortIndicator = (field) => {
    if (sortField !== field) return null;
    return <span className="sort-indicator">{sortDir === "asc" ? "\u2191" : "\u2193"}</span>;
  };

  const sortedTrades = useMemo(() => {
    if (!trades || trades.length === 0) return [];
    if (!sortField) return trades;
    const sorted = [...trades].sort((a, b) => {
      let va = a[sortField];
      let vb = b[sortField];
      if (sortField === "entryTime" || sortField === "exitTime") {
        va = new Date(va).getTime() || 0;
        vb = new Date(vb).getTime() || 0;
      } else if (typeof va === "string") {
        va = va.toLowerCase();
        vb = (vb || "").toLowerCase();
      } else {
        va = Number(va) || 0;
        vb = Number(vb) || 0;
      }
      if (va < vb) return sortDir === "asc" ? -1 : 1;
      if (va > vb) return sortDir === "asc" ? 1 : -1;
      return 0;
    });
    return sorted;
  }, [trades, sortField, sortDir]);

  const stats = useMemo(() => {
    if (!trades || trades.length === 0) return null;
    const wins = [];
    const losses = [];
    let longCount = 0;
    let shortCount = 0;
    let longPnl = 0;
    let shortPnl = 0;
    let bestTrade = -Infinity;
    let worstTrade = Infinity;
    let maxConsecWin = 0;
    let maxConsecLoss = 0;
    let curWin = 0;
    let curLoss = 0;

    const sorted = [...trades].sort(
      (a, b) => new Date(a.exitTime).getTime() - new Date(b.exitTime).getTime()
    );

    sorted.forEach((t) => {
      const p = Number(t.profit) || 0;
      const isLong = (t.type || "").toUpperCase() === "LONG";
      if (isLong) { longCount++; longPnl += p; }
      else { shortCount++; shortPnl += p; }
      if (p > bestTrade) bestTrade = p;
      if (p < worstTrade) worstTrade = p;
      if (p > 0) {
        wins.push(p);
        curWin++;
        curLoss = 0;
        if (curWin > maxConsecWin) maxConsecWin = curWin;
      } else if (p < 0) {
        losses.push(p);
        curLoss++;
        curWin = 0;
        if (curLoss > maxConsecLoss) maxConsecLoss = curLoss;
      } else {
        curWin = 0;
        curLoss = 0;
      }
    });

    const avgWin = wins.length > 0 ? wins.reduce((a, b) => a + b, 0) / wins.length : 0;
    const avgLoss = losses.length > 0 ? losses.reduce((a, b) => a + b, 0) / losses.length : 0;
    const expectancy = trades.length > 0
      ? (baseMetrics.winRate / 100) * avgWin + (1 - baseMetrics.winRate / 100) * avgLoss
      : 0;

    return {
      avgWin,
      avgLoss,
      bestTrade: bestTrade === -Infinity ? 0 : bestTrade,
      worstTrade: worstTrade === Infinity ? 0 : worstTrade,
      longCount,
      shortCount,
      longPnl,
      shortPnl,
      maxConsecWin,
      maxConsecLoss,
      expectancy,
    };
  }, [trades, baseMetrics]);

  const fmt = (v) => {
    if (v === undefined || v === null || isNaN(v)) return "-";
    const sign = v > 0 ? "+" : "";
    return `${sign}${v.toFixed(2)}`;
  };

  const formatTime = (value) => {
    if (!value) return "-";
    const date = new Date(value);
    if (isNaN(date.getTime())) return "-";
    return date.toLocaleString("en-GB", {
      day: "2-digit", month: "short", hour: "2-digit", minute: "2-digit", hour12: false,
    });
  };

  const formatPrice = (value) => {
    if (value === undefined || value === null || Number.isNaN(Number(value))) return "-";
    return Number(value).toLocaleString("en-IN", { minimumFractionDigits: 2, maximumFractionDigits: 2 });
  };

  const formatProfit = (value) => {
    if (value === undefined || value === null || Number.isNaN(Number(value))) return "-";
    const num = Number(value);
    const sign = num >= 0 ? "+" : "";
    return `${sign}${num.toFixed(2)}`;
  };

  const formatPercent = (value) => {
    if (value === undefined || value === null || Number.isNaN(Number(value))) return "-";
    return `${Number(value).toFixed(2)}%`;
  };

  const getReasonBadge = (reason) => {
    const badges = {
      PROFIT_TARGET: { label: "Target", className: "badge-profit" },
      STOP_LOSS: { label: "SL", className: "badge-loss" },
      TRAILING_STOP: { label: "Trail", className: "badge-trail" },
      SIGNAL: { label: "Signal", className: "badge-signal" },
    };
    const badge = badges[reason] || { label: reason || "-", className: "" };
    return <span className={`reason-badge ${badge.className}`}>{badge.label}</span>;
  };

  return (
    <div className={`equity-drawer ${open ? "is-open" : ""}`}>
      <div className="equity-drawer__header">
        <div className="equity-drawer__tabs">
          <button
            type="button"
            className={`equity-drawer__tab ${tab === "equity" ? "is-active" : ""}`}
            onClick={() => setTab("equity")}
          >
            Equity Curve
          </button>
          <button
            type="button"
            className={`equity-drawer__tab ${tab === "trades" ? "is-active" : ""}`}
            onClick={() => setTab("trades")}
          >
            Trades ({trades?.length || 0})
          </button>
        </div>
        <button
          type="button"
          className="equity-drawer__close"
          onClick={() => dispatch(toggleEquityDrawer())}
          aria-label="Close equity drawer"
        >
          <X size={14} weight="regular" />
        </button>
      </div>

      {tab === "equity" && (
        <div className="equity-drawer__body">
          <div className="equity-drawer__chart">
            <EquityCurve trades={trades} height={300} />
          </div>
          {stats && (
            <div className="equity-drawer__stats">
              <div className="eq-stats-section">
                <div className="eq-stats-section__title">Performance</div>
                <div className="eq-stat-row">
                  <span className="eq-stat-label">Net P&L</span>
                  <span className={`eq-stat-value ${baseMetrics.totalPnl >= 0 ? "positive" : "negative"}`}>
                    {fmt(baseMetrics.totalPnl)} pts
                  </span>
                </div>
                <div className="eq-stat-row">
                  <span className="eq-stat-label">Profit Factor</span>
                  <span className="eq-stat-value">{baseMetrics.profitFactor === Infinity ? "INF" : baseMetrics.profitFactor?.toFixed(2) ?? "-"}</span>
                </div>
                <div className="eq-stat-row">
                  <span className="eq-stat-label">Max Drawdown</span>
                  <span className="eq-stat-value negative">{fmt(-baseMetrics.maxDrawdown)}</span>
                </div>
                <div className="eq-stat-row">
                  <span className="eq-stat-label">Expectancy</span>
                  <span className={`eq-stat-value ${stats.expectancy >= 0 ? "positive" : "negative"}`}>
                    {fmt(stats.expectancy)}
                  </span>
                </div>
              </div>

              <div className="eq-stats-section">
                <div className="eq-stats-section__title">Win / Loss</div>
                <div className="eq-stat-row">
                  <span className="eq-stat-label">Win Rate</span>
                  <span className="eq-stat-value">{baseMetrics.winRate?.toFixed(1)}% <span className="eq-stat-sub">{baseMetrics.winCount}/{baseMetrics.totalTrades}</span></span>
                </div>
                <div className="eq-stat-row">
                  <span className="eq-stat-label">Avg Win</span>
                  <span className="eq-stat-value positive">{fmt(stats.avgWin)}</span>
                </div>
                <div className="eq-stat-row">
                  <span className="eq-stat-label">Avg Loss</span>
                  <span className="eq-stat-value negative">{fmt(stats.avgLoss)}</span>
                </div>
                <div className="eq-stat-row">
                  <span className="eq-stat-label">Best Trade</span>
                  <span className="eq-stat-value positive">{fmt(stats.bestTrade)}</span>
                </div>
                <div className="eq-stat-row">
                  <span className="eq-stat-label">Worst Trade</span>
                  <span className="eq-stat-value negative">{fmt(stats.worstTrade)}</span>
                </div>
                <div className="eq-stat-row">
                  <span className="eq-stat-label">Consec. Win</span>
                  <span className="eq-stat-value">{stats.maxConsecWin}</span>
                </div>
                <div className="eq-stat-row">
                  <span className="eq-stat-label">Consec. Loss</span>
                  <span className="eq-stat-value">{stats.maxConsecLoss}</span>
                </div>
              </div>

              <div className="eq-stats-section">
                <div className="eq-stats-section__title">Direction</div>
                <div className="eq-stat-row">
                  <span className="eq-stat-label">Long</span>
                  <span className="eq-stat-value">{stats.longCount} <span className={`eq-stat-sub ${stats.longPnl >= 0 ? "positive" : "negative"}`}>({fmt(stats.longPnl)})</span></span>
                </div>
                <div className="eq-stat-row">
                  <span className="eq-stat-label">Short</span>
                  <span className="eq-stat-value">{stats.shortCount} <span className={`eq-stat-sub ${stats.shortPnl >= 0 ? "positive" : "negative"}`}>({fmt(stats.shortPnl)})</span></span>
                </div>
              </div>

              <div className="eq-stats-section">
                <div className="eq-stats-section__title">Exit Breakdown</div>
                <div className="eq-stat-row">
                  <span className="eq-stat-label">Target</span>
                  <span className="eq-stat-value">{baseMetrics.profitTargetCount}</span>
                </div>
                <div className="eq-stat-row">
                  <span className="eq-stat-label">Trailing</span>
                  <span className="eq-stat-value">{baseMetrics.trailingStopCount}</span>
                </div>
                <div className="eq-stat-row">
                  <span className="eq-stat-label">Stop Loss</span>
                  <span className="eq-stat-value">{baseMetrics.stopLossCount}</span>
                </div>
                <div className="eq-stat-row">
                  <span className="eq-stat-label">Signal</span>
                  <span className="eq-stat-value">{baseMetrics.signalCount}</span>
                </div>
              </div>
            </div>
          )}
        </div>
      )}

      {tab === "trades" && (
        <>
          <div className="equity-drawer__body equity-drawer__body--trades">
            {!trades || trades.length === 0 ? (
              <div className="equity-drawer__empty">No trades yet.</div>
            ) : (
              <table className="trades-table">
                <thead>
                  <tr>
                    <th>#</th>
                    <th className="sortable" role="button" onClick={() => toggleSort("entryTime")}>Entry {sortIndicator("entryTime")}</th>
                    <th className="sortable" role="button" onClick={() => toggleSort("exitTime")}>Exit {sortIndicator("exitTime")}</th>
                    <th className="sortable" role="button" onClick={() => toggleSort("type")}>Type {sortIndicator("type")}</th>
                    <th className="sortable" role="button" onClick={() => toggleSort("entryPrice")}>Entry Price {sortIndicator("entryPrice")}</th>
                    <th className="sortable" role="button" onClick={() => toggleSort("exitPrice")}>Exit Price {sortIndicator("exitPrice")}</th>
                    <th className="sortable" role="button" onClick={() => toggleSort("profit")}>P&L {sortIndicator("profit")}</th>
                    <th className="sortable" role="button" onClick={() => toggleSort("peakProfit")}>Peak Profit {sortIndicator("peakProfit")}</th>
                    <th className="sortable" role="button" onClick={() => toggleSort("peakLoss")}>Peak Loss {sortIndicator("peakLoss")}</th>
                    <th>Reason</th>
                  </tr>
                </thead>
                <tbody>
                  {sortedTrades.map((trade, idx) => {
                    const profit = trade?.profit ?? 0;
                    const profitClass = profit > 0 ? "profit-positive" : profit < 0 ? "profit-negative" : "";
                    return (
                      <tr key={idx} className="trade-entry-row">
                        <td className="eq-trade-sr">{idx + 1}</td>
                        <td className="trade-entry-crosshair-container">
                          <Crosshair size={14} weight="regular" aria-hidden="true" className="trade-entry-crosshair"
                            onClick={() => {
                              window.dispatchEvent(new CustomEvent("onDateChange", { detail: { date: trade?.entryTime } }));
                            }}
                          />
                          {formatTime(trade?.entryTime)}
                        </td>
                        <td>{formatTime(trade?.exitTime)}</td>
                        <td className={`trade-type ${(trade?.type || "").toLowerCase()}`}>
                          {trade?.type ?? "-"}
                        </td>
                        <td>{formatPrice(trade?.entryPrice)}</td>
                        <td>{formatPrice(trade?.exitPrice)}</td>
                        <td className={profitClass}>
                          {formatProfit(profit)}
                          <span className="profit-pct"> ({formatPercent(trade?.profitPct)})</span>
                        </td>
                        <td>{formatPrice(trade?.peakProfit)}</td>
                        <td>{formatPrice(trade?.peakLoss)}</td>
                        <td>{getReasonBadge(trade?.reason)}</td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            )}
          </div>
          <div className="trades-drawer__footer metrics">
            <div>
              <div className="metric-label">Net P&L</div>
              <div className={`metric-value ${baseMetrics.totalPnl >= 0 ? "positive" : "negative"}`}>
                {formatProfit(baseMetrics.totalPnl ?? 0)} pts
              </div>
            </div>
            <div>
              <div className="metric-label">Max Drawdown</div>
              <div className="metric-value negative">
                {formatPrice(baseMetrics.maxDrawdown ?? 0)} pts
              </div>
            </div>
            <div>
              <div className="metric-label">Total Trades</div>
              <div className="metric-value">{baseMetrics.totalTrades ?? trades.length}</div>
            </div>
            <div>
              <div className="metric-label">Win Rate</div>
              <div className="metric-value">
                {formatPercent(baseMetrics.winRate ?? 0)}{" "}
                <span className="metric-sub">{baseMetrics.winCount ?? 0}/{baseMetrics.totalTrades ?? trades.length}</span>
              </div>
            </div>
            <div>
              <div className="metric-label">Profit Factor</div>
              <div className="metric-value">{baseMetrics.profitFactor?.toFixed(2) ?? "-"}</div>
            </div>
            <div>
              <div className="metric-label">Exit Breakdown</div>
              <div className="metric-value metric-breakdown">
                <span className="badge-profit">T:{baseMetrics.profitTargetCount ?? 0}</span>
                <span className="badge-trail">Tr:{baseMetrics.trailingStopCount ?? 0}</span>
                <span className="badge-loss">SL:{baseMetrics.stopLossCount ?? 0}</span>
                <span className="badge-signal">S:{baseMetrics.signalCount ?? 0}</span>
              </div>
            </div>
          </div>
        </>
      )}
    </div>
  );
}

export default EquityDrawer;
