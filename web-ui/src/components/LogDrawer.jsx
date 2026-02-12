import {
  X,
  ArrowClockwise,
  TerminalWindow,
  ArrowsOutSimple,
  ArrowsInSimple,
  MagnifyingGlass,
} from "phosphor-react";
import { useEffect } from "react";
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
  addLog,
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

  // Stream executor/broker events to logs via WebSocket while drawer is open.
  useEffect(() => {
    if (!open) return undefined;

    const wsUrl = "ws://localhost:5001/ws/events";
    let ws;
    let closedByCleanup = false;

    const formatLine = (msg) => {
      if (!msg || typeof msg !== "object") return `[WS] : ${String(msg)}`;
      const type = msg.type || "unknown";
      const data = msg.data;
      const timestamp = msg.timestamp ? new Date(msg.timestamp).toLocaleString() : new Date().toLocaleString();
      if (type === "log" && data?.message) {
        return `${timestamp} [LOG] : ${data.message}`;
      }
      if (type === "event" && data) {
        const kind = data.kind || "-";
        const evType = data.type || "-";
        const price = data.entryPrice ?? "-";
        const reason = data.reason ? ` (${data.reason})` : "";
        return `${timestamp} [EVENT] : ${evType} ${kind} @ ${price}${reason}`;
      }
      return `${timestamp} [${type.toUpperCase()}] : ${JSON.stringify(data ?? msg)}`;
    };

    const connect = () => {
      ws = new WebSocket(wsUrl);
      const timestamp = new Date().toLocaleString();
      dispatch(addLog(`${timestamp} [WS] : connecting to ${wsUrl}`));

      ws.onopen = () => {
        dispatch(addLog(`${timestamp} [WS] : connected`));
      };

      ws.onmessage = (evt) => {
        const raw = String(evt.data ?? "");
        // Server may batch messages separated by newline.
        const frames = raw.split("\n").filter(Boolean);
        for (const frame of frames) {
          try {
            const parsed = JSON.parse(frame);
            dispatch(addLog(formatLine(parsed)));
          } catch {
            // Not JSON, still show.
            dispatch(addLog(`${timestamp} [WS] : ${frame}`));
          }
        }
      };

      ws.onerror = () => {
        dispatch(addLog(`${timestamp} [WS] : error`));
      };

      ws.onclose = () => {
        dispatch(addLog(`${timestamp} [WS] : disconnected`));
        // Simple reconnect loop while drawer is open.
        if (!closedByCleanup) {
          setTimeout(() => {
            if (!closedByCleanup) connect();
          }, 1000);
        }
      };
    };

    connect();

    return () => {
      closedByCleanup = true;
      try {
        ws?.close();
      } catch {
        // ignore
      }
    };
  }, [open, dispatch]);

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
