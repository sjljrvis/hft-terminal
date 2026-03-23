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
} from "../store/slices/tradesSlice";
import EquityCurve from "./EquityCurve";

function EquityDrawer() {
  const dispatch = useAppDispatch();
  const open = useAppSelector(selectEquityDrawerOpen);
  const activePanel = useAppSelector(selectActivePanel);
  const liveTrades = useAppSelector(selectLiveTrades);
  const backtestTrades = useAppSelector(selectTrades);
  const trades = activePanel === "live" ? liveTrades : backtestTrades;

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
        <EquityCurve trades={trades} height={300} />
      </div>
    </div>
  );
}

export default EquityDrawer;
