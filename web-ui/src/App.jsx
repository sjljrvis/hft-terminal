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

// Calculate metrics from trades data
const calculateMetrics = (trades) => {
  if (!trades || trades.length === 0) {
    return {
      totalPnl: 0,
      maxDrawdown: 0,
      totalTrades: 0,
      winRate: 0,
      winCount: 0,
      profitFactor: 0,
      profitTargetCount: 0,
      trailingStopCount: 0,
      stopLossCount: 0,
      signalCount: 0,
    };
  }

  let totalPnl = 0;
  let grossProfit = 0;
  let grossLoss = 0;
  let winCount = 0;
  let peakProfit = 0;
  let maxDrawdown = 0;
  let profitTargetCount = 0;
  let trailingStopCount = 0;
  let stopLossCount = 0;
  let signalCount = 0;

  trades.forEach((trade) => {
    const profit = trade.profit || 0;
    totalPnl += profit;

    if (profit > 0) {
      winCount++;
      grossProfit += profit;
    } else if (profit < 0) {
      grossLoss += Math.abs(profit);
    }

    // Track drawdown
    if (totalPnl > peakProfit) peakProfit = totalPnl;
    const drawdown = peakProfit - totalPnl;
    if (drawdown > maxDrawdown) maxDrawdown = drawdown;

    // Count exit reasons
    switch (trade.reason) {
      case "PROFIT_TARGET": profitTargetCount++; break;
      case "TRAILING_STOP": trailingStopCount++; break;
      case "STOP_LOSS": stopLossCount++; break;
      case "SIGNAL": signalCount++; break;
    }
  });

  const totalTrades = trades.length;
  const winRate = totalTrades > 0 ? (winCount / totalTrades) * 100 : 0;
  const profitFactor = grossLoss > 0 ? grossProfit / grossLoss : grossProfit > 0 ? Infinity : 0;

  return {
    totalPnl,
    maxDrawdown,
    totalTrades,
    winRate,
    winCount,
    profitFactor,
    profitTargetCount,
    trailingStopCount,
    stopLossCount,
    signalCount,
  };
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
  const [trades, setTrades] = useState([]);
  const [tradesFilter, setTradesFilter] = useState("");
  const [tradesSort, setTradesSort] = useState("entryTime-desc");

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

  const fetchTrades = async () => {
    const res = await fetch(`http://localhost:5001/trades`);
    if (!res.ok) {
      throw new Error(`HTTP ${res.status}`);
    }
    const data = await res.json();
    setTrades(data);
  };

  useEffect(() => {
    if (!logsOpen) return undefined;
    fetchLogs();
    const id = setInterval(fetchLogs, 4000);
    return () => clearInterval(id);
  }, [logsOpen]);

  useEffect(() => {
    fetchTrades();
  }, []);

  useEffect(() => {
    document.body.classList.remove("theme-light", "theme-dark");
    document.body.classList.add(theme === "dark" ? "theme-dark" : "theme-light");
  }, [theme]);

  const filteredSortedTrades = (() => {
    const term = tradesFilter.trim().toLowerCase();
    const filtered = term
      ? trades.filter((t) =>
          [t?.entryTime, t?.exitTime, t?.type, t?.entryPrice, t?.exitPrice, t?.profit, t?.reason]
            .map((v) => (v === undefined || v === null ? "" : String(v).toLowerCase()))
            .some((v) => v.includes(term))
        )
      : trades;

    const [field, dir] = (tradesSort || "entryTime-desc").split("-");
    const sorted = [...filtered].sort((a, b) => {
      const mult = dir === "asc" ? 1 : -1;
      switch (field) {
        case "entryPrice":
          return mult * ((Number(a?.entryPrice) || 0) - (Number(b?.entryPrice) || 0));
        case "exitPrice":
          return mult * ((Number(a?.exitPrice) || 0) - (Number(b?.exitPrice) || 0));
        case "profit":
          return mult * ((Number(a?.profit) || 0) - (Number(b?.profit) || 0));
        case "type":
          return mult * String(a?.type || "").localeCompare(String(b?.type || ""));
        case "exitTime": {
          const ta = new Date(a?.exitTime || "").getTime() || 0;
          const tb = new Date(b?.exitTime || "").getTime() || 0;
          return mult * (ta - tb);
        }
        case "entryTime":
        default: {
          const ta = new Date(a?.entryTime || "").getTime() || 0;
          const tb = new Date(b?.entryTime || "").getTime() || 0;
          return mult * (ta - tb);
        }
      }
    });

    return sorted;
  })();

  const tradeMetrics = calculateMetrics(trades);

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
          metrics={tradeMetrics}
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
