import {
  X,
  ArrowClockwise,
  TerminalWindow,
  ArrowsOutSimple,
  ArrowsInSimple,
  MagnifyingGlass,
} from "phosphor-react";

function LogDrawer({
  open,
  size,
  logs,
  isLoading,
  error,
  onClose,
  onRefresh,
  onToggleSize,
  filterValue,
  onFilterChange,
}) {
  return (
    <div
      className={`log-drawer ${open ? "is-open" : ""} ${size === "max" ? "is-max" : ""}`}
      aria-live="polite"
    >
      <div className="log-drawer__header">
        <div className="log-drawer__title">
          <TerminalWindow size={14} weight="regular" />
          <span>Telemetry Logs</span>

        </div>
        <div className="log-drawer__actions">
          <div className="log-drawer__search">
            <MagnifyingGlass size={12} weight="regular" aria-hidden="true" />
            <input
              type="text"
              placeholder="Search logs"
              value={filterValue}
              onChange={(e) => onFilterChange(e.target.value)}
            />
          </div>
          <button
            type="button"
            className="status-bar__button"
            onClick={onToggleSize}
            aria-label={size === "max" ? "Minimize logs panel" : "Maximize logs panel"}
          >
            {size === "max" ? (
              <ArrowsInSimple size={14} weight="regular" />
            ) : (
              <ArrowsOutSimple size={14} weight="regular" />
            )}
          </button>
          <button type="button" className="status-bar__button" onClick={onRefresh} aria-label="Refresh logs">
            <ArrowClockwise size={14} weight="regular" />
          </button>
          <button type="button" className="status-bar__button" onClick={onClose} aria-label="Close logs">
            <X size={14} weight="regular" />
          </button>
        </div>
      </div>
      <div className="log-drawer__body">
        {isLoading && <div className="log-line log-line--muted">Loading logs…</div>}
        {error && <div className="log-line log-line--error">{error}</div>}
        {logs.length === 0 && !isLoading && !error && (
          <div className="log-line log-line--muted">No logs yet.</div>
        )}
        {logs.map((line, idx) => (
          <div key={idx} className="log-line">
            {line}
          </div>
        ))}
      </div>
    </div>
  );
}

export default LogDrawer;
