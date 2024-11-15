import { type Destructor, observe } from "@synnaxlabs/x";
import { createContext, type PropsWithChildren, useContext, useRef } from "react";

export interface LogsContextValue {
  subscribeToLogs: (v: (v: LogMessage) => void) => [LogMessage[], Destructor];
  getLogs: () => LogMessage[];
  addLog: (v: LogMessage) => void;
}

export const LogsContext = createContext<LogsContextValue>({
  subscribeToLogs: () => [[], () => {}],
  getLogs: () => [],
  addLog: () => {},
});

interface LogsProviderProps extends PropsWithChildren<{}> {}

export const useLogsContext = () => useContext(LogsContext);

export interface LogMessage {
  level: string;
  msg: string;
  ts: number;
  error: string;
}

export const parseLogMessage = (v: string): LogMessage | null => {
  try {
    const log = JSON.parse(v);
    return log;
  } catch {
    console.warn("Failed to parse log", v);
    return null;
  }
};

export const LogsProvider = ({ children }: LogsProviderProps) => {
  const logsRef = useRef<LogMessage[]>([]);
  const obsRev = useRef<observe.Observer<LogMessage>>(new observe.Observer());

  const addLog = (v: LogMessage): void => {
    logsRef.current.push(v);
    obsRev.current.notify(v);
  };

  const subscribeToLogs = (v: (v: LogMessage) => void): [LogMessage[], Destructor] => {
    const logs = logsRef.current;
    const d = obsRev.current.onChange(v);
    return [logs, d];
  };

  const getLogs = (): LogMessage[] => logsRef.current;

  return (
    <LogsContext.Provider value={{ addLog, subscribeToLogs, getLogs }}>
      {children}
    </LogsContext.Provider>
  );
};
