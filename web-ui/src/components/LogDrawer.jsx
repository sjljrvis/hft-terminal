import {
  X,
  ArrowClockwise,
  TerminalWindow,
  ArrowsOutSimple,
  ArrowsInSimple,
  MagnifyingGlass,
} from "phosphor-react";
import { useAppDispatch, useAppSelector } from "../store/hooks";
import {
  selectLogsDrawerOpen,
  selectLogSize,
  closeLogsDrawer,
  toggleLogSize,
} from "../store/slices/uiSlice";
import {
  selectFilteredLogs,
  selectLogsLoading,
  selectLogsError,
  selectLogsFilter,
  setFilter,
  fetchLogs,
} from "../store/slices/logsSlice";

function LogDrawer() {
  const dispatch = useAppDispatch();
  const open = useAppSelector(selectLogsDrawerOpen);
  const size = useAppSelector(selectLogSize);
  const logs = useAppSelector(selectFilteredLogs);
  const isLoading = useAppSelector(selectLogsLoading);
  const error = useAppSelector(selectLogsError);
  const filterValue = useAppSelector(selectLogsFilter);

  const onClose = () => dispatch(closeLogsDrawer());
  const onRefresh = () => dispatch(fetchLogs());
  const onToggleSize = () => dispatch(toggleLogSize());
  const onFilterChange = (value) => dispatch(setFilter(value));
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
