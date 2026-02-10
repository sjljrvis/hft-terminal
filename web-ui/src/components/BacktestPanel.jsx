import { useEffect, useState, useRef, useMemo } from "react";
import axios from "axios";
import ChartPanel from "./ChartPanel";
import { useAppDispatch } from "../store/hooks";
import { fetchTrades } from "../store/slices/tradesSlice";
import { Calendar } from "phosphor-react";

function BacktestPanel() {
  const dispatch = useAppDispatch();

  // Set default dates: past year to present
  const getDefaultDates = () => {
    const today = new Date();
    const oneYearAgo = new Date();
    oneYearAgo.setFullYear(today.getFullYear() - 1);

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
        console.error("Backtest error:", err);
        setError(err.response?.data || err.message || "Failed to run backtest");
      } finally {
        setIsRunning(false);
      }
    };

    runInitialBacktest();
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

          <button className="backtest-toolbar-button">
            <span> Run </span>
          </button>
          <button className="backtest-toolbar-button">
            <span> Reset </span>
          </button>
        </div>
      </div>
      <ChartPanel apiEndpoint={backtestDataEndpoint} />
    </div>
  );
}

export default BacktestPanel;
