import { SidebarSimple, TerminalWindow } from "phosphor-react";

function StatusBar({ sidebarCollapsed, onToggleSidebar, logsOpen, onToggleLogs }) {
  return (
    <footer className="status-bar" aria-label="Status bar">
      <div className="status-bar__left">
        <button
          type="button"
          className="status-bar__button"
          onClick={onToggleSidebar}
          aria-pressed={sidebarCollapsed}
        >
          <SidebarSimple
            size={14}
            weight="regular"
            className={`sidebar-toggle-icon ${sidebarCollapsed ? "is-collapsed" : ""}`}
          />
        </button>
        <button
          type="button"
          className="status-bar__button"
          onClick={onToggleLogs}
          aria-pressed={logsOpen}
          aria-label="Toggle logs drawer"
        >
          <TerminalWindow size={14} weight="regular" className={logsOpen ? "status-button--active" : ""} />
        </button>
      </div>
      <div className="status-bar__right">
        <span className="status-dot" aria-hidden="true"></span>
        <span>Connected</span>
      </div>
    </footer>
  );
}

export default StatusBar;
