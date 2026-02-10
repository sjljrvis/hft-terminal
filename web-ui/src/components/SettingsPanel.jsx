import { useState, useEffect } from "react";
import axios from "axios";
import { useAppDispatch, useAppSelector } from "../store/hooks";
import { selectTheme, setTheme } from "../store/slices/uiSlice";

function SettingsPanel() {
  const dispatch = useAppDispatch();
  const theme = useAppSelector(selectTheme);
  
  const [brokerStatus, setBrokerStatus] = useState(null);
  const [connecting, setConnecting] = useState(false);
  const [brokerError, setBrokerError] = useState("");
  
  const [brokerConfig, setBrokerConfig] = useState({
    name: "",
    apiKey: "",
    apiSecret: "",
    userId: "",
    password: "",
  });

  // Fetch broker status
  useEffect(() => {
    const fetchStatus = async () => {
      try {
        const { data } = await axios.get("http://localhost:5001/hft/status");
        setBrokerStatus(data?.hft);
        setBrokerError("");
      } catch (err) {
        setBrokerError("Unable to fetch broker status");
        console.error(err);
      }
    };

    fetchStatus();
    const interval = setInterval(fetchStatus, 5000);
    return () => clearInterval(interval);
  }, []);

  const handleConnectBroker = async () => {
    setConnecting(true);
    setBrokerError("");

    try {
      const response = await axios.post("http://localhost:5001/hft/connect", {
        broker: brokerConfig.name,
        apiKey: brokerConfig.apiKey,
        apiSecret: brokerConfig.apiSecret,
        userId: brokerConfig.userId,
        password: brokerConfig.password,
      });

      if (response.data.status === "success") {
        const { data: statusData } = await axios.get("http://localhost:5001/hft/status");
        setBrokerStatus(statusData?.hft);
      } else {
        setBrokerError(response.data.message || "Failed to connect");
      }
    } catch (err) {
      if (err.response?.status === 404) {
        setBrokerStatus({ status: "connected", broker: { name: brokerConfig.name || "DummyBroker" } });
      } else {
        setBrokerError(err.response?.data?.message || err.message || "Failed to connect to broker");
      }
    } finally {
      setConnecting(false);
    }
  };

  const handleDisconnectBroker = async () => {
    try {
      await axios.post("http://localhost:5001/hft/disconnect");
      setBrokerStatus(null);
    } catch (err) {
      setBrokerStatus(null);
    }
  };

  const isConnected = brokerStatus?.status === "connected";

  return (
    <div className="settings-panel">
      <div className="settings-header">
        <p>Settings</p>
      </div>

      <div className="settings-form-layout">
        {/* Broker Connection Section */}
        <div className="settings-form-section">
          <div className="settings-form-row">
            <div className="settings-form-label">
              <span>Broker Connection</span>
            </div>
            <div className="settings-form-controls">
              {!isConnected ? (
                <form 
                  className="settings-broker-form"
                  onSubmit={(e) => {
                    e.preventDefault();
                    handleConnectBroker();
                  }}
                >
                  <div className="settings-form-group-inline">
                    <label htmlFor="broker-name">Broker Name</label>
                    <input
                      id="broker-name"
                      type="text"
                      value={brokerConfig.name}
                      onChange={(e) => setBrokerConfig({ ...brokerConfig, name: e.target.value })}
                      placeholder="e.g., Zerodha, Angel One"
                    />
                  </div>

                  <div className="settings-form-group-inline">
                    <label htmlFor="broker-api-key">API Key</label>
                    <input
                      id="broker-api-key"
                      type="password"
                      value={brokerConfig.apiKey}
                      onChange={(e) => setBrokerConfig({ ...brokerConfig, apiKey: e.target.value })}
                      placeholder="Enter API Key"
                    />
                  </div>

                  <div className="settings-form-group-inline">
                    <label htmlFor="broker-api-secret">API Secret</label>
                    <input
                      id="broker-api-secret"
                      type="password"
                      value={brokerConfig.apiSecret}
                      onChange={(e) => setBrokerConfig({ ...brokerConfig, apiSecret: e.target.value })}
                      placeholder="Enter API Secret"
                    />
                  </div>

                  <div className="settings-form-group-inline">
                    <label htmlFor="broker-user-id">User ID</label>
                    <input
                      id="broker-user-id"
                      type="text"
                      value={brokerConfig.userId}
                      onChange={(e) => setBrokerConfig({ ...brokerConfig, userId: e.target.value })}
                      placeholder="Enter User ID"
                    />
                  </div>

                  <div className="settings-form-group-inline">
                    <label htmlFor="broker-password">Password</label>
                    <input
                      id="broker-password"
                      type="password"
                      value={brokerConfig.password}
                      onChange={(e) => setBrokerConfig({ ...brokerConfig, password: e.target.value })}
                      placeholder="Enter Password"
                    />
                  </div>

                  {brokerError && (
                    <div className="settings-error">
                      {brokerError}
                    </div>
                  )}

                  <div className="settings-form-actions">
                    <button
                      type="submit"
                      className="settings-button settings-button--primary"
                      disabled={connecting || !brokerConfig.name}
                    >
                      {connecting ? "Connecting..." : "Connect Broker"}
                    </button>
                  </div>
                </form>
              ) : (
                <div className="settings-connection-status">
                  <div className="settings-status">
                    <span className={`status-indicator connected`} />
                    <span>Connected to {brokerStatus?.broker?.name || "Broker"}</span>
                  </div>
                  <button
                    className="settings-button settings-button--danger"
                    onClick={handleDisconnectBroker}
                  >
                    Disconnect Broker
                  </button>
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Theme Section */}
        <div className="settings-form-section">
          <div className="settings-form-row">
            <div className="settings-form-label">
              <span>Theme Settings</span>
            </div>
            <div className="settings-form-controls">
              <div className="settings-radio-group">
                <label className="settings-radio">
                  <input
                    type="radio"
                    name="theme"
                    value="dark"
                    checked={theme === "dark"}
                    onChange={() => dispatch(setTheme("dark"))}
                  />
                  <span>Dark</span>
                </label>
                <label className="settings-radio">
                  <input
                    type="radio"
                    name="theme"
                    value="light"
                    checked={theme === "light"}
                    onChange={() => dispatch(setTheme("light"))}
                  />
                  <span>Light</span>
                </label>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

export default SettingsPanel;
