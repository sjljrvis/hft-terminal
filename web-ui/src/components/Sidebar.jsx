import { NavLink } from "react-router-dom";
import {
  Broadcast,
  TestTube,
  Database,
  Gear,
} from "phosphor-react";

const NAV_ITEMS = [
  { key: "live", label: "Live", icon: Broadcast, path: "/live" },
  { key: "backtest", label: "Backtest", icon: TestTube, path: "/backtest" },
  { key: "query", label: "Query", icon: Database, path: "/query" },
  { key: "settings", label: "Settings", icon: Gear, path: "/settings" },
];

function Sidebar({ collapsed }) {
  return (
    <aside className={`sidebar ${collapsed ? "is-collapsed" : ""}`}>
      <nav className="sidebar__nav" aria-label="Primary">
        {NAV_ITEMS.map((item) => {
          return (
            <NavLink
              key={item.key}
              to={item.path}
              className={({ isActive }) =>
                `sidebar__item ${isActive ? "is-active" : ""}`
              }
            >
              <span className="sidebar__icon" aria-hidden="true">
                <item.icon size={16} weight="regular" />
              </span>
              <span className="sidebar__label">{item.label}</span>
            </NavLink>
          );
        })}
      </nav>
    </aside>
  );
}

export default Sidebar;
