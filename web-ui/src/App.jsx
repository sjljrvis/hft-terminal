import { useEffect, useState } from "react";
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import "./style.css";
import Header from "./components/Header";
import StatusBar from "./components/StatusBar";
import DashboardCard from "./components/DashboardCard";
import Sidebar from "./components/Sidebar";
import LogDrawer from "./components/LogDrawer";
import ChartPanel from "./components/ChartPanel";
import BacktestPanel from "./components/BacktestPanel";
import TradesDrawer from "./components/TradesDrawer";
import { apiBase } from "./api";

const DUMMY_TRADES = [
  {
    time: new Date("2026-01-19 10:00:00"),
    side: "buy",
    price: 100,
    qty: 100,
  },
  {
    time: new Date("2026-01-27 10:00:00"),
    side: "buy",
    price: 100,
    qty: 100,
  },
  {
    time: new Date("2026-01-27 10:01:00"),
    side: "sell",
    price: 101,
    qty: 100,
  },
  {
    time: new Date("2026-01-27 10:02:00"),
    side: "buy",
    price: 102,
    qty: 100,
  },
  {
    time: new Date("2026-01-27 10:03:00"),
    side: "sell",
    price: 103,
    qty: 100,
  },

  // add 100 more trades
  ...Array.from({ length: 1 }, (_, i) => ({
    time: new Date(`2026-01-27 10:0${i}:00`),
    side: i % 2 === 0 ? "buy" : "sell",
    price: 100 + i,
    qty: 100,
  })),
];

const DUMMY_METRICS = {
  totalPnl: 1410.65,
  totalPnlPct: 0.14,
  maxDrawdown: 189.3,
  maxDrawdownPct: 0.02,
  totalTrades: 271,
  profitableTradesPct: 90.41,
  profitableTradesCount: "245/271",
  profitFactor: 1.825,
};

function App() {
  const [theme, setTheme] = useState("light");
  const [sidebarCollapsed, setSidebarCollapsed] = useState(true);
  const [logsOpen, setLogsOpen] = useState(false);
  const [logSize, setLogSize] = useState("min"); // "min" | "max"
  const [logs, setLogs] = useState([]);
  const [logsError, setLogsError] = useState("");
  const [logsLoading, setLogsLoading] = useState(false);
  const [logFilter, setLogFilter] = useState("");
  const [openTradesDrawer, setOpenTradesDrawer] = useState(false);
  const [trades, setTrades] = useState(DUMMY_TRADES);
  const [tradesFilter, setTradesFilter] = useState("");
  const [tradesSort, setTradesSort] = useState("time-desc");

  const fetchLogs = async () => {
    setLogsLoading(true);
    try {
      const res = await fetch(`${apiBase}/logs`);
      if (!res.ok) {
        throw new Error(`HTTP ${res.status}`);
      }
      const data = await res.json();
      const lines = Array.isArray(data) ? data : data?.lines ?? [];
      const normalized = lines.map((line) => String(line));
      setLogs(normalized.slice(-200));
      setLogsError("");
    } catch (err) {
      const fallback = `[local] ${new Date().toISOString()} — backend logs unavailable`;
      setLogs((prev) => [...prev.slice(-180), fallback]);
      setLogsError("Unable to load logs from backend");
    } finally {
      setLogsLoading(false);
    }
  };

  useEffect(() => {
    if (!logsOpen) return undefined;
    fetchLogs();
    const id = setInterval(fetchLogs, 4000);
    return () => clearInterval(id);
  }, [logsOpen]);

  useEffect(() => {
    document.body.classList.remove("theme-light", "theme-dark");
    document.body.classList.add(theme === "dark" ? "theme-dark" : "theme-light");
  }, [theme]);

  const filteredSortedTrades = (() => {
    const term = tradesFilter.trim().toLowerCase();
    const filtered = term
      ? trades.filter((t) =>
          [t?.time, t?.side, t?.price, t?.qty]
            .map((v) => (v === undefined || v === null ? "" : String(v).toLowerCase()))
            .some((v) => v.includes(term))
        )
      : trades;

    const [field, dir] = (tradesSort || "time-desc").split("-");
    const sorted = [...filtered].sort((a, b) => {
      const mult = dir === "asc" ? 1 : -1;
      switch (field) {
        case "price":
          return mult * ((Number(a?.price) || 0) - (Number(b?.price) || 0));
        case "qty":
          return mult * ((Number(a?.qty) || 0) - (Number(b?.qty) || 0));
        case "side":
          return mult * String(a?.side || "").localeCompare(String(b?.side || ""));
        case "time":
        default: {
          const ta = new Date(a?.time || "").getTime() || 0;
          const tb = new Date(b?.time || "").getTime() || 0;
          return mult * (ta - tb);
        }
      }
    });

    return sorted;
  })();

  return (
    <BrowserRouter>
      <div className="app-shell">
        <Header theme={theme} onThemeChange={setTheme} />
        <div className="app-body">
          <Sidebar collapsed={sidebarCollapsed} />
          <main className="app-main">
            <Routes>
              <Route path="/" element={<Navigate to="/chart" replace />} />
              <Route path="/chart" element={<ChartPanel setTrades={setTrades} openTradesDrawer={setOpenTradesDrawer} />} />
              <Route path="/backtest" element={<BacktestPanel />} />
              <Route path="*" element={<DashboardCard />} />
            </Routes>
          </main>
        </div>
        <TradesDrawer
          open={openTradesDrawer}
          onClose={() => setOpenTradesDrawer(false)}
          trades={filteredSortedTrades}
          filterValue={tradesFilter}
          onFilterChange={setTradesFilter}
          sortValue={tradesSort}
          onSortChange={setTradesSort}
          metrics={DUMMY_METRICS}
        />
        <StatusBar
          sidebarCollapsed={sidebarCollapsed}
          onToggleSidebar={() => setSidebarCollapsed((v) => v)}
          logsOpen={logsOpen}
          onToggleLogs={() => setLogsOpen((v) => !v)}
        />
        <LogDrawer
          open={logsOpen}
          size={logSize}
          logs={
            logFilter
              ? logs.filter((line) => line.toLowerCase().includes(logFilter.toLowerCase()))
              : logs
          }
          isLoading={logsLoading}
          error={logsError}
          onClose={() => {
            setLogsOpen(false);
            setLogSize("min");
          }}
          onRefresh={fetchLogs}
          filterValue={logFilter}
          onFilterChange={setLogFilter}
          onToggleSize={() => setLogSize((s) => (s === "min" ? "max" : "min"))}
        />
      </div>
    </BrowserRouter>
  );
}

export default App;
