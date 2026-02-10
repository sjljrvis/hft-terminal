import { createSlice, createAsyncThunk } from '@reduxjs/toolkit';

// Calculate metrics from trades data
export const calculateMetrics = (trades) => {
  if (!trades || trades.length === 0) {
    return {
      totalPnl: 0,
      maxDrawdown: 0,
      totalTrades: 0,
      winRate: 0,
      winCount: 0,
      profitFactor: 0,
      profitTargetCount: 0,
      trailingStopCount: 0,
      stopLossCount: 0,
      signalCount: 0,
    };
  }

  let totalPnl = 0;
  let grossProfit = 0;
  let grossLoss = 0;
  let winCount = 0;
  let peakProfit = 0;
  let maxDrawdown = 0;
  let profitTargetCount = 0;
  let trailingStopCount = 0;
  let stopLossCount = 0;
  let signalCount = 0;

  trades.forEach((trade) => {
    const profit = trade.profit || 0;
    totalPnl += profit;

    if (profit > 0) {
      winCount++;
      grossProfit += profit;
    } else if (profit < 0) {
      grossLoss += Math.abs(profit);
    }

    // Track drawdown
    if (totalPnl > peakProfit) peakProfit = totalPnl;
    const drawdown = peakProfit - totalPnl;
    if (drawdown > maxDrawdown) maxDrawdown = drawdown;

    // Count exit reasons
    switch (trade.reason) {
      case "PROFIT_TARGET": profitTargetCount++; break;
      case "TRAILING_STOP": trailingStopCount++; break;
      case "STOP_LOSS": stopLossCount++; break;
      case "SIGNAL": signalCount++; break;
    }
  });

  const totalTrades = trades.length;
  const winRate = totalTrades > 0 ? (winCount / totalTrades) * 100 : 0;
  const profitFactor = grossLoss > 0 ? grossProfit / grossLoss : grossProfit > 0 ? Infinity : 0;

  return {
    totalPnl,
    maxDrawdown,
    totalTrades,
    winRate,
    winCount,
    profitFactor,
    profitTargetCount,
    trailingStopCount,
    stopLossCount,
    signalCount,
  };
};

// Async thunk for fetching backtest trades
export const fetchTrades = createAsyncThunk(
  'trades/fetchTrades',
  async (_, { rejectWithValue }) => {
    try {
      const res = await fetch('http://localhost:5001/backtest/trades');
      if (!res.ok) {
        throw new Error(`HTTP ${res.status}`);
      }
      const data = await res.json();
      return data;
    } catch (error) {
      return rejectWithValue(error.message);
    }
  }
);

// Async thunk for fetching live trades
export const fetchLiveTrades = createAsyncThunk(
  'trades/fetchLiveTrades',
  async (_, { rejectWithValue }) => {
    try {
      const res = await fetch('http://localhost:5001/live/trades');
      if (!res.ok) {
        throw new Error(`HTTP ${res.status}`);
      }
      const data = await res.json();
      return data;
    } catch (error) {
      return rejectWithValue(error.message);
    }
  }
);

const initialState = {
  trades: [],
  liveTrades: [],
  loading: false,
  liveLoading: false,
  error: null,
  liveError: null,
  filter: '',
  sort: 'entryTime-desc',
};

