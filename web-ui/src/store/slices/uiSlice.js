import { createSlice } from '@reduxjs/toolkit';

const initialState = {
  theme: 'dark',
  sidebarCollapsed: true,
  tradesDrawerOpen: false,
  equityDrawerOpen: false,
  logsDrawerOpen: false,
  logSize: 'min', // "min" | "max"
  activePanel: 'live', // "live" | "backtest"
  tickerTapeVisible: false,
};

const uiSlice = createSlice({
  name: 'ui',
  initialState,
  reducers: {
    setTheme: (state, action) => {
      state.theme = action.payload;
    },
    setSidebarCollapsed: (state, action) => {
      state.sidebarCollapsed = action.payload;
    },
    toggleSidebar: (state) => {
      state.sidebarCollapsed = !state.sidebarCollapsed;
    },
    setTradesDrawerOpen: (state, action) => {
      state.tradesDrawerOpen = action.payload;
    },
    toggleTradesDrawer: (state) => {
      state.tradesDrawerOpen = !state.tradesDrawerOpen;
    },
    toggleEquityDrawer: (state) => {
      state.equityDrawerOpen = !state.equityDrawerOpen;
    },
    setLogsDrawerOpen: (state, action) => {
      state.logsDrawerOpen = action.payload;
    },
    toggleLogsDrawer: (state) => {
      state.logsDrawerOpen = !state.logsDrawerOpen;
    },
    setLogSize: (state, action) => {
      state.logSize = action.payload;
    },
    toggleLogSize: (state) => {
      state.logSize = state.logSize === 'min' ? 'max' : 'min';
    },
    closeLogsDrawer: (state) => {
      state.logsDrawerOpen = false;
      state.logSize = 'min';
    },
    setActivePanel: (state, action) => {
      state.activePanel = action.payload;
    },
    setTickerTapeVisible: (state, action) => {
      state.tickerTapeVisible = action.payload;
    },
    toggleTickerTape: (state) => {
      state.tickerTapeVisible = !state.tickerTapeVisible;
    },
  },
});

export const {
  setTheme,
  setSidebarCollapsed,
  toggleSidebar,
  setTradesDrawerOpen,
  toggleTradesDrawer,
  toggleEquityDrawer,
  setLogsDrawerOpen,
  toggleLogsDrawer,
  setLogSize,
  toggleLogSize,
  closeLogsDrawer,
  setActivePanel,
  setTickerTapeVisible,
  toggleTickerTape,
} = uiSlice.actions;

// Selectors
export const selectTheme = (state) => state.ui.theme;
export const selectSidebarCollapsed = (state) => state.ui.sidebarCollapsed;
export const selectTradesDrawerOpen = (state) => state.ui.tradesDrawerOpen;
export const selectEquityDrawerOpen = (state) => state.ui.equityDrawerOpen;
export const selectLogsDrawerOpen = (state) => state.ui.logsDrawerOpen;
export const selectLogSize = (state) => state.ui.logSize;
export const selectActivePanel = (state) => state.ui.activePanel;
export const selectTickerTapeVisible = (state) => state.ui.tickerTapeVisible;

export default uiSlice.reducer;
