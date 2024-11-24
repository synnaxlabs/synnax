export interface History<T> {
  past: T[];
  present: T;
  future: T[];
  _latestUnfiltered: T | null;
  group: string | null;
}

export interface Action {
  type: string;
  index?: number;
}

export interface UndoableConfig {
  debug?: boolean;
  limit?: number;
  filter?: (action: Action, currentState: any, history: History<any>) => boolean;
  groupBy?: (action: Action, currentState: any, history: History<any>) => string | null;
  undoType?: string;
  redoType?: string;
  jumpToPastType?: string;
  jumpToFutureType?: string;
  jumpType?: string;
  neverSkipReducer?: boolean;
  ignoreInitialState?: boolean;
  syncFilter?: boolean;
  initTypes?: string[];
  clearHistoryType?: string[];
}
