import { useCallback, useEffect, useRef, useState } from "react";
import axios from "axios";
import { createChart, CandlestickSeries, LineSeries } from "lightweight-charts";
import { SlidersHorizontal, ChartLine, TestTube, GearSix, Database } from "phosphor-react";

import { useAppDispatch } from "../store/hooks";
import { setTradesDrawerOpen } from "../store/slices/uiSlice";

function ChartPanel({ apiEndpoint = "http://localhost:5001/ticks" }) {
  const dispatch = useAppDispatch();
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [ticks, setTicks] = useState([]);
  const [hoverValues, setHoverValues] = useState(null);
  const chartContainerRef = useRef(null);
  const resizeObserverRef = useRef(null);
  const chartRef = useRef(null);
  const candleSeriesRef = useRef(null);
  const lineSeriesSlowRef = useRef(null);
  const lineSeriesFastRef = useRef(null);
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
      lineSeriesFastRef.current = lineFast;

      const handleCrosshairMove = (param) => {
        if (!param?.time || !param?.point) {
          setHoverValues(null);
          return;
        }
        const candleData = param.seriesData?.get?.(candles);
        const lineSlowData = param.seriesData?.get?.(lineSlow);
        const lineFastData = param.seriesData?.get?.(lineFast);

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
        if (resizeObserverRef.current) {
          resizeObserverRef.current.disconnect();
        }
        chartRef.current = null;
        candleSeriesRef.current = null;
        lineSeriesSlowRef.current = null;
        lineSeriesFastRef.current = null;
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

  useEffect(() => {
    if (!pendingDateKeyRef.current) return;
    if (focusDateKey(pendingDateKeyRef.current)) {
      pendingDateKeyRef.current = null;
    }
  }, [focusDateKey, ticks]);

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
              className="chart-canvas__inner"
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
            {/* {loading && <div className="chart-canvas__overlay muted">Loading chart…</div>}
            {error && <div className="chart-canvas__overlay error">{error}</div>} */}
          </div>
        </div>

        <aside className="chart-sidebar chart-sidebar--compact" aria-label="Chart quick actions">
          <div className="chart-sidebar__iconbar" role="toolbar" aria-orientation="vertical">
            <button type="button" className="chart-sidebar__iconbtn" aria-label="Chart action">
              <SlidersHorizontal size={14} weight="regular" />
            </button>
            <button type="button" className="chart-sidebar__iconbtn" aria-label="Chart action">
              <Database size={14} weight="regular" onClick={() => dispatch(setTradesDrawerOpen(true))} />
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
