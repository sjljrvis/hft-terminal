import { useEffect } from "react";
import { BrowserRouter, Routes, Route, Navigate, useLocation } from "react-router-dom";
import "./style.css";
import Header from "./components/Header";
import StatusBar from "./components/StatusBar";
import DashboardCard from "./components/DashboardCard";
import Sidebar from "./components/Sidebar";
import LogDrawer from "./components/LogDrawer";
import ChartPanel from "./components/ChartPanel";
import BacktestPanel from "./components/BacktestPanel";
import LivePanel from "./components/LivePanel";
import TradesDrawer from "./components/TradesDrawer";
import DBQueryPanel from "./components/DBQueryPanel";
import SettingsPanel from "./components/SettingsPanel";
import { useAppDispatch, useAppSelector } from "./store/hooks";
import {
  selectTheme,
  selectSidebarCollapsed,
  selectTradesDrawerOpen,
  selectLogsDrawerOpen,
  selectLogSize,
  setTheme,
  toggleSidebar,
  toggleLogsDrawer,
  closeLogsDrawer,
  setActivePanel,
} from "./store/slices/uiSlice";

function AppContent() {
  const location = useLocation();
  const dispatch = useAppDispatch();

  // UI state from Redux
  const theme = useAppSelector(selectTheme);
  const sidebarCollapsed = useAppSelector(selectSidebarCollapsed);
  const logsOpen = useAppSelector(selectLogsDrawerOpen);

  // Set active panel based on route
  useEffect(() => {
    if (location.pathname === '/live') {
      dispatch(setActivePanel('live'));
    } else if (location.pathname === '/backtest') {
      dispatch(setActivePanel('backtest'));
    }
  }, [location.pathname, dispatch]);

  // Fetch logs when drawer opens


  // Apply theme to body
  useEffect(() => {
    document.body.classList.remove("theme-light", "theme-dark");
    document.body.classList.add(theme === "dark" ? "theme-dark" : "theme-light");
  }, [theme]);

  return (
    <div className="app-shell">
      <Header theme={theme} onThemeChange={(newTheme) => dispatch(setTheme(newTheme))} />
      <div className="app-body">
        <Sidebar collapsed={sidebarCollapsed} />
        <main className="app-main">
          <Routes>
            <Route path="/" element={<Navigate to="/live" replace />} />
            <Route path="/live" element={<LivePanel />} />
            <Route path="/backtest" element={<BacktestPanel />} />
            <Route path="/query" element={<DBQueryPanel />} />
            <Route path="/settings" element={<SettingsPanel />} />
            <Route path="*" element={<DashboardCard />} />
          </Routes>
        </main>
      </div>
      <TradesDrawer />
      <StatusBar
        sidebarCollapsed={sidebarCollapsed}
        onToggleSidebar={() => dispatch(toggleSidebar())}
        logsOpen={logsOpen}
        onToggleLogs={() => dispatch(toggleLogsDrawer())}
      />
      <LogDrawer />
    </div>
  );
}

function App() {
  return (
    <BrowserRouter>
      <AppContent />
    </BrowserRouter>
  );
}

export default App;
