import { createSlice } from '@reduxjs/toolkit';

const initialState = {
  theme: 'dark',
  sidebarCollapsed: true,
  tradesDrawerOpen: false,
  logsDrawerOpen: false,
  logSize: 'min', // "min" | "max"
  activePanel: 'live', // "live" | "backtest"
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
  },
});

export const {
  setTheme,
  setSidebarCollapsed,
  toggleSidebar,
  setTradesDrawerOpen,
  toggleTradesDrawer,
  setLogsDrawerOpen,
  toggleLogsDrawer,
  setLogSize,
  toggleLogSize,
  closeLogsDrawer,
  setActivePanel,
} = uiSlice.actions;

// Selectors
export const selectTheme = (state) => state.ui.theme;
export const selectSidebarCollapsed = (state) => state.ui.sidebarCollapsed;
export const selectTradesDrawerOpen = (state) => state.ui.tradesDrawerOpen;
export const selectLogsDrawerOpen = (state) => state.ui.logsDrawerOpen;
export const selectLogSize = (state) => state.ui.logSize;
export const selectActivePanel = (state) => state.ui.activePanel;

export default uiSlice.reducer;
