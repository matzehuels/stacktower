import { useState } from 'react';
import type { Language, VizType, VisualizeRequest } from '../types/api';
import { LANGUAGES } from '../types/api';

interface Props {
  onSubmit: (request: VisualizeRequest) => void;
  isLoading: boolean;
}

export function PlaygroundForm({ onSubmit, isLoading }: Props) {
  const [language, setLanguage] = useState<Language>('python');
  const [packageName, setPackageName] = useState('');
  const [vizType, setVizType] = useState<VizType>('tower');
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [maxDepth, setMaxDepth] = useState(10);
  const [maxNodes, setMaxNodes] = useState(500);

  const selectedLang = LANGUAGES.find(l => l.value === language);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!packageName.trim()) return;

    onSubmit({
      language,
      package: packageName.trim(),
      formats: ['svg', 'png', 'pdf'],
      viz_type: vizType,
      max_depth: maxDepth,
      max_nodes: maxNodes,
    });
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-3">
      <div className="flex gap-3">
        {/* Language selector */}
        <div className="w-40">
          <label className="block text-[10px] font-medium text-[var(--color-text-muted)] uppercase tracking-wider mb-1.5">
            Registry
          </label>
          <select
            value={language}
            onChange={e => setLanguage(e.target.value as Language)}
            disabled={isLoading}
            className="select font-mono"
          >
            {LANGUAGES.map(lang => (
              <option key={lang.value} value={lang.value}>
                {lang.label.split(' ')[0]}
              </option>
            ))}
          </select>
        </div>

        {/* Package name input */}
        <div className="flex-1">
          <label className="block text-[10px] font-medium text-[var(--color-text-muted)] uppercase tracking-wider mb-1.5">
            Package
          </label>
          <input
            type="text"
            value={packageName}
            onChange={e => setPackageName(e.target.value)}
            placeholder={selectedLang?.placeholder}
            disabled={isLoading}
            className="input font-mono"
          />
        </div>

        {/* Viz type */}
        <div className="w-28">
          <label className="block text-[10px] font-medium text-[var(--color-text-muted)] uppercase tracking-wider mb-1.5">
            Style
          </label>
          <select
            value={vizType}
            onChange={e => setVizType(e.target.value as VizType)}
            disabled={isLoading}
            className="select"
          >
            <option value="tower">Tower</option>
            <option value="nodelink">Graph</option>
          </select>
        </div>

        {/* Submit */}
        <div className="flex items-end">
          <button
            type="submit"
            disabled={isLoading || !packageName.trim()}
            className="btn btn-primary h-[34px]"
          >
            {isLoading ? (
              <>
                <svg className="w-3.5 h-3.5 animate-spin" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                </svg>
                <span>Analyzing</span>
              </>
            ) : (
              <>
                <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
                </svg>
                <span>Analyze</span>
              </>
            )}
          </button>
        </div>
      </div>

      {/* Advanced options toggle */}
      <button
        type="button"
        onClick={() => setShowAdvanced(!showAdvanced)}
        className="text-[11px] text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] transition-colors flex items-center gap-1"
      >
        <svg className={`w-3 h-3 transition-transform ${showAdvanced ? 'rotate-90' : ''}`} fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9 5l7 7-7 7" />
        </svg>
        Advanced
      </button>
      
      {showAdvanced && (
        <div className="flex gap-3 pt-2 border-t border-[var(--color-border)] animate-fade-in">
          <div className="w-32">
            <label className="block text-[10px] font-medium text-[var(--color-text-muted)] uppercase tracking-wider mb-1.5">
              Max Depth
            </label>
            <input
              type="number"
              min={1}
              max={20}
              value={maxDepth}
              onChange={e => setMaxDepth(Number(e.target.value))}
              disabled={isLoading}
              className="input font-mono"
            />
          </div>
          <div className="w-32">
            <label className="block text-[10px] font-medium text-[var(--color-text-muted)] uppercase tracking-wider mb-1.5">
              Max Nodes
            </label>
            <input
              type="number"
              min={10}
              max={5000}
              step={10}
              value={maxNodes}
              onChange={e => setMaxNodes(Number(e.target.value))}
              disabled={isLoading}
              className="input font-mono"
            />
          </div>
        </div>
      )}
    </form>
  );
}
