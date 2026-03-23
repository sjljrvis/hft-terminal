import { useEffect, useState } from "react";

const DUMMY_NEWS = [
  { ts: "10:05", tag: "RBI", text: "RBI holds repo rate at 6.5%, maintains neutral stance" },
  { ts: "09:58", tag: "NIFTY", text: "NIFTY 50 breaches 23,500 support — bearish engulfing on 15m" },
  { ts: "09:45", tag: "FII", text: "FII net sold \u20B92,340 Cr in cash segment yesterday" },
  { ts: "09:32", tag: "GLOBAL", text: "US futures flat ahead of FOMC minutes release tonight" },
  { ts: "09:20", tag: "VIX", text: "India VIX spikes 8% — options premium elevated" },
  { ts: "09:15", tag: "PRE-MKT", text: "SGX Nifty indicates gap-down open at 23,420" },
  { ts: "09:00", tag: "CRUDE", text: "Crude oil rises 1.2% on Middle East" },
];

const DUMMY_EVENTS = [
  { ts: "09:15:02", type: "log", text: "Executor started (mode=live)" },
  { ts: "09:15:03", type: "log", text: "Loading history: NIFTY50-INDEX 1m" },
  { ts: "09:15:08", type: "log", text: "Kalman filter initialized (fast=0.03, slow=0.008)" },
  { ts: "09:15:10", type: "log", text: "Indicators computed: 4,832 rows" },
  { ts: "09:16:01", type: "log", text: "Regime: BULLISH (kalman spread +12.4)" },
  { ts: "09:18:33", type: "entry", text: "ENTRY BUY @ 23,456.80" },
  { ts: "09:18:33", type: "log", text: "Position opened: BUY qty=1 target=+45 sl=-30" },
  { ts: "09:22:15", type: "log", text: "Trailing stop armed at +28.5 pts" },
  { ts: "09:24:47", type: "exit", text: "EXIT BUY @ 23,498.30 (TRAILING_STOP)" },
  { ts: "09:24:47", type: "log", text: "Trade closed: +41.50 pts | peak +48.2 | trough -4.1" },
  { ts: "09:26:10", type: "log", text: "Regime shift: BEARISH (kalman spread -8.7)" },
  { ts: "09:31:05", type: "entry", text: "ENTRY SELL @ 23,512.40" },
  { ts: "09:31:05", type: "log", text: "Position opened: SELL qty=1 target=+45 sl=-30" },
  { ts: "09:35:22", type: "log", text: "Unrealized P&L: +22.6 pts" },
  { ts: "09:38:44", type: "exit", text: "EXIT SELL @ 23,467.90 (PROFIT_TARGET)" },
  { ts: "09:38:44", type: "log", text: "Trade closed: +44.50 pts | peak +46.1 | trough -2.3" },
  { ts: "09:40:01", type: "log", text: "Session P&L: +86.00 pts (2W 0L)" },
  { ts: "09:42:18", type: "log", text: "Regime: BULLISH (kalman spread +15.2)" },
  { ts: "09:45:30", type: "entry", text: "ENTRY BUY @ 23,478.60" },
  { ts: "09:49:12", type: "log", text: "Trailing stop armed at +31.2 pts" },
  { ts: "09:51:55", type: "exit", text: "EXIT BUY @ 23,452.10 (STOP_LOSS)" },
  { ts: "09:51:55", type: "log", text: "Trade closed: -26.50 pts | peak +33.4 | trough -26.5" },
  { ts: "09:53:00", type: "log", text: "Session P&L: +59.50 pts (2W 1L)" },
  { ts: "09:55:14", type: "log", text: "Margin check: available 4,940.50 | utilized 0.00" },
  { ts: "10:01:33", type: "entry", text: "ENTRY SELL @ 23,430.20" },
  { ts: "10:05:48", type: "log", text: "Unrealized P&L: +18.3 pts" },
  { ts: "10:08:02", type: "exit", text: "EXIT SELL @ 23,392.70 (SIGNAL)" },
  { ts: "10:08:02", type: "log", text: "Trade closed: +37.50 pts | peak +39.8 | trough -1.2" },
  { ts: "10:08:05", type: "log", text: "Session P&L: +97.00 pts (3W 1L) | WR 75%" },
];

const TAG_COLORS = {
  RBI: "var(--accent)",
  NIFTY: "var(--green-candle)",
  FII: "var(--red-candle)",
  GLOBAL: "#6b7cff",
  VIX: "var(--red-candle)",
  "PRE-MKT": "var(--accent)",
  CRUDE: "#d4a853",
};

function NewsFeed() {
  return (
    <div className="ap__news">
      {DUMMY_NEWS.map((n, i) => (
        <div key={i} className="ap__news-item">
          <span className="ap__news-text">{n.text}</span>
          <span className="ap__news-ts">{n.ts}</span>
        </div>
      ))}
    </div>
  );
}

function ActivityFeed() {
  const [logs, setLogs] = useState(DUMMY_EVENTS.slice().reverse());

  useEffect(() => {
    const ws = new WebSocket("ws://localhost:5001/ws/events");
    ws.onmessage = (evt) => {
      try {
        const msg = JSON.parse(evt.data);
        const ts = msg.timestamp
          ? new Date(msg.timestamp * 1000).toLocaleTimeString("en-GB", {
              hour: "2-digit",
              minute: "2-digit",
              second: "2-digit",
            })
          : "";
        let text = "";
        let type = "log";
        if (msg.type === "log" && msg.data?.message) {
          text = msg.data.message;
        } else if (msg.type === "event" && msg.data) {
          const d = msg.data;
          type = d.type === "ENTRY" ? "entry" : "exit";
          text = `${d.type} ${d.kind} @ ${d.entryPrice}${d.reason ? ` (${d.reason})` : ""}`;
        } else {
          text = JSON.stringify(msg.data ?? msg);
        }
        setLogs((prev) => [{ ts, text, type }, ...prev].slice(0, 100));
      } catch {}
    };
    return () => ws.close();
  }, []);

  return (
    <div className="ap__feed">
      {logs.map((l, i) => (
        <div key={i} className={`ap__feed-item ap__feed-item--${l.type}`}>
          <span className="ap__feed-ts">{l.ts}</span>
          <span className="ap__feed-type">
            {l.type === "entry" ? "ENTRY" : l.type === "exit" ? "EXIT" : "LOG"}
          </span>
          <span className="ap__feed-text">{l.text}</span>
        </div>
      ))}
    </div>
  );
}

export default function ActivityPanel() {
  return (
    <aside className="ap">
      <section className="ap__section ap__section--news">
        <div className="ap__section-label">News</div>
        <NewsFeed />
      </section>
      <section className="ap__section ap__section--feed">
        <div className="ap__section-label">Activity Feed</div>
        <ActivityFeed />
      </section>
    </aside>
  );
}
