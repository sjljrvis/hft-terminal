import { useState } from "react";
import { MagnifyingGlass } from "phosphor-react";

function DBQueryPanel() {
  const [query, setQuery] = useState("");
  const [results, setResults] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const handleSearch = async () => {
    if (!query.trim()) {
      setError("Please enter a query");
      return;
    }

    setLoading(true);
    setError("");
    setResults([]);

    try {
      const encodedQuery = encodeURIComponent(query.trim());
      const response = await fetch(`http://localhost:5001/db/query?query=${encodedQuery}`);
      
      if (!response.ok) {
        const errorText = await response.text();
        throw new Error(errorText || `HTTP ${response.status}`);
      }

      const data = await response.json();
      setResults(Array.isArray(data) ? data : []);
    } catch (err) {
      setError(err.message || "Failed to execute query");
      setResults([]);
    } finally {
      setLoading(false);
    }
  };

  const handleKeyPress = (e) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSearch();
    }
  };

  const formatValue = (value) => {
    if (value === null || value === undefined) {
      return <span className="muted">null</span>;
    }
    if (typeof value === "number") {
      return value.toLocaleString("en-IN", { maximumFractionDigits: 2 });
    }
    if (typeof value === "boolean") {
      return value ? "true" : "false";
    }
    return String(value);
  };

  const getColumnNames = () => {
    if (results.length === 0) return [];
    return Object.keys(results[0]);
  };

  return (
    <section 
      className="card" 
      style={{ 
        height: "100%", 
        display: "flex", 
        flexDirection: "column",
        minHeight: 0,
        padding: "18px 16px"
      }}
    >
      <h3 style={{ textTransform: "uppercase", fontWeight: "100", fontSize: "12px", marginTop: 0, marginBottom: "16px", flexShrink: 0 }}>
        Database Query
      </h3>
      
      <div style={{ flexShrink: 0, marginBottom: "16px" }}>
        <div style={{ display: "flex", gap: "8px", marginBottom: "12px" }}>
          <div style={{ flex: 1, position: "relative" }}>
            <MagnifyingGlass 
              size={14} 
              weight="regular" 
              style={{ 
                position: "absolute", 
                left: "12px", 
                top: "50%", 
                transform: "translateY(-50%)",
                color: "var(--muted-color)",
                pointerEvents: "none"
              }} 
            />
            <input
              type="text"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              onKeyPress={handleKeyPress}
              placeholder="Enter SQL query (e.g., SELECT * FROM ticks WHERE symbol = 'hindalco')"
              style={{
                width: "100%",
                padding: "8px 12px 8px 32px",
                background: "var(--card-bg)",
                border: "1px solid var(--border-color)",
                borderRadius: "4px",
                color: "var(--text-color)",
                fontSize: "12px",
                fontFamily: "inherit",
              }}
            />
          </div>
          <button
            type="button"
            onClick={handleSearch}
            disabled={loading}
            style={{
              padding: "8px 16px",
              background: loading ? "var(--muted-color)" : "var(--accent)",
              border: "none",
              borderRadius: "4px",
              color: "#fff",
              fontSize: "12px",
              fontFamily: "inherit",
              cursor: loading ? "not-allowed" : "pointer",
              textTransform: "uppercase",
              letterSpacing: "0.5px",
              fontWeight: "500",
            }}
          >
            {loading ? "Executing..." : "Search"}
          </button>
        </div>
        {error && (
          <div
            style={{
              padding: "8px 12px",
              background: "rgba(239, 68, 68, 0.1)",
              border: "1px solid rgba(239, 68, 68, 0.3)",
              borderRadius: "4px",
              color: "var(--red-candle)",
              fontSize: "11px",
              marginTop: "8px",
            }}
          >
            {error}
          </div>
        )}
      </div>

      <div 
        style={{ 
          flex: 1,
          display: "flex",
          flexDirection: "column",
          minHeight: 0,
          overflow: "hidden"
        }}
      >
        {results.length > 0 ? (
          <>
            <div 
              style={{ 
                flex: 1,
                overflow: "auto",
                minHeight: 0
              }}
            >
              <table className="trades-table" style={{ width: "100%" }}>
                <thead>
                  <tr>
                    {getColumnNames().map((col) => (
                      <th key={col}>{col}</th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {results.map((row, idx) => (
                    <tr key={idx}>
                      {getColumnNames().map((col) => (
                        <td style={{ maxWidth: "200px" , textOverflow: "ellipsis", whiteSpace: "nowrap", overflowX: "scroll" }} key={col}>{formatValue(row[col])}</td>
                      ))}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <div style={{ flexShrink: 0, marginTop: "12px", fontSize: "11px", color: "var(--muted-color)" }}>
              {results.length} row{results.length !== 1 ? "s" : ""} returned
            </div>
          </>
        ) : (
          <div style={{ flex: 1, display: "flex", alignItems: "center", justifyContent: "center" }}>
            {!loading && !error && query && (
              <p className="muted" style={{ margin: 0 }}>
                No results. Enter a query and click Search to execute.
              </p>
            )}
            {!loading && !error && !query && (
              <p className="muted" style={{ margin: 0 }}>
                Enter a SQL query above to query the database. Examples:
                <br />
                <code style={{ fontSize: "11px", marginTop: "8px", display: "block" }}>
                  SELECT * FROM ticks WHERE symbol = 'hindalco' AND tf = '1'
                  <br />
                  SELECT * FROM orders
                </code>
              </p>
            )}
          </div>
        )}
      </div>
    </section>
  );
}

export default DBQueryPanel;
