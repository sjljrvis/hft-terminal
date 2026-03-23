import { useEffect, useRef } from "react";
import { createChart, AreaSeries } from "lightweight-charts";

function EquityCurve({ trades, height = 120 }) {
  const containerRef = useRef(null);
  const chartRef = useRef(null);

  useEffect(() => {
    if (!containerRef.current || !trades || trades.length === 0) return;

    const isDark = document.body.classList.contains("theme-dark");
    const bg = isDark ? "#000000" : "#ffffff";
    const textColor = isDark ? "#9ca3af" : "#6b7280";
    const gridColor = isDark ? "rgba(255,255,255,0.04)" : "rgba(0,0,0,0.04)";

    const chart = createChart(containerRef.current, {
      height,
      layout: {
        background: { color: bg },
        textColor,
        fontFamily: "Google Sans Code, monospace",
        fontSize: 10,
      },
      grid: {
        vertLines: { color: gridColor },
        horzLines: { color: gridColor },
      },
      rightPriceScale: {
        borderVisible: false,
      },
      timeScale: {
        borderVisible: false,
        timeVisible: true,
        secondsVisible: false,
      },
      crosshair: {
        horzLine: { visible: false, labelVisible: false },
        vertLine: { labelVisible: false },
      },
      handleScale: false,
      handleScroll: false,
    });

    // Build cumulative P&L data
    let cumulative = 0;
    let peak = 0;
    const equityData = [];
    const drawdownData = [];

    const sortedTrades = [...trades].sort(
      (a, b) => new Date(a.exitTime).getTime() - new Date(b.exitTime).getTime()
    );

    let lastTime = 0;
    sortedTrades.forEach((trade) => {
      const profit = Number(trade.profit) || 0;
      cumulative += profit;
      if (cumulative > peak) peak = cumulative;
      const dd = peak - cumulative;
      let time = Math.floor(new Date(trade.exitTime).getTime() / 1000);
      if (time <= lastTime) time = lastTime + 1;
      lastTime = time;
      equityData.push({ time, value: cumulative });
      drawdownData.push({ time, value: -dd });
    });

    // Equity line
    const equitySeries = chart.addSeries(AreaSeries, {
      lineColor: cumulative >= 0 ? "#22c55e" : "#ef4444",
      topColor: cumulative >= 0 ? "rgba(34,197,94,0.15)" : "rgba(239,68,68,0.15)",
      bottomColor: "transparent",
      lineWidth: 1.5,
      priceLineVisible: false,
      lastValueVisible: true,
      crosshairMarkerVisible: false,
    });
    equitySeries.setData(equityData);

    // Drawdown area
    if (drawdownData.some((d) => d.value < 0)) {
      const ddSeries = chart.addSeries(AreaSeries, {
        lineColor: "rgba(239,68,68,0.5)",
        topColor: "transparent",
        bottomColor: "rgba(239,68,68,0.1)",
        lineWidth: 1,
        priceLineVisible: false,
        lastValueVisible: false,
        crosshairMarkerVisible: false,
      });
      ddSeries.setData(drawdownData);
    }

    chart.timeScale().fitContent();
    chartRef.current = chart;

    const ro = new ResizeObserver(() => {
      if (containerRef.current) {
        chart.applyOptions({
          width: containerRef.current.clientWidth,
          height: containerRef.current.clientHeight || height,
        });
      }
    });
    ro.observe(containerRef.current);

    return () => {
      ro.disconnect();
      chart.remove();
      chartRef.current = null;
    };
  }, [trades]);

  if (!trades || trades.length === 0) {
    return (
      <div className="equity-curve">
        <div className="equity-curve__header">Equity Curve</div>
        <div className="equity-curve__empty">No trades yet</div>
      </div>
    );
  }

  return (
    <div className="equity-curve">
      <div className="equity-curve__chart" ref={containerRef} />
    </div>
  );
}

export default EquityCurve;
