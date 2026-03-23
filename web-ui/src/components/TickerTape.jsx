import { useEffect, useState, useRef } from "react";

const FALLBACK_TICKERS = [
  { symbol: "NIFTY 50", price: 23456.80, change: +124.35, changePct: +0.53 },
  { symbol: "BANK NIFTY", price: 49872.15, change: -187.60, changePct: -0.37 },
  { symbol: "SENSEX", price: 77234.50, change: +312.80, changePct: +0.41 },
  { symbol: "NIFTY IT", price: 34521.90, change: -98.45, changePct: -0.28 },
  { symbol: "NIFTY FIN", price: 21345.60, change: +56.20, changePct: +0.26 },
  { symbol: "INDIA VIX", price: 13.42, change: -0.38, changePct: -2.75 },
];

function TickerTape() {
  const [tickers, setTickers] = useState(FALLBACK_TICKERS);
  const trackRef = useRef(null);

  // Duplicate items for seamless looping
  const items = [...tickers, ...tickers];

  return (
    <div className="ticker-tape">
      <div className="ticker-tape__track" ref={trackRef}>
        {items.map((t, i) => (
          <span className="ticker-tape__item" key={i}>
            <span className="ticker-tape__symbol">{t.symbol}</span>
            <span className="ticker-tape__price">{t.price.toLocaleString("en-IN", { minimumFractionDigits: 2 })}</span>
            <span className={`ticker-tape__change ${t.change >= 0 ? "is-up" : "is-down"}`}>
              {t.change >= 0 ? "+" : ""}
              {t.change.toFixed(2)} ({t.change >= 0 ? "+" : ""}{t.changePct.toFixed(2)}%)
            </span>
          </span>
        ))}
      </div>
    </div>
  );
}

export default TickerTape;
