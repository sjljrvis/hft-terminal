import { useEffect, useState } from "react";
import axios from "axios";
import ChartPanel from "./ChartPanel";
import { useAppDispatch, useAppSelector } from "../store/hooks";
import { fetchLiveTrades, selectLiveTotalPnL } from "../store/slices/tradesSlice";

function LivePanel() {
  const [status, setStatus] = useState(null);
  const [error, setError] = useState("");
  const dispatch = useAppDispatch();
  const positionPnL = useAppSelector(selectLiveTotalPnL);

  // Fetch live HFT status
  useEffect(() => {
    const controller = new AbortController();
    const fetchStatus = async () => {
      try {
        const { data } = await axios.get("http://localhost:5001/hft/status", { signal: controller.signal });
        setStatus(data);
        setError("");
      } catch (err) {
        if (controller.signal.aborted) return;
        setError("Unable to load live status");
        console.error(err);
      }
    };

    fetchStatus();
    const interval = setInterval(fetchStatus, 5000); // Poll every 5 seconds

    return () => {
      controller.abort();
      clearInterval(interval);
    };
  }, []);

  // Fetch live trades once on mount and store in Redux
  useEffect(() => {
    dispatch(fetchLiveTrades());
  }, [dispatch]);

  const isConnected = status?.hft?.status === "connected";
  const formatPnL = (pnl) => {
    if (pnl === null || pnl === undefined) return "No Position";
    const sign = pnl >= 0 ? "+" : "";
    const color = pnl >= 0 ? "var(--green-candle)" : "var(--red-candle)";
    return { text: `${sign}${pnl.toFixed(2)}`, color };
  };

  const pnlDisplay = formatPnL(positionPnL);

  return (
    <div className="live-panel">
      <div className="live-toolbar">
        <p> LIVE </p>
        <div className="live-toolbar-pnl">
          <span className="live-toolbar-pnl-label">P&L:</span>
          <span 
            className="live-toolbar-pnl-value" 
            style={{ color: pnlDisplay.color }}
          >
            {pnlDisplay.text}
          </span>
        </div>
        <div className="live-toolbar-buttons">
          {!isConnected && (
            <button className="live-toolbar-button">
              <span> Connect Broker </span>
            </button>
          )}
          <button className="live-toolbar-button">
            <span> Close All Trades </span>
          </button>
          <button className="live-toolbar-button live-toolbar-button--kill">
            <span> Kill Switch </span>
          </button>
        </div>
      </div>
      <ChartPanel apiEndpoint="http://localhost:5001/live/ticks" />
    </div>
  );
}

export default LivePanel;
