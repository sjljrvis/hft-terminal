import { useEffect, useState } from "react";
import axios from "axios";

function StrategyControls() {
  const [config, setConfig] = useState(null);
  const [dirty, setDirty] = useState(false);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    axios
      .get("http://localhost:5001/live/config")
      .then(({ data }) => setConfig(data))
      .catch(() => {});
  }, []);

  const update = (key, value) => {
    setConfig((prev) => ({ ...prev, [key]: value }));
    setDirty(true);
  };

  const save = async () => {
    setSaving(true);
    try {
      const { data } = await axios.post(
        "http://localhost:5001/live/config",
        config
      );
      setConfig(data);
      setDirty(false);
    } catch {}
    setSaving(false);
  };

  const reset = async () => {
    try {
      const { data } = await axios.post("http://localhost:5001/live/config", {
        activationMFEPts: 500,
        mfeCaptureRatio: 0.4,
        signalConfirmBars: 0,
        enableFixedSL: false,
        fixedSL: -50,
      });
      setConfig(data);
      setDirty(false);
    } catch {}
  };

  if (!config) return null;

  return (
    <section className="ln__section sc">
      <div className="ln__section-label">Strategy Config</div>
      <ParamRow
        label="Activation MFE"
        value={config.activationMFEPts}
        onChange={(v) => update("activationMFEPts", v)}
        step={10}
        suffix=" pts"
      />
      <ParamRow
        label="MFE Capture"
        value={config.mfeCaptureRatio}
        onChange={(v) => update("mfeCaptureRatio", v)}
        step={0.05}
        precision={2}
      />
      <ParamRow
        label="Confirm Bars"
        value={config.signalConfirmBars}
        onChange={(v) => update("signalConfirmBars", Math.max(0, Math.round(v)))}
        step={1}
        precision={0}
      />
      <ParamRow
        label="Fixed SL"
        value={config.fixedSL}
        onChange={(v) => update("fixedSL", v)}
        step={5}
        suffix=" pts"
      />
      <div className="sc__toggle-row">
        <span>Enable Fixed SL</span>
        <button
          className={`sc__toggle ${config.enableFixedSL ? "is-on" : ""}`}
          onClick={() => update("enableFixedSL", !config.enableFixedSL)}
        >
          <span className="sc__toggle-knob" />
        </button>
      </div>
      {dirty && (
        <div className="sc__actions">
          <button className="sc__btn sc__btn--save" onClick={save} disabled={saving}>
            {saving ? "Saving..." : "Apply"}
          </button>
          <button className="sc__btn sc__btn--reset" onClick={reset}>
            Reset
          </button>
        </div>
      )}
    </section>
  );
}

function ParamRow({ label, value, onChange, step, precision = 2, suffix = "" }) {
  const display =
    precision === 0 ? Math.round(value) : Number(value).toFixed(precision);

  return (
    <div className="sc__param">
      <span className="sc__param-label">{label}</span>
      <div className="sc__param-controls">
        <button
          className="sc__param-btn"
          onClick={() => onChange(Number((value - step).toFixed(4)))}
        >
          -
        </button>
        <span className="sc__param-value">
          {display}
          {suffix}
        </span>
        <button
          className="sc__param-btn"
          onClick={() => onChange(Number((value + step).toFixed(4)))}
        >
          +
        </button>
      </div>
    </div>
  );
}

export default StrategyControls;
