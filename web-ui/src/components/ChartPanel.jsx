import { useCallback, useEffect, useRef, useState } from "react";
import axios from "axios";
import { createChart, CandlestickSeries, LineSeries } from "lightweight-charts";
import { SlidersHorizontal, ChartLine, TestTube, GearSix, Table, Ruler, X, XCircle } from "phosphor-react";

import { useAppDispatch } from "../store/hooks";
import { setTradesDrawerOpen } from "../store/slices/uiSlice";

function ChartPanel({ apiEndpoint = "http://localhost:5001/ticks" }) {
  const dispatch = useAppDispatch();
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [ticks, setTicks] = useState([]);
  const [hoverValues, setHoverValues] = useState(null);
  const [measureMode, setMeasureMode] = useState(false);
  const [measurePoints, setMeasurePoints] = useState({ start: null, end: null });
  const [measureHover, setMeasureHover] = useState(null); // Track hover position during measurement
  const measureModeRef = useRef(false);
  const measurePointsRef = useRef({ start: null, end: null });
  const chartContainerRef = useRef(null);
  const resizeObserverRef = useRef(null);
  const chartRef = useRef(null);
  const candleSeriesRef = useRef(null);
  const lineSeriesSlowRef = useRef(null);
  const lineSeriesFastRef = useRef(null);
  const measureLineSeriesRef = useRef(null);
  const pendingDateKeyRef = useRef(null);

  const toDateKey = useCallback((value) => {
    const parseValue = (raw) => {
      if (raw instanceof Date) {
        const ms = raw.getTime();
        return Number.isFinite(ms) ? new Date(ms).toISOString().slice(0, 10) : null;
      }
      if (typeof raw === "number") {
        const ms = raw < 1e12 ? raw * 1000 : raw;
        return Number.isFinite(ms) ? new Date(ms).toISOString().slice(0, 10) : null;
      }
      if (typeof raw === "string") {
        const trimmed = raw.trim();
        if (!trimmed) return null;
        if (/^\d{4}-\d{2}-\d{2}$/.test(trimmed)) return trimmed;
        const parsed = new Date(trimmed);
        return Number.isNaN(parsed.getTime()) ? null : parsed.toISOString().slice(0, 10);
      }
      return null;
    };

    if (value && typeof value === "object" && !(value instanceof Date)) {
      const candidate = value.date ?? value.value ?? value.timestamp ?? value.time;
      if (candidate !== undefined) return parseValue(candidate);
    }
    return parseValue(value);
  }, []);

  const getTickDateKey = useCallback((timeSeconds) => {
    if (!Number.isFinite(timeSeconds)) return null;
    return new Date(timeSeconds * 1000).toISOString().slice(0, 10);
  }, []);

  const focusDateKey = useCallback(
    (dateKey) => {
      if (!dateKey || !chartRef.current || !ticks.length) return false;

      let from = null;
      let to = null;
      let prevTs = null;
      let nextTs = null;

      for (const tick of ticks) {
        const ts = tick?.timeIst ?? tick?.time;
        if (!Number.isFinite(ts)) continue;
        const key = getTickDateKey(ts);
        if (!key) continue;

        if (key === dateKey) {
          if (from === null) from = ts;
          to = ts;
          continue;
        }

        if (from !== null) break;
        if (key < dateKey) prevTs = ts;
        if (key > dateKey) {
          nextTs = ts;
          break;
        }
      }

      if (from !== null && to !== null) {
        const span = Math.max(to - from, 60 * 5);
        const center = from + (to - from) / 2;
        chartRef.current.timeScale().setVisibleRange({
          from: center - span / 2,
          to: center + span / 2,
        });
        return true;
      }

      const target = nextTs ?? prevTs;
      if (Number.isFinite(target)) {
        const padding = 60 * 30;
        chartRef.current.timeScale().setVisibleRange({
          from: target - padding,
          to: target + padding,
        });
        return true;
      }

      return false;
    },
    [getTickDateKey, ticks]
  );

  const requestDateFocus = useCallback(
    (payload) => {
      const dateKey = toDateKey(payload);
      if (!dateKey) return;
      pendingDateKeyRef.current = dateKey;
      if (focusDateKey(dateKey)) {
        pendingDateKeyRef.current = null;
      }
    },
    [focusDateKey, toDateKey]
  );

  useEffect(() => {
    if (typeof window === "undefined") return undefined;
    const handleDateChange = (event) => {
      const detail = event?.detail ?? event;
      requestDateFocus(detail);
    };
    window.addEventListener("onDateChange", handleDateChange);
    return () => window.removeEventListener("onDateChange", handleDateChange);
  }, [requestDateFocus]);

  const measureDimensions = useCallback(() => {
    // Fixed-height; width follows available chart area
    const container = chartContainerRef.current;
    const parent = container?.parentElement;
    const actualWidth =
      parent?.getBoundingClientRect?.().width ||
      parent?.clientWidth ||
      container?.clientWidth ||
      0;

    const actualHeight =
      parent?.getBoundingClientRect?.().height ||
      parent?.clientHeight ||
      container?.clientHeight ||
      0;
    // Guard against momentary zero widths during layout/transition; keep a sensible min
    const width = Math.max(1380, Math.round(actualWidth));
    const height = Math.max(690, Math.round(actualHeight));
    return { width, height };
  }, []);

  const resizeToContainer = useCallback(() => {
    if (!chartRef.current) return;
    // Defer to the next frame so flex/layout updates settle before measuring
    requestAnimationFrame(() => {
      if (chartRef.current) {
        chartRef.current.applyOptions(measureDimensions());
      }
    });
  }, [measureDimensions]);

  const IST_OFFSET_SECONDS = 5.5 * 60 * 60; // +05:30

  const normalizeTick = useCallback((tick) => {
    let ts = tick?.time;
    if (typeof ts === "string") {
      const parsed = Date.parse(ts);
      ts = Number.isNaN(parsed) ? undefined : Math.floor(parsed / 1000);
    }
    if (typeof ts !== "number") {
      const parsed = Date.parse(tick?.timestamp);
      ts = Number.isNaN(parsed) ? undefined : Math.floor(parsed / 1000);
    }
    const tsIst = typeof ts === "number" ? ts + IST_OFFSET_SECONDS : ts;

    return {
      time: ts,
      open: Number(tick?.open) || 0,
      high: Number(tick?.high) || 0,
      low: Number(tick?.low) || 0,
      close: Number(tick?.close) || 0,
      fast_tempx: Number.isFinite(Number(tick?.fast_tempx)) ? Number(tick?.fast_tempx) : undefined,
      slow_tempx: Number.isFinite(Number(tick?.slow_tempx)) ? Number(tick?.slow_tempx) : undefined,

      swap: Number.isFinite(Number(tick?.swap)) ? Number(tick?.swap) : undefined,
      swap_base: Number.isFinite(Number(tick?.swap_base)) ? Number(tick?.swap_base) : undefined,
      timeIst: tsIst,
    };
  }, []);

  const swapColor = useCallback((swapVal) => {
    if (swapVal === 1) return "#00e6a8"; // cyan-green
    if (swapVal === -1) return "#ff4da6"; // magenta-pink
    return "#facc15"; // yellow for 0 / default
  }, []);

  const formatValue = useCallback((value) => {
    if (!Number.isFinite(value)) return "--";
    return String(value);
  }, []);

  const formatMeasureValue = useCallback((value, decimals = 2) => {
    if (!Number.isFinite(value)) return "--";
    return value.toFixed(decimals);
  }, []);

  const findTickAtTime = useCallback((timeSeconds) => {
    return ticks.find((t) => {
      const tTime = t.timeIst ?? t.time;
      return Math.abs(tTime - timeSeconds) < 60; // Within 1 minute
    });
  }, [ticks]);

  const calculateMeasureStats = useCallback((startPoint, endPoint) => {
    if (!startPoint || !endPoint || !startPoint.time || !endPoint.time) return null;

    const startTick = findTickAtTime(startPoint.time);
    const endTick = findTickAtTime(endPoint.time);
    
    if (!startTick || !endTick) return null;

    const startPrice = startPoint.price;
    const endPrice = endPoint.price;
    const priceDiff = endPrice - startPrice;
    const priceDiffPercent = startPrice !== 0 ? (priceDiff / startPrice) * 100 : 0;

    // Count bars between the two points
    const startTime = startTick.timeIst ?? startTick.time;
    const endTime = endTick.timeIst ?? endTick.time;
    const timeDiff = Math.abs(endTime - startTime);
    
    let barCount = 0;
    const minTime = Math.min(startTime, endTime);
    const maxTime = Math.max(startTime, endTime);
    for (const tick of ticks) {
      const tTime = tick.timeIst ?? tick.time;
      if (tTime >= minTime && tTime <= maxTime) {
        barCount++;
      }
    }
    barCount = Math.max(0, barCount - 1); // Subtract 1 to exclude the start point

    // Format time difference
    const hours = Math.floor(timeDiff / 3600);
    const minutes = Math.floor((timeDiff % 3600) / 60);
    const timeStr = hours > 0 ? `${hours}h ${minutes}m` : `${minutes}m`;

    return {
      priceDiff,
      priceDiffPercent,
      barCount,
      timeDiff: timeStr,
      startPrice,
      endPrice,
    };
  }, [findTickAtTime, ticks]);

  const clearMeasurement = useCallback(() => {
    setMeasurePoints({ start: null, end: null });
    setMeasureHover(null);
    measurePointsRef.current = { start: null, end: null };
    if (measureLineSeriesRef.current) {
      measureLineSeriesRef.current.setData([]);
    }
  }, []);

  const toggleMeasureMode = useCallback(() => {
    const newMode = !measureModeRef.current;
    measureModeRef.current = newMode;
    setMeasureMode(newMode);
    if (!newMode) {
      clearMeasurement();
    }
  }, [clearMeasurement]);


  useEffect(() => {
    const controller = new AbortController();
    const fetchTicks = async () => {
      setLoading(true);
      try {
        const { data } = await axios.get(apiEndpoint, { signal: controller.signal });
        const normalized = Array.isArray(data) ? data.map((d) => normalizeTick(d)) : [];
        
        // Filter out ticks with invalid time values
        const validTicks = normalized.filter((t) => Number.isFinite(t.time) && t.time > 0);
        
        // Sort by time (ascending)
        validTicks.sort((a, b) => a.time - b.time);
        
        // Deduplicate by time (keep the first occurrence of each timestamp)
        const deduplicated = [];
        const seenTimes = new Set();
        for (const tick of validTicks) {
          if (!seenTimes.has(tick.time)) {
            seenTimes.add(tick.time);
            deduplicated.push(tick);
          }
        }
        
        setTicks(deduplicated);
        setError("");
      } catch (err) {
        if (controller.signal.aborted) return;
        setError("Unable to load ticks");
        console.error(err);
      } finally {
        setLoading(false);
      }
    };
    fetchTicks();
    return () => controller.abort();
  }, [normalizeTick, apiEndpoint]);

  const withAlpha = useCallback((color, alpha) => {
    if (!color) return color;
    const trimmed = color.trim();
    if (trimmed.startsWith("#")) {
      const hex = trimmed.replace("#", "");
      const normalize = hex.length === 3 ? hex.split("").map((c) => c + c).join("") : hex;
      const intVal = parseInt(normalize, 16);
      const r = (intVal >> 16) & 255;
      const g = (intVal >> 8) & 255;
      const b = intVal & 255;
      return `rgba(${r}, ${g}, ${b}, ${alpha})`;
    }
    if (trimmed.startsWith("rgb(")) {
      return trimmed.replace("rgb(", "rgba(").replace(")", `, ${alpha})`);
    }
    return trimmed;
  }, []);

  const getThemeColors = useCallback(() => {
    if (typeof window === "undefined") {
      return {
        background: "#0b1221",
        text: "#6b7280",
        grid: "rgba(255,255,255,0.05)",
        accent: "#22c55e",
        redCandle: "#ef4444",
        greenCandle: "#22c55e",
      };
    }
    const styles = getComputedStyle(document.body);
    const pick = (varName, fallback) => styles.getPropertyValue(varName)?.trim() || fallback;
    return {
      background: pick("--bg-color", "#0b1221"),
      text: pick("--text-color-1", "#cbd5e1"),
      grid: pick("--border-color-light", "rgba(225, 225, 225, 0.68)"),
      accent: pick("--accent", "#22c55e"),
      redCandle: pick("--red-candle", "#ef4444"),
      greenCandle: pick("--green-candle", "#22c55e"),
    };
  }, []);

  const applyThemeOptions = useCallback(() => {
    if (!chartRef.current) return;
    const colors = getThemeColors();
    chartRef.current.applyOptions({
      layout: {
        background: { color: colors.background },
        textColor: colors.text,
        fontSize: 8
      },
      grid: {
        vertLines: { color: colors.grid },
        horzLines: { color: colors.grid },
      },
      crosshair: {
        vertLine: {
          color: colors.accent,
          labelBackgroundColor: colors.accent,
          width: 1,
        },
        horzLine: {
          color: colors.accent,
          labelBackgroundColor: colors.accent,
          width: 1,
        },
      },
      rightPriceScale: { borderVisible: false, textColor: colors.text, textSize: 10, textStyle: { fontSize: 10 } },
      timeScale: { borderVisible: false, rightOffset: 6, barSpacing: 10, timeVisible: true, },
    });
    candleSeriesRef.current?.applyOptions({
      upColor: colors.greenCandle,
      borderUpColor: colors.greenCandle,
      wickUpColor: colors.greenCandle,
      downColor: colors.redCandle,
      borderDownColor: colors.redCandle,
      wickDownColor: colors.redCandle,
    });
  }, [getThemeColors, withAlpha]);

  useEffect(() => {
    if (!chartContainerRef.current) return;

    try {
      const container = chartContainerRef.current;

      const colors = getThemeColors();
      const chart = createChart(container, {
        ...measureDimensions(),
        layout: {
          background: { color: colors.background },
          textColor: colors.text,
        },
        grid: {
          vertLines: { color: colors.grid },
          horzLines: { color: colors.grid },
        },
        crosshair: {
          vertLine: { color: colors.accent, labelBackgroundColor: colors.accent, width: 1 },
          horzLine: { color: colors.accent, labelBackgroundColor: colors.accent, width: 1 },
        },
        rightPriceScale: { borderVisible: false, textColor: colors.text },
        timeScale: {
          borderVisible: false,
          rightOffset: 6,
          barSpacing: 10,
        },
      });

      const candles = chart.addSeries(CandlestickSeries, {
        upColor: colors.accent,
        borderUpColor: colors.accent,
        wickUpColor: colors.accent,
        downColor: withAlpha(colors.accent, 0.22),
        borderDownColor: withAlpha(colors.accent, 0.6),
        wickDownColor: withAlpha(colors.accent, 0.6),
      });
      candleSeriesRef.current = candles;

      const lineSlow = chart.addSeries(LineSeries, {
        color: colors.greenCandle,
        lineWidth: 3,
        crosshairMarkerVisible: false,
      });
      lineSeriesSlowRef.current = lineSlow; 

      const lineFast = chart.addSeries(LineSeries, {
        color: colors.accent,
        lineWidth: 2,
        crosshairMarkerVisible: false,
      });
      // lineSeriesFastRef.current = lineFast;

      const measureLine = chart.addSeries(LineSeries, {
        color: colors.accent,
        lineWidth: 2,
        lineStyle: 1, // Dashed line style
        crosshairMarkerVisible: true,
        priceLineVisible: false,
      });
      measureLineSeriesRef.current = measureLine;

      const handleCrosshairMove = (param) => {
        if (!param?.time || !param?.point) {
          setHoverValues(null);
          // Clear measure hover if crosshair leaves chart
          if (measureModeRef.current) {
            setMeasureHover(null);
          }
          return;
        }
        const candleData = param.seriesData?.get?.(candles);
        const lineSlowData = param.seriesData?.get?.(lineSlow);
        const lineFastData = param.seriesData?.get?.(lineFast);

        // Handle measurement mode hover preview
        if (measureModeRef.current && measurePointsRef.current.start && !measurePointsRef.current.end) {
          const time = candleData?.time;
          const price = candleData?.close ?? candleData?.high ?? candleData?.low ?? candleData?.open;
          if (Number.isFinite(time) && Number.isFinite(price)) {
            setMeasureHover({ time, price });
          } else {
            setMeasureHover(null);
          }
        } else {
          setMeasureHover(null);
        }

        if (!candleData && !lineSlowData && !lineFastData) {
          setHoverValues(null);
          return;
        }

        setHoverValues({
          open: candleData?.open,
          high: candleData?.high,
          low: candleData?.low,
          close: candleData?.close,
          lineSlow: lineSlowData?.value,
          lineSlowColor: lineSlowData?.color,
          lineFast: lineFastData?.value,
          lineFastColor: lineFastData?.color,
          deviationFactor: Math.abs(lineSlowData?.value - lineFastData?.value)
        });
      };

      chart.subscribeCrosshairMove(handleCrosshairMove);

      const handleClick = (param) => {
        if (!measureModeRef.current) return;
        
        const coord = param.point;
        if (!coord) return;

        const logicalPoint = param.seriesData?.get?.(candles);
        if (!logicalPoint || !logicalPoint.time) return;

        const time = logicalPoint.time;
        const price = logicalPoint.close ?? logicalPoint.high ?? logicalPoint.low ?? logicalPoint.open;

        if (!Number.isFinite(time) || !Number.isFinite(price)) return;

        setMeasurePoints((prev) => {
          let newState;
          if (!prev.start) {
            // First click - set start point
            newState = { start: { time, price }, end: null };
            setMeasureHover(null); // Clear any previous hover
          } else if (!prev.end) {
            // Second click - set end point, but check if it's the same point
            if (prev.start.time === time) {
              // Same point clicked twice, reset
              newState = { start: null, end: null };
              setMeasureHover(null);
            } else {
              newState = { start: prev.start, end: { time, price } };
              setMeasureHover(null); // Clear hover when end point is set
            }
          } else {
            // Third click - reset and start new measurement
            newState = { start: { time, price }, end: null };
            setMeasureHover(null);
          }
          measurePointsRef.current = newState;
          return newState;
        });
      };

      chart.subscribeClick(handleClick);

      chart.timeScale().fitContent();
      chartRef.current = chart;
      applyThemeOptions();

      const resize = () => {
        resizeToContainer();
      };

      // Keep both window resize + ResizeObserver to react to layout changes
      window.addEventListener("resize", resize);
      if (window.ResizeObserver) {
        resizeObserverRef.current = new ResizeObserver(resize);
        resizeObserverRef.current.observe(container.parentElement || container);
      }

      setLoading(false);

      return () => {
        window.removeEventListener("resize", resize);
        chart.unsubscribeCrosshairMove(handleCrosshairMove);
        chart.unsubscribeClick(handleClick);
        if (resizeObserverRef.current) {
          resizeObserverRef.current.disconnect();
        }
        chartRef.current = null;
        candleSeriesRef.current = null;
        lineSeriesSlowRef.current = null;
        lineSeriesFastRef.current = null;
        measureLineSeriesRef.current = null;
        chart.remove();
      };
    } catch (err) {
      setError("Unable to render chart");
      setLoading(false);
      console.error(err);
    }
  }, [applyThemeOptions, getThemeColors, measureDimensions, resizeToContainer, withAlpha]);

  // Push tick data into the candlestick series when fetched
  useEffect(() => {
    if (!ticks.length) return;
    if (candleSeriesRef.current) {
      const candleData = ticks.map((t) => ({ ...t, time: t.timeIst ?? t.time }));
      candleSeriesRef.current.setData(candleData);
    }

    if (lineSeriesSlowRef.current) {
      const lineData = ticks
        .filter((t) => typeof t.slow_tempx === "number" && Number.isFinite(t.slow_tempx))
        .map((t) => ({
          time: t.timeIst ?? t.time,
          value: t.slow_tempx,
          color: swapColor(t.swap_base),
        }));
      if (lineData.length) {
        lineSeriesSlowRef.current.setData(lineData);
      }
    }
    if (lineSeriesFastRef.current) {
      const lineDataBase = ticks
        .filter((t) => typeof t.fast_tempx === "number" && Number.isFinite(t.fast_tempx))
        .map((t) => ({
          time: t.timeIst ?? t.time,
          value: t.fast_tempx,
          color: swapColor(t.swap),
        }));
      if (lineDataBase.length) {
        lineSeriesFastRef.current.setData(lineDataBase);
      }
    }
    chartRef.current?.timeScale().fitContent();
  }, [ticks]);

  // Update measurement line when points change or hover
  useEffect(() => {
    if (!measureLineSeriesRef.current) return;
    
    // Determine end point: use actual end if set, otherwise use hover preview
    const endPoint = measurePoints.end || measureHover;
    
    if (measurePoints.start && endPoint) {
      // Ensure data is sorted by time (ascending order)
      const lineData = [
        { time: measurePoints.start.time, value: measurePoints.start.price },
        { time: endPoint.time, value: endPoint.price },
      ].sort((a, b) => {
        // Sort by time, if times are equal, keep original order
        if (a.time === b.time) {
          return 0;
        }
        return a.time - b.time;
      });
      
      // Only set data if we have valid, distinct points
      if (lineData.length === 2 && lineData[0].time !== lineData[1].time) {
        measureLineSeriesRef.current.setData(lineData);
      } else {
        // If same point clicked twice, clear the measurement
        measureLineSeriesRef.current.setData([]);
        if (measurePoints.end) {
          setMeasurePoints({ start: null, end: null });
        }
      }
    } else {
      measureLineSeriesRef.current.setData([]);
    }
  }, [measurePoints, measureHover]);

  useEffect(() => {
    if (!pendingDateKeyRef.current) return;
    if (focusDateKey(pendingDateKeyRef.current)) {
      pendingDateKeyRef.current = null;
    }
  }, [focusDateKey, ticks]);

  // Sync measureModeRef with measureMode state
  useEffect(() => {
    measureModeRef.current = measureMode;
    if (!measureMode) {
      // Clear hover when measurement mode is disabled
      setMeasureHover(null);
    }
  }, [measureMode]);

  // Sync measurePointsRef with measurePoints state
  useEffect(() => {
    measurePointsRef.current = measurePoints;
  }, [measurePoints]);

  // Re-apply colors when theme class on body changes
  useEffect(() => {
    if (typeof window === "undefined") return undefined;
    const apply = () => applyThemeOptions();
    const observer = new MutationObserver(apply);
    observer.observe(document.body, { attributes: true, attributeFilter: ["class"] });
    apply();
    return () => observer.disconnect();
  }, [applyThemeOptions]);

  const isBearish =
    hoverValues &&
    Number.isFinite(hoverValues.open) &&
    Number.isFinite(hoverValues.close)
      ? hoverValues.close < hoverValues.open
      : null;
  const candleColor =
    isBearish === null ? undefined : isBearish ? "var(--red-candle)" : "var(--green-candle)";
  const lineColor = hoverValues?.lineSlowColor || "var(--green-candle)";
  const lineFastColor = hoverValues?.lineFastColor || "var(--accent)";

  return (
    <section className="chart-panel">
      <div className="chart-layout">
        <div className="chart-area">
          <div className="chart-toolbar muted">Live OHLC / trades stream (placeholder)</div>
          <div className="chart-canvas">
            <div
              ref={chartContainerRef}
              className={`chart-canvas__inner ${measureMode ? 'measure-mode-active' : ''}`}
              style={{ width: "100%", height: "100%" }}
            />
            {hoverValues && (
              <div className="chart-hover-values">
                <span style={{ color: candleColor }}>
                  O {formatValue(hoverValues.open)}
                </span>
                <span style={{ color: candleColor }}>
                  H {formatValue(hoverValues.high)}
                </span>
                <span style={{ color: candleColor }}>
                  L {formatValue(hoverValues.low)}
                </span>
                <span style={{ color: candleColor }}>
                  C {formatValue(hoverValues.close)}
                </span>
                <span style={{ color: lineColor }}>
                  Line Slow {formatValue(hoverValues.lineSlow)}
                </span>
                <span style={{ color: lineFastColor }}>
                  Line Fast {formatValue(hoverValues.lineFast)}
                </span>
                <span style={{ color: 'white' }}>Deviation Factor: {formatValue(hoverValues.deviationFactor)}</span>
              </div>
            )}
            {(() => {
              // Show overlay if we have both start and end, OR if we have start and hover preview
              const endPoint = measurePoints.end || measureHover;
              if (!measurePoints.start || !endPoint) return null;
              
              const stats = calculateMeasureStats(measurePoints.start, endPoint);
              if (!stats) return null;
              
              const isPreview = !measurePoints.end && measureHover;
              
              return (
                <div className="chart-measure-overlay">
                  <div className={`chart-measure-stats ${isPreview ? 'preview' : ''}`}>
                    {isPreview && (
                      <div className="chart-measure-preview-label">Preview</div>
                    )}
                    <div className="chart-measure-row">
                      <span className="chart-measure-label">Price:</span>
                      <span className="chart-measure-value">{formatMeasureValue(measurePoints.start.price)} → {formatMeasureValue(endPoint.price)}</span>
                    </div>
                    <div className="chart-measure-row">
                      <span className="chart-measure-label">Change:</span>
                      <span className="chart-measure-value" style={{ color: stats.priceDiff >= 0 ? 'var(--green-candle)' : 'var(--red-candle)' }}>
                        {formatMeasureValue(stats.priceDiff)} ({formatMeasureValue(stats.priceDiffPercent)}%)
                      </span>
                    </div>
                    <div className="chart-measure-row">
                      <span className="chart-measure-label">Bars:</span>
                      <span className="chart-measure-value">{stats.barCount}</span>
                    </div>
                    <div className="chart-measure-row">
                      <span className="chart-measure-label">Time:</span>
                      <span className="chart-measure-value">{stats.timeDiff}</span>
                    </div>
                    {measurePoints.end && (
                      <button 
                        className="chart-measure-clear"
                        onClick={clearMeasurement}
                        title="Clear measurement"
                      >
                        <XCircle size={18} weight="regular" color="var(--accent)" />
                      </button>
                    )}
                  </div>
                </div>
              );
            })()}
            {/* {loading && <div className="chart-canvas__overlay muted">Loading chart…</div>}
            {error && <div className="chart-canvas__overlay error">{error}</div>} */}
          </div>
        </div>

        <aside className="chart-sidebar chart-sidebar--compact" aria-label="Chart quick actions">
          <div className="chart-sidebar__iconbar" role="toolbar" aria-orientation="vertical">
            <button 
              type="button" 
              className={`chart-sidebar__iconbtn ${measureMode ? 'active' : ''}`}
              aria-label="Measure tool"
              onClick={toggleMeasureMode}
              title={measureMode ? "Disable measurement tool" : "Enable measurement tool"}
            >
              <Ruler size={14} weight={measureMode ? "fill" : "regular"} />
            </button>
            <button type="button" className="chart-sidebar__iconbtn" aria-label="Chart action">
              <SlidersHorizontal size={14} weight="regular" />
            </button>
            <button type="button" className="chart-sidebar__iconbtn" aria-label="Chart action">
              <Table size={14} weight="regular" onClick={() => dispatch(setTradesDrawerOpen(true))} />
            </button>
            <button type="button" className="chart-sidebar__iconbtn" aria-label="Chart action">
              <TestTube size={14} weight="regular" />
            </button>
            <button type="button" className="chart-sidebar__iconbtn" aria-label="Chart action">
              <GearSix size={14} weight="regular" />
            </button>
          </div>
        </aside>
      </div>
    </section>
  );
}

export default ChartPanel;
