import { useMemo } from "react";
import { X } from "phosphor-react";
import { useAppDispatch, useAppSelector } from "../store/hooks";
import {
  selectEquityDrawerOpen,
  toggleEquityDrawer,
  selectActivePanel,
} from "../store/slices/uiSlice";
import {
  selectTrades,
  selectLiveTrades,
  selectTradeMetrics,
  selectLiveTradeMetrics,
} from "../store/slices/tradesSlice";
import EquityCurve from "./EquityCurve";

function EquityDrawer() {
  const dispatch = useAppDispatch();
  const open = useAppSelector(selectEquityDrawerOpen);
  const activePanel = useAppSelector(selectActivePanel);
  const liveTrades = useAppSelector(selectLiveTrades);
  const backtestTrades = useAppSelector(selectTrades);
  const trades = activePanel === "live" ? liveTrades : backtestTrades;
  const baseMetrics = useAppSelector(
    activePanel === "live" ? selectLiveTradeMetrics : selectTradeMetrics
  );

  const stats = useMemo(() => {
    if (!trades || trades.length === 0) return null;
    const wins = [];
    const losses = [];
    let longCount = 0;
    let shortCount = 0;
    let longPnl = 0;
    let shortPnl = 0;
    let bestTrade = -Infinity;
    let worstTrade = Infinity;
    let maxConsecWin = 0;
    let maxConsecLoss = 0;
    let curWin = 0;
    let curLoss = 0;

    const sorted = [...trades].sort(
      (a, b) => new Date(a.exitTime).getTime() - new Date(b.exitTime).getTime()
    );

    sorted.forEach((t) => {
      const p = Number(t.profit) || 0;
      const isLong = (t.type || "").toUpperCase() === "LONG";
      if (isLong) { longCount++; longPnl += p; }
      else { shortCount++; shortPnl += p; }
      if (p > bestTrade) bestTrade = p;
      if (p < worstTrade) worstTrade = p;
      if (p > 0) {
        wins.push(p);
        curWin++;
        curLoss = 0;
        if (curWin > maxConsecWin) maxConsecWin = curWin;
      } else if (p < 0) {
        losses.push(p);
        curLoss++;
        curWin = 0;
        if (curLoss > maxConsecLoss) maxConsecLoss = curLoss;
      } else {
        curWin = 0;
        curLoss = 0;
      }
    });

    const avgWin = wins.length > 0 ? wins.reduce((a, b) => a + b, 0) / wins.length : 0;
    const avgLoss = losses.length > 0 ? losses.reduce((a, b) => a + b, 0) / losses.length : 0;
    const expectancy = trades.length > 0
      ? (baseMetrics.winRate / 100) * avgWin + (1 - baseMetrics.winRate / 100) * avgLoss
      : 0;

    return {
      avgWin,
      avgLoss,
      bestTrade: bestTrade === -Infinity ? 0 : bestTrade,
      worstTrade: worstTrade === Infinity ? 0 : worstTrade,
      longCount,
      shortCount,
      longPnl,
      shortPnl,
      maxConsecWin,
      maxConsecLoss,
      expectancy,
    };
  }, [trades, baseMetrics]);

  const fmt = (v) => {
    if (v === undefined || v === null || isNaN(v)) return "-";
    const sign = v > 0 ? "+" : "";
    return `${sign}${v.toFixed(2)}`;
  };

  return (
    <div className={`equity-drawer ${open ? "is-open" : ""}`}>
      <div className="equity-drawer__header">
        <span className="equity-drawer__title">Equity Curve</span>
        <button
          type="button"
          className="equity-drawer__close"
          onClick={() => dispatch(toggleEquityDrawer())}
          aria-label="Close equity drawer"
        >
          <X size={14} weight="regular" />
        </button>
      </div>
      <div className="equity-drawer__body">
        <div className="equity-drawer__chart">
          <EquityCurve trades={trades} height={300} />
        </div>
        {stats && (
          <div className="equity-drawer__stats">
            <div className="eq-stats-section">
              <div className="eq-stats-section__title">Performance</div>
              <div className="eq-stat-row">
                <span className="eq-stat-label">Net P&L</span>
                <span className={`eq-stat-value ${baseMetrics.totalPnl >= 0 ? "positive" : "negative"}`}>
                  {fmt(baseMetrics.totalPnl)} pts
                </span>
              </div>
              <div className="eq-stat-row">
                <span className="eq-stat-label">Profit Factor</span>
                <span className="eq-stat-value">{baseMetrics.profitFactor === Infinity ? "INF" : baseMetrics.profitFactor?.toFixed(2) ?? "-"}</span>
              </div>
              <div className="eq-stat-row">
                <span className="eq-stat-label">Max Drawdown</span>
                <span className="eq-stat-value negative">{fmt(-baseMetrics.maxDrawdown)}</span>
              </div>
              <div className="eq-stat-row">
                <span className="eq-stat-label">Expectancy</span>
                <span className={`eq-stat-value ${stats.expectancy >= 0 ? "positive" : "negative"}`}>
                  {fmt(stats.expectancy)}
                </span>
              </div>
            </div>

            <div className="eq-stats-section">
              <div className="eq-stats-section__title">Win / Loss</div>
              <div className="eq-stat-row">
                <span className="eq-stat-label">Win Rate</span>
                <span className="eq-stat-value">{baseMetrics.winRate?.toFixed(1)}% <span className="eq-stat-sub">{baseMetrics.winCount}/{baseMetrics.totalTrades}</span></span>
              </div>
              <div className="eq-stat-row">
                <span className="eq-stat-label">Avg Win</span>
                <span className="eq-stat-value positive">{fmt(stats.avgWin)}</span>
              </div>
              <div className="eq-stat-row">
                <span className="eq-stat-label">Avg Loss</span>
                <span className="eq-stat-value negative">{fmt(stats.avgLoss)}</span>
              </div>
              <div className="eq-stat-row">
                <span className="eq-stat-label">Best Trade</span>
                <span className="eq-stat-value positive">{fmt(stats.bestTrade)}</span>
              </div>
              <div className="eq-stat-row">
                <span className="eq-stat-label">Worst Trade</span>
                <span className="eq-stat-value negative">{fmt(stats.worstTrade)}</span>
              </div>
              <div className="eq-stat-row">
                <span className="eq-stat-label">Consec. Win</span>
                <span className="eq-stat-value">{stats.maxConsecWin}</span>
              </div>
              <div className="eq-stat-row">
                <span className="eq-stat-label">Consec. Loss</span>
                <span className="eq-stat-value">{stats.maxConsecLoss}</span>
              </div>
            </div>

            <div className="eq-stats-section">
              <div className="eq-stats-section__title">Direction</div>
              <div className="eq-stat-row">
                <span className="eq-stat-label">Long</span>
                <span className="eq-stat-value">{stats.longCount} <span className={`eq-stat-sub ${stats.longPnl >= 0 ? "positive" : "negative"}`}>({fmt(stats.longPnl)})</span></span>
              </div>
              <div className="eq-stat-row">
                <span className="eq-stat-label">Short</span>
                <span className="eq-stat-value">{stats.shortCount} <span className={`eq-stat-sub ${stats.shortPnl >= 0 ? "positive" : "negative"}`}>({fmt(stats.shortPnl)})</span></span>
              </div>
            </div>

            <div className="eq-stats-section">
              <div className="eq-stats-section__title">Exit Breakdown</div>
              <div className="eq-stat-row">
                <span className="eq-stat-label">Target</span>
                <span className="eq-stat-value">{baseMetrics.profitTargetCount}</span>
              </div>
              <div className="eq-stat-row">
                <span className="eq-stat-label">Trailing</span>
                <span className="eq-stat-value">{baseMetrics.trailingStopCount}</span>
              </div>
              <div className="eq-stat-row">
                <span className="eq-stat-label">Stop Loss</span>
                <span className="eq-stat-value">{baseMetrics.stopLossCount}</span>
              </div>
              <div className="eq-stat-row">
                <span className="eq-stat-label">Signal</span>
                <span className="eq-stat-value">{baseMetrics.signalCount}</span>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

export default EquityDrawer;
