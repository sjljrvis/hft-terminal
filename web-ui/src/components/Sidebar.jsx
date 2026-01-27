import {
  LinkSimpleHorizontal,
  Atom,
  GlobeSimple,
  GearSix,
  ChartLine,
  TestTube,
  Money,
} from "phosphor-react";

const NAV_ITEMS = [
  { key: "Live Charts", label: "REST", icon: ChartLine },
  { key: "Backtest", label: "GraphQL", icon: TestTube },
  { key: "Risk Management", label: "Realtime", icon: Money },
];

function Sidebar({ collapsed, activeKey = "settings" }) {
  return (
    <aside className={`sidebar ${collapsed ? "is-collapsed" : ""}`}>
      <nav className="sidebar__nav" aria-label="Primary">
        {NAV_ITEMS.map((item) => {
          const isActive = item.key === activeKey;
          return (
            <button
              key={item.key}
              className={`sidebar__item ${isActive ? "is-active" : ""}`}
              type="button"
            >
              <span className="sidebar__icon" aria-hidden="true">
                <item.icon size={16} weight="regular" />
              </span>
              <span className="sidebar__label">{item.label}</span>
            </button>
          );
        })}
      </nav>
    </aside>
  );
}

export default Sidebar;
