/**
 * Reusable package search bar with autocomplete suggestions.
 * Used in PackagesView and TowerExplorer.
 */

import { useState, useMemo, useCallback } from 'react';
import { Search } from 'lucide-react';
import { Combobox, type ComboboxOption } from '@/components/ui/combobox';
import { LanguageIcon } from '@/components/icons';
import { usePackageSuggestions } from '@/hooks/queries';
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

interface PackageSearchBarProps {
  /** Called when a package is selected from suggestions */
  onSelect: (language: Language, packageName: string) => void;
  /** Current language filter (optional, shows language selector if not provided) */
  language?: Language | 'all';
  /** Called when language changes */
  onLanguageChange?: (language: Language | 'all') => void;
  /** Placeholder text */
  placeholder?: string;
  /** Disabled state */
  disabled?: boolean;
  /** Show language icons as buttons instead of dropdown */
  compactLanguageFilter?: boolean;
  /** Additional class for the container */
  className?: string;
}

export function PackageSearchBar({
  onSelect,
  language = 'all',
  onLanguageChange,
  placeholder = 'Search packages...',
  disabled = false,
  compactLanguageFilter = false,
  className,
}: PackageSearchBarProps) {
  const [searchQuery, setSearchQuery] = useState('');
  const [inputValue, setInputValue] = useState('');
  
  // For suggestions, use a specific language or default to python
  const suggestionsLanguage = language === 'all' ? 'python' : language;
  
  // Debounce search query for API calls
  const debouncedQuery = useDebounce(searchQuery, 200);
  
  // Fetch package suggestions
  const { data: suggestions = [], isLoading: suggestionsLoading } = usePackageSuggestions(
    suggestionsLanguage,
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

  const handleChange = useCallback((value: string) => {
    setInputValue(value);
    // Check if this value matches an option (user selected from dropdown)
    const matchedOption = options.find(opt => opt.value === value);
    if (matchedOption) {
      const selectedLanguage = language === 'all' ? suggestionsLanguage : language;
      onSelect(selectedLanguage, value);
      // Clear input after selection
      setInputValue('');
      setSearchQuery('');
    }
  }, [options, language, suggestionsLanguage, onSelect]);

  return (
    <div className={cn('flex gap-2 items-center', className)}>
      {/* Language filter - compact icon buttons */}
      {compactLanguageFilter && onLanguageChange && (
        <div className="flex gap-1 p-1 bg-muted rounded-lg shrink-0">
          <button
            onClick={() => onLanguageChange('all')}
            className={cn(
              'px-2 py-1.5 text-xs font-medium rounded-md transition-colors whitespace-nowrap',
              language === 'all'
                ? 'bg-background text-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground'
            )}
          >
            All
          </button>
          {LANGUAGES.map((lang) => (
            <button
              key={lang.value}
              onClick={() => onLanguageChange(lang.value)}
              className={cn(
                'px-1.5 py-1.5 rounded-md transition-colors whitespace-nowrap flex items-center',
                language === lang.value
                  ? 'bg-background shadow-sm'
                  : 'text-muted-foreground hover:text-foreground'
              )}
              title={lang.label}
            >
              <LanguageIcon language={lang.value} className="h-4 w-4" />
            </button>
          ))}
        </div>
      )}

      {/* Search input with autocomplete */}
      <div className="relative flex-1 min-w-0">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground pointer-events-none z-10" />
        <Combobox
          value={inputValue}
          onChange={handleChange}
          options={options}
          placeholder={placeholder}
          disabled={disabled}
          loading={suggestionsLoading}
          onInputChange={handleInputChange}
          className="w-full [&_input]:pl-9"
        />
      </div>
    </div>
  );
}

