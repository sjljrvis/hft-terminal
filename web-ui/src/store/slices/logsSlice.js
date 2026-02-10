import { createSlice, createAsyncThunk } from '@reduxjs/toolkit';
import { apiBase } from '../../api';

// Async thunk for fetching logs
export const fetchLogs = createAsyncThunk(
  'logs/fetchLogs',
  async (_, { rejectWithValue }) => {
    try {
      const res = await fetch(`${apiBase}/logs`);
      if (!res.ok) {
        throw new Error(`HTTP ${res.status}`);
      }
      const data = await res.json();
      const lines = Array.isArray(data) ? data : data?.lines ?? [];
      const normalized = lines.map((line) => String(line));
      return normalized.slice(-200);
    } catch (error) {
      const fallback = `[local] ${new Date().toISOString()} — backend logs unavailable`;
      return rejectWithValue({ message: 'Unable to load logs from backend', fallback });
    }
  }
);

const initialState = {
  logs: [],
  loading: false,
  error: '',
  filter: '',
};

const logsSlice = createSlice({
  name: 'logs',
  initialState,
  reducers: {
    setFilter: (state, action) => {
      state.filter = action.payload;
    },
    addFallbackLog: (state, action) => {
      state.logs = [...state.logs.slice(-180), action.payload];
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(fetchLogs.pending, (state) => {
        state.loading = true;
        state.error = '';
      })
      .addCase(fetchLogs.fulfilled, (state, action) => {
        state.loading = false;
        state.logs = action.payload;
        state.error = '';
      })
      .addCase(fetchLogs.rejected, (state, action) => {
        state.loading = false;
        state.error = action.payload?.message || 'Unable to load logs from backend';
        if (action.payload?.fallback) {
          state.logs = [...state.logs.slice(-180), action.payload.fallback];
        }
      });
  },
});

export const { setFilter, addFallbackLog } = logsSlice.actions;

// Selectors
export const selectLogs = (state) => state.logs.logs;
export const selectLogsLoading = (state) => state.logs.loading;
export const selectLogsError = (state) => state.logs.error;
export const selectLogsFilter = (state) => state.logs.filter;

// Computed selector for filtered logs
export const selectFilteredLogs = (state) => {
  const logs = state.logs.logs;
  const filter = state.logs.filter;
  if (!filter) return logs;
  const filterLower = filter.toLowerCase();
  return logs.filter((line) => line.toLowerCase().includes(filterLower));
};

export default logsSlice.reducer;
