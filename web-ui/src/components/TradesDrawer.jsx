import { X, MagnifyingGlass, Download, Crosshair } from "phosphor-react";

function TradesDrawer({
  open,
  onClose,
  trades = [],
  filterValue = "",
  onFilterChange = () => { },
  sortValue = "entryTime-desc",
  onSortChange = () => { },
  metrics = {},
}) {
  const formatTime = (value) => {
    if (!value) return "-";
    const date = new Date(value);
    if (isNaN(date.getTime())) return "-";
    return date.toLocaleString("en-GB", {
      day: "2-digit",
      month: "short",
      hour: "2-digit",
      minute: "2-digit",
      hour12: false,
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

  const toggleSort = (field) => {
    const [curField, curDir] = (sortValue || "").split("-");
    const nextDir = curField === field && curDir === "asc" ? "desc" : "asc";
    onSortChange(`${field}-${nextDir}`);
  };

  const renderSortIndicator = (field) => {
    const [curField, curDir] = (sortValue || "").split("-");
    if (curField !== field) return null;
    return <span className="sort-indicator">{curDir === "asc" ? "↑" : "↓"}</span>;
  };

  const downloadCSV = () => {
    const csv = trades.map((trade) => `${trade.entryTime},${trade.exitTime},${trade.type},${trade.entryPrice},${trade.exitPrice},${trade.profit}, ${trade.peakProfit}, ${trade.peakLoss}, ${trade.reason}`).join("\n");
    const blob = new Blob([csv], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "trades.csv";
    a.click();
  };

  return (
    <div className={`trades-drawer ${open ? "is-open" : ""}`} aria-live="polite">
      <div className="trades-drawer__header">
        <div className="trades-drawer__title">Trades ({trades.length})</div>
        <div className="trades-drawer__actions">
          <button type="button" className="trades-drawer__download" aria-label="Download trades" onClick={downloadCSV}>
            <Download size={14} weight="regular" aria-hidden="true" /> Download
          </button>
          <div className="trades-drawer__search">
            <MagnifyingGlass size={12} weight="regular" aria-hidden="true" />
            <input
              type="text"
              placeholder="Filter trades"
              value={filterValue}
              onChange={(e) => onFilterChange(e.target.value)}
            />
          </div>
        </div>
        <button type="button" className="trades-drawer__close" onClick={onClose} aria-label="Close trades">
          <X size={16} weight="regular" />
        </button>
      </div>
      <div className="trades-drawer__body">
        {trades.length === 0 ? (
          <div className="trades-drawer__empty">No trades yet.</div>
        ) : (
          <table className="trades-table">
            <thead>
              <tr>
                <th
                  className="sortable"
                  onClick={() => toggleSort("entryTime")}
                  role="button"
                >


                  Entry {renderSortIndicator("entryTime")}
                </th>
                <th
                  className="sortable"
                  onClick={() => toggleSort("exitTime")}
                  role="button"
                >
                  Exit {renderSortIndicator("exitTime")}
                </th>
                <th
                  className="sortable"
                  onClick={() => toggleSort("type")}
                  role="button"
                >
                  Type {renderSortIndicator("type")}
                </th>
                <th
                  className="sortable"
                  onClick={() => toggleSort("entryPrice")}
                  role="button"
                >
                  Entry ₹ {renderSortIndicator("entryPrice")}
                </th>
                <th
                  className="sortable"
                  onClick={() => toggleSort("exitPrice")}
                  role="button"
                >
                  Exit ₹ {renderSortIndicator("exitPrice")}
                </th>
                <th
                  className="sortable"
                  onClick={() => toggleSort("profit")}
                  role="button"
                >
                  P&L {renderSortIndicator("profit")}
                </th>
                <th
                  className="sortable"
                  onClick={() => toggleSort("peakProfit")}
                  role="button"
                >Peak Profit {renderSortIndicator("peakProfit")}</th>
                <th
                  className="sortable"
                  onClick={() => toggleSort("peakLoss")}
                  role="button"
                >Peak Loss {renderSortIndicator("peakLoss")}</th>
                <th>Reason</th>
              </tr>
            </thead>
            <tbody>
              {trades.map((trade, idx) => {
                const profit = trade?.profit ?? 0;
                const profitClass = profit > 0 ? "profit-positive" : profit < 0 ? "profit-negative" : "";
                return (
                  <tr key={idx} className="trade-entry-row">
                    <td className="trade-entry-crosshair-container">
                        <Crosshair size={16} weight="regular" aria-hidden="true" className="trade-entry-crosshair"
                          onClick={() => {
                            // send event to chart panel to focus on this date
                            window.dispatchEvent(new CustomEvent("onDateChange", { detail: { date: trades[0].entryTime } }));
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
                    <td>
                      {formatPrice(trade?.peakProfit)}
                    </td>
                    <td>
                      {formatPrice(trade?.peakLoss)}
                    </td>
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
          <div className="metric-label">Net P&amp;L</div>
          <div className={`metric-value ${metrics.totalPnl >= 0 ? "positive" : "negative"}`}>
            {formatProfit(metrics.totalPnl ?? 0)} pts
          </div>
        </div>
        <div>
          <div className="metric-label">Max Drawdown</div>
          <div className="metric-value negative">
            {formatPrice(metrics.maxDrawdown ?? 0)} pts
          </div>
        </div>
        <div>
          <div className="metric-label">Total Trades</div>
          <div className="metric-value">{metrics.totalTrades ?? trades.length}</div>
        </div>
        <div>
          <div className="metric-label">Win Rate</div>
          <div className="metric-value">
            {formatPercent(metrics.winRate ?? 0)}{" "}
            <span className="metric-sub">{metrics.winCount ?? 0}/{metrics.totalTrades ?? trades.length}</span>
          </div>
        </div>
        <div>
          <div className="metric-label">Profit Factor</div>
          <div className="metric-value">{metrics.profitFactor?.toFixed(2) ?? "-"}</div>
        </div>
        <div>
          <div className="metric-label">Exit Breakdown</div>
          <div className="metric-value metric-breakdown">
            <span className="badge-profit">T:{metrics.profitTargetCount ?? 0}</span>
            <span className="badge-trail">Tr:{metrics.trailingStopCount ?? 0}</span>
            <span className="badge-loss">SL:{metrics.stopLossCount ?? 0}</span>
            <span className="badge-signal">S:{metrics.signalCount ?? 0}</span>
          </div>
        </div>
      </div>
    </div>
  );
}

export default TradesDrawer;