const tradesSlice = createSlice({
  name: 'trades',
  initialState,
  reducers: {
    setFilter: (state, action) => {
      state.filter = action.payload;
    },
    setSort: (state, action) => {
      state.sort = action.payload;
    },
    setTrades: (state, action) => {
      state.trades = action.payload;
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(fetchTrades.pending, (state) => {
        state.loading = true;
        state.error = null;
      })
      .addCase(fetchTrades.fulfilled, (state, action) => {
        state.loading = false;
        state.trades = action.payload;
        state.error = null;
      })
      .addCase(fetchTrades.rejected, (state, action) => {
        state.loading = false;
        state.error = action.payload;
      })
      .addCase(fetchLiveTrades.pending, (state) => {
        state.liveLoading = true;
        state.liveError = null;
      })
      .addCase(fetchLiveTrades.fulfilled, (state, action) => {
        state.liveLoading = false;
        state.liveTrades = action.payload;
        state.liveError = null;
      })
      .addCase(fetchLiveTrades.rejected, (state, action) => {
        state.liveLoading = false;
        state.liveError = action.payload;
      });
  },
});

export const { setFilter, setSort, setTrades } = tradesSlice.actions;

// Selectors
export const selectTrades = (state) => state.trades.trades;
export const selectTradesLoading = (state) => state.trades.loading;
export const selectTradesError = (state) => state.trades.error;
export const selectTradesFilter = (state) => state.trades.filter;
export const selectTradesSort = (state) => state.trades.sort;
export const selectLiveTrades = (state) => state.trades.liveTrades;
export const selectLiveTradesLoading = (state) => state.trades.liveLoading;
export const selectLiveTradesError = (state) => state.trades.liveError;

// Helper function to filter and sort trades
const filterAndSortTrades = (trades, filter, sort) => {
  const term = filter.trim().toLowerCase();
  const filtered = term
    ? trades.filter((t) =>
        [t?.entryTime, t?.exitTime, t?.type, t?.entryPrice, t?.exitPrice, t?.profit, t?.reason]
          .map((v) => (v === undefined || v === null ? '' : String(v).toLowerCase()))
          .some((v) => v.includes(term))
      )
    : trades;

  const [field, dir] = (sort || 'entryTime-desc').split('-');
  const sorted = [...filtered].sort((a, b) => {
    const mult = dir === 'asc' ? 1 : -1;
    switch (field) {
      case 'entryPrice':
        return mult * ((Number(a?.entryPrice) || 0) - (Number(b?.entryPrice) || 0));
      case 'exitPrice':
        return mult * ((Number(a?.exitPrice) || 0) - (Number(b?.exitPrice) || 0));
      case 'profit':
        return mult * ((Number(a?.profit) || 0) - (Number(b?.profit) || 0));
      case 'type':
        return mult * String(a?.type || '').localeCompare(String(b?.type || ''));
      case 'exitTime': {
        const ta = new Date(a?.exitTime || '').getTime() || 0;
        const tb = new Date(b?.exitTime || '').getTime() || 0;
        return mult * (ta - tb);
      }
      case 'entryTime':
      default: {
        const ta = new Date(a?.entryTime || '').getTime() || 0;
        const tb = new Date(b?.entryTime || '').getTime() || 0;
        return mult * (ta - tb);
      }
    }
  });

  return sorted;
};

// Computed selector for filtered and sorted trades (backtest)
export const selectFilteredSortedTrades = (state) => {
  const trades = state.trades.trades;
  const filter = state.trades.filter;
  const sort = state.trades.sort;
  return filterAndSortTrades(trades, filter, sort);
};

// Computed selector for filtered and sorted live trades
export const selectFilteredSortedLiveTrades = (state) => {
  const trades = state.trades.liveTrades;
  const filter = state.trades.filter;
  const sort = state.trades.sort;
  return filterAndSortTrades(trades, filter, sort);
};

// Unified selector: prefers live trades if available, otherwise backtest trades
export const selectActiveFilteredSortedTrades = (state) => {
  const liveTrades = state.trades.liveTrades;
  const backtestTrades = state.trades.trades;
  const filter = state.trades.filter;
  const sort = state.trades.sort;
  
  // Prefer live trades if they exist, otherwise use backtest trades
  const trades = liveTrades && liveTrades.length > 0 ? liveTrades : backtestTrades;
  return filterAndSortTrades(trades, filter, sort);
};

// Computed selector for trade metrics (backtest)
export const selectTradeMetrics = (state) => {
  return calculateMetrics(state.trades.trades);
};

// Computed selector for live trade metrics
export const selectLiveTradeMetrics = (state) => {
  return calculateMetrics(state.trades.liveTrades);
};

// Unified selector: prefers live metrics if available, otherwise backtest metrics
export const selectActiveTradeMetrics = (state) => {
  const liveTrades = state.trades.liveTrades;
  const backtestTrades = state.trades.trades;
  // Prefer live trades if they exist, otherwise use backtest trades
  const trades = liveTrades && liveTrades.length > 0 ? liveTrades : backtestTrades;
  return calculateMetrics(trades);
};

// Selector for total live P&L
export const selectLiveTotalPnL = (state) => {
  const trades = state.trades.liveTrades;
  if (!trades || trades.length === 0) return null;
  return trades.reduce((sum, trade) => {
    const profit = Number(trade.profit) || 0;
    return sum + profit;
  }, 0);
};

export default tradesSlice.reducer;
