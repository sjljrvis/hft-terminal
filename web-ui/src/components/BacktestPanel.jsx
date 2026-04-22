import { useEffect, useState, useRef, useMemo, useCallback } from "react";
import axios from "axios";
import ChartPanel from "./ChartPanel";
import { useAppDispatch } from "../store/hooks";
import { fetchTrades } from "../store/slices/tradesSlice";
import { Calendar, Play, Stop, Lightning } from "phosphor-react";

/* ───────────────── Simulation sub-component ───────────────── */

function SimulatePanel() {
  const [simDate, setSimDate] = useState(() => {
    const d = new Date();
    d.setDate(d.getDate() - 1);
    return d.toISOString().slice(0, 10);
  });
  const [tickDelay, setTickDelay] = useState(1);
  const [warmupDays, setWarmupDays] = useState(100);
  const [isRunning, setIsRunning] = useState(false);
  const [error, setError] = useState("");

  // Sim state from WS events
  const [simStatus, setSimStatus] = useState(null); // "loading" | "replaying" | "done" | "error"
  const [currentTick, setCurrentTick] = useState(null);
  const [trades, setTrades] = useState([]);
  const [summary, setSummary] = useState(null);
  const [events, setEvents] = useState([]); // activity feed

  const wsRef = useRef(null);
  const feedEndRef = useRef(null);

  // Auto-scroll feed
  useEffect(() => {
    feedEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [events]);

  // Connect to WebSocket when simulation starts
  const connectWS = useCallback(() => {
    if (wsRef.current) {
      wsRef.current.close();
    }
    const ws = new WebSocket("ws://localhost:5001/ws/events");
    wsRef.current = ws;

    ws.onmessage = (evt) => {
      // writePump coalesces queued messages into one frame separated by \n.
      // Split and parse each independently so no events are lost.
      const parts = evt.data.split('\n');
      for (const part of parts) {
        try {
          const msg = JSON.parse(part);
          const { type, data } = msg;

          if (type === "sim_start") {
            setSimStatus(data.status);
            addEvent("info", data.status === "loading" ? `Loading ${data.simDate}...` : `Replaying ${data.simBars} bars`);
          } else if (type === "sim_tick") {
            setCurrentTick(data);
          } else if (type === "sim_entry") {
            setTrades((prev) => [...prev, { type: "entry", ...data }]);
            addEvent("entry", `${data.side} @ ${data.price} (${data.tranche})`);
          } else if (type === "sim_exit") {
            setTrades((prev) => [...prev, { type: "exit", ...data }]);
            const pnlClass = data.pnl >= 0 ? "positive" : "negative";
            addEvent("exit", `${data.side} exit (${data.reason}) PnL: ${data.pnl > 0 ? "+" : ""}${data.pnl}`, pnlClass);
          } else if (type === "sim_position") {
            addEvent("info", `${data.event} @ ${data.unrealPct?.toFixed(3)}%`);
          } else if (type === "sim_end") {
            setSimStatus("done");
            setSummary(data);
            setIsRunning(false);
            addEvent("info", `Done: ${data.tradeCount} trades, ${data.winRate}% WR, net ${data.netPnl} pts`);
          } else if (type === "sim_error") {
            setSimStatus("error");
            setError(data.error || "Simulation error");
            setIsRunning(false);
            addEvent("error", data.error || "Simulation error");
          }
        } catch {}
      }
    };

    ws.onerror = () => {
      setError("WebSocket connection error");
    };

    ws.onclose = () => {
      // Natural close on sim end
    };
  }, []);

  const addEvent = useCallback((type, text, pnlClass) => {
    const ts = new Date().toLocaleTimeString("en-GB", {
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    });
    setEvents((prev) => [...prev.slice(-200), { ts, type, text, pnlClass }]);
  }, []);

  const handleStart = async () => {
    if (!simDate) {
      setError("Select a date");
      return;
    }
    setError("");
    setIsRunning(true);
    setSimStatus("loading");
    setCurrentTick(null);
    setTrades([]);
    setSummary(null);
    setEvents([]);

    // Connect WS first
    connectWS();

    // Small delay to let WS connect
    await new Promise((r) => setTimeout(r, 300));

    try {
      await axios.post("http://localhost:5001/simulate", {
        date: simDate,
        warmupDays,
        tickDelay,
      });
    } catch (err) {
      setError(err.response?.data || err.message || "Failed to start simulation");
      setIsRunning(false);
      setSimStatus("error");
    }
  };

  const handleStop = () => {
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
    setIsRunning(false);
    setSimStatus(null);
  };

  // Cleanup WS on unmount
  useEffect(() => {
    return () => {
      if (wsRef.current) {
        wsRef.current.close();
      }
    };
  }, []);

  const progress = currentTick
    ? Math.round((currentTick.barIndex / currentTick.totalBars) * 100)
    : 0;

  return (
    <div className="sim-panel">
      {/* Controls */}
      <div className="sim-controls">
        <div className="sim-controls__row">
          <div className="sim-input-group">
            <label>Date</label>
            <input
              type="date"
              className="sim-input"
              value={simDate}
              onChange={(e) => setSimDate(e.target.value)}
              disabled={isRunning}
            />
          </div>
          <div className="sim-input-group">
            <label>Delay (s)</label>
            <input
              type="number"
              className="sim-input sim-input--narrow"
              value={tickDelay}
              onChange={(e) => setTickDelay(Math.max(1, parseInt(e.target.value) || 1))}
              min={1}
              max={60}
              disabled={isRunning}
            />
          </div>
          <div className="sim-input-group">
            <label>Warmup</label>
            <input
              type="number"
              className="sim-input sim-input--narrow"
              value={warmupDays}
              onChange={(e) => setWarmupDays(Math.max(10, parseInt(e.target.value) || 100))}
              min={10}
              max={500}
              disabled={isRunning}
            />
          </div>
          {!isRunning ? (
            <button className="sim-btn sim-btn--start" onClick={handleStart}>
              <Play size={12} weight="fill" />
              <span>Simulate</span>
            </button>
          ) : (
            <button className="sim-btn sim-btn--stop" onClick={handleStop}>
              <Stop size={12} weight="fill" />
              <span>Stop</span>
            </button>
          )}
        </div>
        {error && <div className="sim-error">{error}</div>}
      </div>

      {/* Content only shows when sim is active or has results */}
      {(simStatus || summary) && (
        <div className="sim-body">
          {/* Progress + current tick info */}
          {currentTick && (
            <div className="sim-ticker">
              <div className="sim-progress-bar">
                <div className="sim-progress-fill" style={{ width: `${progress}%` }} />
              </div>
              <div className="sim-ticker__row">
                <span className="sim-ticker__item">
                  <span className="sim-ticker__label">Time</span>
                  <span className="sim-ticker__value">{currentTick.time}</span>
                </span>
                <span className="sim-ticker__item">
                  <span className="sim-ticker__label">Close</span>
                  <span className="sim-ticker__value">{currentTick.close?.toFixed(2)}</span>
                </span>
                <span className="sim-ticker__item">
                  <span className="sim-ticker__label">Regime</span>
                  <span className={`sim-ticker__value sim-regime--${currentTick.regime?.toLowerCase()}`}>
                    {currentTick.regime || "—"}
                  </span>
                </span>
                <span className="sim-ticker__item">
                  <span className="sim-ticker__label">Bull</span>
                  <span className="sim-ticker__value">{currentTick.probBull?.toFixed(3)}</span>
                </span>
                <span className="sim-ticker__item">
                  <span className="sim-ticker__label">Bear</span>
                  <span className="sim-ticker__value">{currentTick.probBear?.toFixed(3)}</span>
                </span>
                <span className="sim-ticker__item">
                  <span className="sim-ticker__label">Tranche</span>
                  <span className="sim-ticker__value">{currentTick.tranche}</span>
                </span>
                <span className="sim-ticker__item">
                  <span className="sim-ticker__label">Position</span>
                  <span className={`sim-ticker__value ${currentTick.position === "LONG" ? "is-up" : currentTick.position === "SHORT" ? "is-down" : ""}`}>
                    {currentTick.position}
                  </span>
                </span>
                {currentTick.unrealPnl !== undefined && (
                  <span className="sim-ticker__item">
                    <span className="sim-ticker__label">Unreal</span>
                    <span className={`sim-ticker__value ${currentTick.unrealPnl >= 0 ? "is-up" : "is-down"}`}>
                      {currentTick.unrealPnl > 0 ? "+" : ""}{currentTick.unrealPnl?.toFixed(2)}
                    </span>
                  </span>
                )}
                <span className="sim-ticker__item">
                  <span className="sim-ticker__label">Net P&L</span>
                  <span className={`sim-ticker__value ${currentTick.netPnl >= 0 ? "is-up" : "is-down"}`}>
                    {currentTick.netPnl > 0 ? "+" : ""}{currentTick.netPnl?.toFixed(2)}
                  </span>
                </span>
                <span className="sim-ticker__item sim-ticker__item--muted">
                  {currentTick.barIndex + 1}/{currentTick.totalBars}
                </span>
              </div>
            </div>
          )}

          {/* Summary card (after sim finishes) */}
          {summary && (
            <div className="sim-summary">
              <div className="sim-summary__item">
                <span className="sim-summary__label">Trades</span>
                <span className="sim-summary__value">{summary.tradeCount}</span>
              </div>
              <div className="sim-summary__item">
                <span className="sim-summary__label">Wins</span>
                <span className="sim-summary__value is-up">{summary.winCount}</span>
              </div>
              <div className="sim-summary__item">
                <span className="sim-summary__label">Win Rate</span>
                <span className="sim-summary__value">{summary.winRate}%</span>
              </div>
              <div className="sim-summary__item">
                <span className="sim-summary__label">Net P&L</span>
                <span className={`sim-summary__value ${summary.netPnl >= 0 ? "is-up" : "is-down"}`}>
                  {summary.netPnl > 0 ? "+" : ""}{summary.netPnl} pts
                </span>
              </div>
            </div>
          )}

          {/* Activity feed */}
          <div className="sim-feed">
            <div className="sim-feed__header">
              <span>Activity</span>
              <span className="sim-feed__count">{trades.length} trades</span>
            </div>
            <div className="sim-feed__body">
              {events.map((e, i) => (
                <div key={i} className={`sim-feed__item sim-feed__item--${e.type}`}>
                  <span className="sim-feed__ts">{e.ts}</span>
                  <span className="sim-feed__type">
                    {e.type === "entry" ? "ENTRY" : e.type === "exit" ? "EXIT" : e.type === "error" ? "ERR" : "INFO"}
                  </span>
                  <span className={`sim-feed__text ${e.pnlClass ? `sim-feed__text--${e.pnlClass}` : ""}`}>
                    {e.text}
                  </span>
                </div>
              ))}
              <div ref={feedEndRef} />
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

/* ───────────────── Main BacktestPanel ───────────────── */

function BacktestPanel() {
  const dispatch = useAppDispatch();

  const [showSimulate, setShowSimulate] = useState(false);

  // Set default dates: past 6 months to present
  const getDefaultDates = () => {
    const today = new Date();
    const oneYearAgo = new Date();
    oneYearAgo.setMonth(today.getMonth() - 6);

    const formatDate = (date) => {
      const year = date.getFullYear();
      const month = String(date.getMonth() + 1).padStart(2, '0');
      const day = String(date.getDate()).padStart(2, '0');
      return `${year}-${month}-${day}`;
    };

    return {
      start: formatDate(oneYearAgo),
      end: formatDate(today)
    };
  };

  const defaultDates = getDefaultDates();
  const [startDate, setStartDate] = useState(defaultDates.start);
  const [endDate, setEndDate] = useState(defaultDates.end);
  const [showCalendar, setShowCalendar] = useState(false);
  const [isRunning, setIsRunning] = useState(false);
  const [error, setError] = useState("");
  const [backtestKey, setBacktestKey] = useState(0); // Key to force ChartPanel refetch
  const calendarRef = useRef(null);
  const hasRunInitialBacktest = useRef(false);

  // Build the backtest data API endpoint (returns processed data with indicators)
  // Add a timestamp query param to force refetch when backtest completes
  const backtestDataEndpoint = useMemo(() => {
    return `http://localhost:5001/backtest/data?t=${backtestKey}`;
  }, [backtestKey]);

  // Run backtest automatically on page load with default dates
  useEffect(() => {
    if (hasRunInitialBacktest.current) return;
    if (!startDate || !endDate) return;

    hasRunInitialBacktest.current = true;

    const runInitialBacktest = async () => {
      setIsRunning(true);
      setError("");

      try {
        // Run backtest with default dates
        const response = await axios.post("http://localhost:5001/backtest/run", {
          startDate,
          endDate
        });

        if (response.data.status === "success") {
          // Wait for backtest to complete processing (increased wait time)
          await new Promise(resolve => setTimeout(resolve, 1000));

          // Fetch trades after backtest completes
          await dispatch(fetchTrades());

          // Trigger ChartPanel to refetch data
          setBacktestKey(prev => prev + 1);
        } else {
          setError("Backtest completed but returned unexpected status");
        }
      } catch (err) {
        // 409 means server startup already triggered a run — not an error
        if (err.response?.status === 409) {
          console.log("Backtest already running on server, skipping duplicate run");
        } else {
          console.error("Backtest error:", err);
          setError(err.response?.data || err.message || "Failed to run backtest");
        }
      } finally {
        setIsRunning(false);
      }
    };

    // runInitialBacktest();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []); // Run only once on mount

  // Fetch backtest trades when panel mounts
  useEffect(() => {
    dispatch(fetchTrades());
  }, [dispatch]);

  // Close calendar when clicking outside
  useEffect(() => {
    const handleClickOutside = (event) => {
      if (calendarRef.current && !calendarRef.current.contains(event.target)) {
        setShowCalendar(false);
      }
    };

    if (showCalendar) {
      document.addEventListener("mousedown", handleClickOutside);
      return () => {
        document.removeEventListener("mousedown", handleClickOutside);
      };
    }
  }, [showCalendar]);

  const handleRunBacktest = async () => {
    if (!startDate || !endDate) {
      setError("Please select both start and end dates");
      return;
    }

    setIsRunning(true);
    setError("");

    try {
      // Run backtest with selected dates
      const response = await axios.post("http://localhost:5001/backtest/run", {
        startDate,
        endDate
      });

      if (response.data.status === "success") {
        // Wait for backtest to complete processing (increased wait time)
        await new Promise(resolve => setTimeout(resolve, 1000));

        // Fetch trades after backtest completes
        await dispatch(fetchTrades());

        // Trigger ChartPanel to refetch data
        setBacktestKey(prev => prev + 1);

        setShowCalendar(false);
      } else {
        setError("Backtest completed but returned unexpected status");
      }
    } catch (err) {
      console.error("Backtest error:", err);
      setError(err.response?.data || err.message || "Failed to run backtest");
    } finally {
      setIsRunning(false);
    }
  };

  return (
    <div className="backtest-panel">
      <div className="backtest-toolbar">
        <p> BACKTEST </p>

        <div className="backtest-toolbar-buttons">
          <div className="backtest-toolbar-calendar" ref={calendarRef}>
            <button
              className="backtest-toolbar-calendar-toggle"
              onClick={() => setShowCalendar(!showCalendar)}
            >
              <Calendar size={14} weight="regular" />
              <span>Date Range</span>
            </button>
            {showCalendar && (
              <div className="backtest-toolbar-calendar-picker">
                <div className="backtest-calendar-input-group">
                  <label>Start:</label>
                  <input
                    type="date"
                    className="backtest-calendar-input"
                    value={startDate}
                    onChange={(e) => setStartDate(e.target.value)}
                  />
                </div>
                <div className="backtest-calendar-input-group">
                  <label>End:</label>
                  <input
                    type="date"
                    className="backtest-calendar-input"
                    value={endDate}
                    onChange={(e) => setEndDate(e.target.value)}
                    min={startDate}
                  />
                </div>
                <button
                  className="backtest-toolbar-button backtest-run-button"
                  onClick={handleRunBacktest}
                  disabled={!startDate || !endDate || isRunning}
                >
                  <span>{isRunning ? "Running..." : "Run"}</span>
                </button>
                {error && (
                  <div style={{
                    color: "var(--red-candle)",
                    fontSize: "10px",
                    marginTop: "4px",
                    textAlign: "center"
                  }}>
                    {error}
                  </div>
                )}
              </div>
            )}
          </div>

          <button
            className={`backtest-toolbar-button ${showSimulate ? "backtest-toolbar-button--active" : ""}`}
            onClick={() => setShowSimulate(!showSimulate)}
          >
            <Lightning size={12} weight="fill" />
            <span style={{ marginLeft: 4 }}>Simulate</span>
          </button>

          <button className="backtest-toolbar-button">
            <span> Run </span>
          </button>
          <button className="backtest-toolbar-button">
            <span> Reset </span>
          </button>
        </div>
      </div>

      {showSimulate && <SimulatePanel />}

      {!showSimulate && <ChartPanel apiEndpoint={backtestDataEndpoint} />}
    </div>
  );
}

export default BacktestPanel;
