import { X, MagnifyingGlass } from "phosphor-react";

function TradesDrawer({
  open,
  onClose,
  trades = [],
  filterValue = "",
  onFilterChange = () => { },
  sortValue = "time-desc",
  onSortChange = () => { },
  metrics = {},
}) {
  const formatTime = (value) => {
    if (!value) return "-";
    if (value instanceof Date) {
      return value.toLocaleString("en-GB", { hour12: false });
    }
    return String(value);
  };

  const formatCurrency = (value) => {
    if (value === undefined || value === null || Number.isNaN(Number(value))) return "-";
    return `${Number(value).toLocaleString("en-IN", { minimumFractionDigits: 2, maximumFractionDigits: 2 })} INR`;
  };

  const formatPercent = (value) => {
    if (value === undefined || value === null || Number.isNaN(Number(value))) return "-";
    return `${Number(value).toFixed(2)}%`;
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

  return (
    <div className={`trades-drawer ${open ? "is-open" : ""}`} aria-live="polite">
      <div className="trades-drawer__header">
        <div className="trades-drawer__title">Recent Trades</div>
        <div className="trades-drawer__actions">
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
                  onClick={() => toggleSort("time")}
                  role="button"
                  aria-sort={sortValue.startsWith("time-") ? sortValue.endsWith("asc") ? "ascending" : "descending" : "none"}
                >
                  Time {renderSortIndicator("time")}
                </th>
                <th
                  className="sortable"
                  onClick={() => toggleSort("side")}
                  role="button"
                  aria-sort={sortValue.startsWith("side-") ? sortValue.endsWith("asc") ? "ascending" : "descending" : "none"}
                >
                  Side {renderSortIndicator("side")}
                </th>
                <th
                  className="sortable"
                  onClick={() => toggleSort("price")}
                  role="button"
                  aria-sort={sortValue.startsWith("price-") ? sortValue.endsWith("asc") ? "ascending" : "descending" : "none"}
                >
                  Price {renderSortIndicator("price")}
                </th>
                <th
                  className="sortable"
                  onClick={() => toggleSort("qty")}
                  role="button"
                  aria-sort={sortValue.startsWith("qty-") ? sortValue.endsWith("asc") ? "ascending" : "descending" : "none"}
                >
                  Qty {renderSortIndicator("qty")}
                </th>
              </tr>
            </thead>
            <tbody>
              {trades.map((trade, idx) => (
                <tr key={idx}>
                  <td>{formatTime(trade?.time)}</td>
                  <td className={`trade-side ${trade?.side ?? ""}`}>{trade?.side ?? "-"}</td>
                  <td>{trade?.price ?? "-"}</td>
                  <td>{trade?.qty ?? "-"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}

      </div>
      <div className="trades-drawer__footer metrics">
        <div>
          <div className="metric-label">Total P&amp;L</div>
          <div className="metric-value positive">
            {formatCurrency(metrics.totalPnl ?? 0)}{" "}
            <span className="metric-sub">+{formatPercent(metrics.totalPnlPct ?? 0)}</span>
          </div>
        </div>
        <div>
          <div className="metric-label">Max equity drawdown</div>
          <div className="metric-value">
            {formatCurrency(metrics.maxDrawdown ?? 0)}{" "}
            <span className="metric-sub">{formatPercent(metrics.maxDrawdownPct ?? 0)}</span>
          </div>
        </div>
        <div>
          <div className="metric-label">Total trades</div>
          <div className="metric-value">{metrics.totalTrades ?? trades.length}</div>
        </div>
        <div>
          <div className="metric-label">Profitable trades</div>
          <div className="metric-value">
            {formatPercent(metrics.profitableTradesPct ?? 0)}{" "}
            <span className="metric-sub">{metrics.profitableTradesCount ?? ""}</span>
          </div>
        </div>
        <div>
          <div className="metric-label">Profit factor</div>
          <div className="metric-value">{metrics.profitFactor ?? "-"}</div>
        </div>
      </div>
    </div>
  );
}

export default TradesDrawer;