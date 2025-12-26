import { useEffect, useRef } from 'react';
import ReactMarkdown from 'react-markdown';
import { useAgent } from '../context/AgentContext';
import type { AgentEvent } from '../types/api';

export function AgentSidebar() {
  const { events, isConnected, isVisible, toggleVisible, clear } = useAgent();
  const scrollRef = useRef<HTMLDivElement>(null);

  // Auto-scroll to bottom on new events
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [events]);

  if (!isVisible) {
    return null;
  }

  return (
    <aside className="w-80 bg-[var(--color-bg-elevated)] flex flex-col border-l border-[var(--color-border)]">
      {/* Header */}
      <div className="h-12 flex items-center justify-between px-4 border-b border-[var(--color-border)]">
        <div className="flex items-center gap-2">
          <svg className="w-4 h-4 text-[var(--color-primary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
          </svg>
          <span className="text-sm font-medium text-[var(--color-text)]">Agent</span>
          {isConnected && (
            <span className="flex items-center gap-1 text-[10px] text-[var(--color-success)]">
              <span className="w-1.5 h-1.5 bg-[var(--color-success)] rounded-full animate-pulse" />
              Live
            </span>
          )}
        </div>
        <div className="flex items-center gap-1">
          {events.length > 0 && (
            <button
              onClick={clear}
              className="p-1.5 text-[var(--color-text-muted)] hover:text-[var(--color-text)] hover:bg-[var(--color-bg-hover)] rounded transition-colors"
              title="Clear"
            >
              <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
            </button>
          )}
          <button
            onClick={toggleVisible}
            className="p-1.5 text-[var(--color-text-muted)] hover:text-[var(--color-text)] hover:bg-[var(--color-bg-hover)] rounded transition-colors"
            title="Close"
          >
            <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
      </div>

      {/* Event count */}
      <div className="px-4 py-2 border-b border-[var(--color-border)] text-[10px] text-[var(--color-text-muted)] uppercase tracking-wider">
        Activity · {events.length}
      </div>

      {/* Event stream */}
      <div 
        ref={scrollRef}
        className="flex-1 overflow-y-auto"
      >
        {events.length === 0 ? (
          <div className="h-full flex flex-col items-center justify-center p-6">
            <svg className="w-10 h-10 text-[var(--color-text-muted)] opacity-30 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
            </svg>
            <p className="text-xs text-[var(--color-text-muted)] text-center">
              No activity yet
            </p>
            <p className="text-[10px] text-[var(--color-text-muted)] text-center mt-1 opacity-60">
              Click "AI Analyze" to start
            </p>
          </div>
        ) : (
          <div className="divide-y divide-[var(--color-border)]">
            {events.map((event, i) => (
              <AgentEventItem key={i} event={event} />
            ))}
            
            {isConnected && (
              <div className="flex items-center gap-3 px-4 py-3">
                <div className="flex gap-1">
                  <span className="w-1.5 h-1.5 bg-[var(--color-primary)] rounded-full animate-bounce" style={{ animationDelay: '0ms' }} />
                  <span className="w-1.5 h-1.5 bg-[var(--color-primary)] rounded-full animate-bounce" style={{ animationDelay: '150ms' }} />
                  <span className="w-1.5 h-1.5 bg-[var(--color-primary)] rounded-full animate-bounce" style={{ animationDelay: '300ms' }} />
                </div>
                <span className="text-xs text-[var(--color-text-muted)]">Thinking...</span>
              </div>
            )}
          </div>
        )}
      </div>
    </aside>
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
          <svg className="w-4 h-4 text-amber-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
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
          <svg className="w-4 h-4 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
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
        return (
          <svg className="w-4 h-4 text-[var(--color-text-muted)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <circle cx="12" cy="12" r="3" />
          </svg>
        );
    }
  };

  const formatTime = (timestamp: string) => {
    try {
      const date = new Date(timestamp);
      if (isNaN(date.getTime())) return '';
      return date.toLocaleTimeString('en-US', { 
        hour12: false, 
        hour: '2-digit', 
        minute: '2-digit', 
        second: '2-digit' 
      });
    } catch {
      return '';
    }
  };

  return (
    <div className="group px-4 py-3 hover:bg-[var(--color-bg-hover)] transition-colors">
      <div className="flex items-start gap-3">
        <div className="flex-shrink-0 mt-0.5">{getIcon()}</div>
        <div className="flex-1 min-w-0">
          {/* Markdown rendered message */}
          <div className="agent-message text-xs text-[var(--color-text-secondary)] leading-relaxed">
            <ReactMarkdown
              components={{
                // Customize rendering for better styling
                p: ({ children }) => <p className="mb-2 last:mb-0">{children}</p>,
                strong: ({ children }) => <strong className="font-semibold text-[var(--color-text)]">{children}</strong>,
                em: ({ children }) => <em className="italic">{children}</em>,
                code: ({ children }) => (
                  <code className="px-1 py-0.5 bg-[var(--color-bg)] rounded text-[10px] font-mono text-[var(--color-primary)]">
                    {children}
                  </code>
                ),
                pre: ({ children }) => (
                  <pre className="my-2 p-2 bg-[var(--color-bg)] rounded text-[10px] font-mono overflow-x-auto">
                    {children}
                  </pre>
                ),
                ul: ({ children }) => <ul className="list-disc list-inside my-1 space-y-0.5">{children}</ul>,
                ol: ({ children }) => <ol className="list-decimal list-inside my-1 space-y-0.5">{children}</ol>,
                li: ({ children }) => <li className="text-[var(--color-text-secondary)]">{children}</li>,
                h1: ({ children }) => <h1 className="text-sm font-semibold text-[var(--color-text)] mt-2 mb-1">{children}</h1>,
                h2: ({ children }) => <h2 className="text-xs font-semibold text-[var(--color-text)] mt-2 mb-1">{children}</h2>,
                h3: ({ children }) => <h3 className="text-xs font-medium text-[var(--color-text)] mt-1 mb-0.5">{children}</h3>,
                a: ({ href, children }) => (
                  <a href={href} target="_blank" rel="noopener noreferrer" className="text-[var(--color-primary)] hover:underline">
                    {children}
                  </a>
                ),
                blockquote: ({ children }) => (
                  <blockquote className="border-l-2 border-[var(--color-border)] pl-2 my-1 text-[var(--color-text-muted)] italic">
                    {children}
                  </blockquote>
                ),
              }}
            >
              {event.message || event.type}
            </ReactMarkdown>
          </div>
          
          {/* Tool name tag */}
          {event.tool_name && (
            <div className="mt-1.5">
              <span className="inline-flex items-center px-1.5 py-0.5 bg-[var(--color-bg)] rounded text-[10px] font-mono text-[var(--color-text-muted)]">
                {event.tool_name}
              </span>
            </div>
          )}
        </div>
        
        {/* Timestamp */}
        <span className="text-[10px] text-[var(--color-text-muted)] opacity-0 group-hover:opacity-100 transition-opacity flex-shrink-0">
          {formatTime(event.timestamp)}
        </span>
      </div>
    </div>
  );
}
