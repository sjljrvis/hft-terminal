import { configureStore } from '@reduxjs/toolkit';
import tradesReducer from './slices/tradesSlice';
import logsReducer from './slices/logsSlice';
import uiReducer from './slices/uiSlice';

export const store = configureStore({
  reducer: {
    trades: tradesReducer,
    logs: logsReducer,
    ui: uiReducer,
  },
});
