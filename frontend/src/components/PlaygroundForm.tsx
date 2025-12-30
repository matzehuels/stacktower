/**
 * Form for submitting package visualization requests.
 */

import { useState, useMemo, useCallback } from 'react';
import { Zap, ChevronRight } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Combobox, type ComboboxOption } from '@/components/ui/combobox';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { LanguageIcon } from '@/components/icons';
import { usePackageSuggestions } from '@/hooks/queries';
import type { RenderRequest } from '@/types/api';
import type { Language } from '@/config/constants';
import { LANGUAGES } from '@/config/constants';
import { cn } from '@/lib/utils';

// Simple debounce hook
function useDebounce<T>(value: T, delay: number): T {
  const [debouncedValue, setDebouncedValue] = useState(value);

  useMemo(() => {
    const timer = setTimeout(() => setDebouncedValue(value), delay);
    return () => clearTimeout(timer);
  }, [value, delay]);

  return debouncedValue;
}

interface Props {
  onSubmit: (request: RenderRequest) => void;
  isLoading: boolean;
}

export function PlaygroundForm({ onSubmit, isLoading }: Props) {
  const [language, setLanguage] = useState<Language>('python');
  const [packageName, setPackageName] = useState('');
  const [searchQuery, setSearchQuery] = useState('');
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [maxDepth, setMaxDepth] = useState(10);
  const [maxNodes, setMaxNodes] = useState(500);

  const selectedLang = LANGUAGES.find(l => l.value === language);

  // Debounce search query for API calls
  const debouncedQuery = useDebounce(searchQuery, 200);
  
  // Fetch package suggestions
  const { data: suggestions = [], isLoading: suggestionsLoading } = usePackageSuggestions(
    language,
    debouncedQuery
  );

  // Convert suggestions to combobox options
  const options: ComboboxOption[] = useMemo(() => {
    return suggestions.map((s) => ({
      value: s.package,
      label: s.package,
      secondary: s.popularity > 0 ? `${s.popularity} saved` : undefined,
    }));
  }, [suggestions]);

  const handleInputChange = useCallback((value: string) => {
    setSearchQuery(value);
  }, []);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!packageName.trim()) return;

    onSubmit({
      language,
      package: packageName.trim(),
      formats: ['svg', 'png', 'pdf', 'json'],
      viz_type: 'tower', // Default to tower, user can switch in visualization view
      max_depth: maxDepth,
      max_nodes: maxNodes,
      merge: true,
    });
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      {/* Main inputs */}
      <div className="flex gap-3">
        {/* Language selector */}
        <div className="w-48">
          <label className="block text-xs font-medium text-muted-foreground mb-2">
            Registry
          </label>
          <Select
            value={language}
            onValueChange={(value) => setLanguage(value as Language)}
            disabled={isLoading}
          >
            <SelectTrigger>
              <SelectValue>
                <span className="flex items-center gap-2">
                  <LanguageIcon language={language} className="h-4 w-4" />
                  <span className="font-mono">{LANGUAGES.find(l => l.value === language)?.label.split(' (')[0]}</span>
                </span>
              </SelectValue>
            </SelectTrigger>
            <SelectContent>
            {LANGUAGES.map(lang => (
                <SelectItem key={lang.value} value={lang.value}>
                  <span className="flex items-center gap-2">
                    <LanguageIcon language={lang.value} className="h-4 w-4" />
                    <span className="font-mono">{lang.label.split(' (')[0]}</span>
                  </span>
                </SelectItem>
            ))}
            </SelectContent>
          </Select>
        </div>

        {/* Package name input with autocomplete */}
        <div className="flex-1">
          <label className="block text-xs font-medium text-muted-foreground mb-2">
            Package
          </label>
          <Combobox
            value={packageName}
            onChange={setPackageName}
            options={options}
            placeholder={selectedLang?.placeholder}
            disabled={isLoading}
            loading={suggestionsLoading}
            onInputChange={handleInputChange}
          />
        </div>

        {/* Submit button */}
        <div className="flex items-end">
          <Button
            type="submit"
            disabled={isLoading || !packageName.trim()}
            size="lg"
          >
            {isLoading ? (
              <>
                <span className="h-4 w-4 border-2 border-current border-t-transparent rounded-full animate-spin mr-2" />
                Analyzing
              </>
            ) : (
              <>
                <Zap className="h-4 w-4 mr-2" />
                Generate
              </>
            )}
          </Button>
        </div>
      </div>

      {/* Advanced options toggle */}
      <div className="pt-2 border-t border-border">
        <button
          type="button"
          onClick={() => setShowAdvanced(!showAdvanced)}
          className="flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors"
        >
          <ChevronRight className={cn('h-3 w-3 transition-transform', showAdvanced && 'rotate-90')} />
          <span>Advanced options</span>
        </button>

        {showAdvanced && (
          <div className="flex gap-4 mt-4 animate-in fade-in slide-in-from-top-2 duration-200">
            {/* Max depth */}
            <div className="w-24">
              <label className="block text-xs font-medium text-muted-foreground mb-2">
                Max Depth
              </label>
              <Input
                type="number"
                min={1}
                max={20}
                value={maxDepth}
                onChange={e => setMaxDepth(Number(e.target.value))}
                disabled={isLoading}
                className="h-9 font-mono"
              />
            </div>

            {/* Max nodes */}
            <div className="w-24">
              <label className="block text-xs font-medium text-muted-foreground mb-2">
                Max Nodes
              </label>
              <Input
                type="number"
                min={10}
                max={5000}
                step={10}
                value={maxNodes}
                onChange={e => setMaxNodes(Number(e.target.value))}
                disabled={isLoading}
                className="h-9 font-mono"
              />
            </div>
          </div>
        )}
      </div>
    </form>
  );
}
