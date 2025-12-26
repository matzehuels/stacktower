import { createContext, useContext, useState, useCallback, type ReactNode } from 'react';
import type { AgentEvent, AnalysisReport } from '../types/api';

interface AgentContextType {
  events: AgentEvent[];
  report: AnalysisReport | null;
  isConnected: boolean;
  isVisible: boolean;
  jobId: string | null;
  addEvent: (event: AgentEvent) => void;
  setReport: (report: AnalysisReport | null) => void;
  setConnected: (connected: boolean) => void;
  setVisible: (visible: boolean) => void;
  setJobId: (jobId: string | null) => void;
  clear: () => void;
  toggleVisible: () => void;
}

const AgentContext = createContext<AgentContextType | null>(null);

export function AgentProvider({ children }: { children: ReactNode }) {
  const [events, setEvents] = useState<AgentEvent[]>([]);
  const [report, setReportState] = useState<AnalysisReport | null>(null);
  const [isConnected, setConnected] = useState(false);
  const [isVisible, setVisibleState] = useState(false);
  const [jobId, setJobIdState] = useState<string | null>(null);

  const addEvent = useCallback((event: AgentEvent) => {
    setEvents(prev => [...prev, event]);
    // Auto-show panel when events come in
    setVisibleState(true);
  }, []);

  const setReport = useCallback((r: AnalysisReport | null) => {
    setReportState(r);
  }, []);

  const setVisible = useCallback((visible: boolean) => {
    setVisibleState(visible);
  }, []);

  const setJobId = useCallback((id: string | null) => {
    setJobIdState(id);
  }, []);

  const clear = useCallback(() => {
    setEvents([]);
    setReportState(null);
    setConnected(false);
    setJobIdState(null);
  }, []);

  const toggleVisible = useCallback(() => {
    setVisibleState(prev => !prev);
  }, []);

  return (
    <AgentContext.Provider value={{
      events,
      report,
      isConnected,
      isVisible,
      jobId,
      addEvent,
      setReport,
      setConnected,
      setVisible,
      setJobId,
      clear,
      toggleVisible,
    }}>
      {children}
    </AgentContext.Provider>
  );
}

export function useAgent() {
  const context = useContext(AgentContext);
  if (!context) {
    throw new Error('useAgent must be used within AgentProvider');
  }
  return context;
}

