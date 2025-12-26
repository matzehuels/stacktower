import { useEffect, useRef } from 'react';
import type { AgentEvent } from '../types/api';

interface AgentStreamProps {
  events: AgentEvent[];
  isConnected: boolean;
}

export function AgentStream({ events, isConnected }: AgentStreamProps) {
  const scrollRef = useRef<HTMLDivElement>(null);

  // Auto-scroll to bottom on new events
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [events]);

  if (events.length === 0 && !isConnected) {
    return null;
  }

  return (
    <div className="panel h-full flex flex-col">
      <div className="panel-header flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="font-semibold">Agent Activity</span>
          {isConnected && (
            <span className="flex items-center gap-1.5 text-xs text-[var(--color-success)]">
              <span className="w-1.5 h-1.5 bg-[var(--color-success)] rounded-full animate-pulse" />
              Live
            </span>
          )}
        </div>
        <span className="text-xs text-[var(--color-text-muted)]">
          {events.length} events
        </span>
      </div>
      
      <div 
        ref={scrollRef}
        className="flex-1 overflow-y-auto p-4 space-y-3 font-mono text-sm bg-[var(--color-sidebar)]"
      >
        {events.map((event, i) => (
          <AgentEventItem key={i} event={event} />
        ))}
        
        {isConnected && (
          <div className="flex items-center gap-2 text-[var(--color-sidebar-text)]">
            <div className="flex gap-1">
              <span className="w-1.5 h-1.5 bg-[var(--color-primary)] rounded-full animate-bounce" style={{ animationDelay: '0ms' }} />
              <span className="w-1.5 h-1.5 bg-[var(--color-primary)] rounded-full animate-bounce" style={{ animationDelay: '150ms' }} />
              <span className="w-1.5 h-1.5 bg-[var(--color-primary)] rounded-full animate-bounce" style={{ animationDelay: '300ms' }} />
            </div>
            <span className="text-xs">Thinking...</span>
          </div>
        )}
      </div>
    </div>
  );
}

function AgentEventItem({ event }: { event: AgentEvent }) {
  const getIcon = () => {
    switch (event.type) {
      case 'thinking':
        return (
          <svg className="w-4 h-4 text-[var(--color-primary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
          </svg>
        );
      case 'tool_call':
        return (
          <svg className="w-4 h-4 text-amber-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
          </svg>
        );
      case 'tool_result':
        return (
          <svg className="w-4 h-4 text-[var(--color-success)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M5 13l4 4L19 7" />
          </svg>
        );
      case 'progress':
        return (
          <svg className="w-4 h-4 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
          </svg>
        );
      case 'complete':
        return (
          <svg className="w-4 h-4 text-[var(--color-success)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        );
      case 'error':
        return (
          <svg className="w-4 h-4 text-[var(--color-error)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        );
      default:
        return null;
    }
  };

  const getTextColor = () => {
    switch (event.type) {
      case 'thinking':
        return 'text-[var(--color-sidebar-text-active)]';
      case 'tool_call':
        return 'text-amber-300';
      case 'tool_result':
        return 'text-emerald-300';
      case 'progress':
        return 'text-blue-300';
      case 'complete':
        return 'text-[var(--color-success)]';
      case 'error':
        return 'text-[var(--color-error)]';
      default:
        return 'text-[var(--color-sidebar-text)]';
    }
  };

  const formatTime = (timestamp: string) => {
    const date = new Date(timestamp);
    return date.toLocaleTimeString('en-US', { 
      hour12: false, 
      hour: '2-digit', 
      minute: '2-digit', 
      second: '2-digit' 
    });
  };

  return (
    <div className="flex items-start gap-3 group">
      <div className="flex-shrink-0 mt-0.5">{getIcon()}</div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className={`${getTextColor()} break-words`}>
            {event.message || event.type}
          </span>
          <span className="text-xs text-[var(--color-sidebar-text)] opacity-50 group-hover:opacity-100 transition-opacity">
            {formatTime(event.timestamp)}
          </span>
        </div>
        
        {event.tool_name && (
          <div className="mt-1 text-xs text-[var(--color-sidebar-text)]">
            <span className="px-1.5 py-0.5 bg-white/10 rounded">
              {event.tool_name}
            </span>
          </div>
        )}
      </div>
    </div>
  );
}

// Mini version for inline display
export function AgentStreamMini({ events, isConnected }: AgentStreamProps) {
  const latestEvent = events[events.length - 1];
  
  if (!latestEvent && !isConnected) {
    return null;
  }

  return (
    <div className="flex items-center gap-2 text-sm text-[var(--color-text-secondary)]">
      {isConnected && (
        <span className="flex gap-0.5">
          <span className="w-1 h-1 bg-[var(--color-primary)] rounded-full animate-bounce" style={{ animationDelay: '0ms' }} />
          <span className="w-1 h-1 bg-[var(--color-primary)] rounded-full animate-bounce" style={{ animationDelay: '100ms' }} />
          <span className="w-1 h-1 bg-[var(--color-primary)] rounded-full animate-bounce" style={{ animationDelay: '200ms' }} />
        </span>
      )}
      {latestEvent && (
        <span className="truncate max-w-xs">
          {latestEvent.message || latestEvent.type}
        </span>
      )}
    </div>
  );
}

