/**
 * Main packages view for entering package names to visualize.
 */

import { useState } from 'react';
import { ArrowRight, ChevronRight } from 'lucide-react';
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
import { useDebounce } from '@/hooks';
import type { RenderRequest } from '@/types/api';
import type { Language } from '@/config/constants';
import { LANGUAGES, DEFAULT_FORMATS } from '@/config/constants';
import { cn } from '@/lib/utils';

const EXAMPLES_BY_LANGUAGE: Record<string, string[]> = {
  python: ['fastapi', 'openai', 'pydantic', 'requests', 'typer', 'flask', 'django', 'numpy', 'pandas'],
  javascript: ['express', 'ioredis', 'knex', 'mongoose', 'pino', 'yup', 'lodash', 'axios', 'zod'],
  rust: ['serde', 'diesel', 'hyper', 'rayon', 'ureq', 'tokio', 'actix-web', 'clap'],
  go: ['github.com/gin-gonic/gin', 'github.com/gofiber/fiber/v2', 'github.com/labstack/echo/v4', 'github.com/spf13/cobra', 'github.com/spf13/viper', 'gorm.io/gorm'],
  ruby: ['rails', 'sinatra', 'devise', 'rspec', 'sidekiq'],
  php: ['laravel/framework', 'symfony/symfony', 'guzzlehttp/guzzle', 'monolog/monolog', 'phpunit/phpunit'],
  java: ['org.springframework:spring-core', 'com.google.guava:guava', 'org.apache.commons:commons-lang3', 'junit:junit', 'org.hibernate:hibernate-core'],
};

interface PackagesViewProps {
  onSubmit: (request: RenderRequest) => void;
  isLoading: boolean;
}

export function PackagesView({ onSubmit, isLoading }: PackagesViewProps) {
  const [language, setLanguage] = useState<Language>('python');
  const [packageName, setPackageName] = useState('');
  const [searchQuery, setSearchQuery] = useState('');
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [maxDepth, setMaxDepth] = useState(10);
  const [maxNodes, setMaxNodes] = useState(500);

  const selectedLang = LANGUAGES.find(l => l.value === language);
  const debouncedQuery = useDebounce(searchQuery, 200);
  
  const { data: suggestions = [], isLoading: suggestionsLoading } = usePackageSuggestions(
    language,
    debouncedQuery
  );

  const options: ComboboxOption[] = suggestions.map((s) => ({
    value: s.package,
    label: s.package,
    secondary: s.popularity > 0 ? `${s.popularity} saved` : undefined,
  }));

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!packageName.trim()) return;

    onSubmit({
      language,
      package: packageName.trim(),
      formats: DEFAULT_FORMATS as ('svg' | 'png' | 'pdf' | 'json')[],
      viz_type: 'tower',
      max_depth: maxDepth,
      max_nodes: maxNodes,
      merge: true,
    });
  };

  const handleExampleClick = (lang: Language, pkg: string) => {
    onSubmit({
      language: lang,
      package: pkg,
      formats: DEFAULT_FORMATS as ('svg' | 'png' | 'pdf' | 'json')[],
      merge: true,
    });
  };

  return (
    <div className="flex-1 overflow-y-auto">
      {/* Hero search section */}
      <div className="border-b bg-muted/30 px-6 py-12">
        <div className="max-w-2xl mx-auto">
          <h1 className="text-2xl font-semibold text-center mb-2">
            Visualize any package
          </h1>
          <p className="text-sm text-muted-foreground text-center mb-8">
            Enter a package name to generate its dependency tower
          </p>

          {/* Search form */}
          <form onSubmit={handleSubmit} className="space-y-3">
            <div className="flex gap-2">
              {/* Language selector */}
              <Select
                value={language}
                onValueChange={(value) => setLanguage(value as Language)}
                disabled={isLoading}
              >
                <SelectTrigger className="w-[140px] bg-background">
                  <SelectValue>
                    <span className="flex items-center gap-2">
                      <LanguageIcon language={language} className="h-4 w-4" />
                      <span>{LANGUAGES.find(l => l.value === language)?.label.split(' (')[0]}</span>
                    </span>
                  </SelectValue>
                </SelectTrigger>
                <SelectContent>
                  {LANGUAGES.map(lang => (
                    <SelectItem key={lang.value} value={lang.value}>
                      <span className="flex items-center gap-2">
                        <LanguageIcon language={lang.value} className="h-4 w-4" />
                        <span>{lang.label.split(' (')[0]}</span>
                      </span>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>

              {/* Package search */}
              <div className="flex-1">
                <Combobox
                  value={packageName}
                  onChange={setPackageName}
                  options={options}
                  placeholder={selectedLang?.placeholder}
                  disabled={isLoading}
                  loading={suggestionsLoading}
                  onInputChange={setSearchQuery}
                />
              </div>

              {/* Submit */}
              <Button
                type="submit"
                disabled={isLoading || !packageName.trim()}
                className="gap-2"
              >
                {isLoading ? (
                  <>
                    <span className="h-4 w-4 border-2 border-current border-t-transparent rounded-full animate-spin" />
                    Analyzing
                  </>
                ) : (
                  <>
                    Generate
                    <ArrowRight className="h-4 w-4" />
                  </>
                )}
              </Button>
            </div>

            {/* Advanced options */}
            <div className="flex justify-center">
              <button
                type="button"
                onClick={() => setShowAdvanced(!showAdvanced)}
                className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
              >
                <ChevronRight className={cn('h-3 w-3 transition-transform', showAdvanced && 'rotate-90')} />
                Advanced options
              </button>
            </div>

            {showAdvanced && (
              <div className="flex justify-center gap-4 animate-in fade-in slide-in-from-top-2 duration-200">
                <div className="flex items-center gap-2">
                  <label className="text-xs text-muted-foreground">Max Depth</label>
                  <Input
                    type="number"
                    min={1}
                    max={20}
                    value={maxDepth}
                    onChange={e => setMaxDepth(Number(e.target.value))}
                    disabled={isLoading}
                    className="h-8 w-20 text-xs"
                  />
                </div>
                <div className="flex items-center gap-2">
                  <label className="text-xs text-muted-foreground">Max Nodes</label>
                  <Input
                    type="number"
                    min={10}
                    max={5000}
                    step={10}
                    value={maxNodes}
                    onChange={e => setMaxNodes(Number(e.target.value))}
                    disabled={isLoading}
                    className="h-8 w-20 text-xs"
                  />
                </div>
              </div>
            )}
          </form>
        </div>
      </div>

      {/* Examples by language */}
      <div className="px-6 py-8">
        <div className="max-w-3xl mx-auto">
          <h2 className="text-sm font-medium text-muted-foreground mb-6 text-center">
            Or try one of these examples
          </h2>
          
          <div className="space-y-6">
            {Object.entries(EXAMPLES_BY_LANGUAGE).map(([lang, packages]) => (
              <div key={lang} className="flex items-start gap-4">
                <div className="flex items-center gap-2 w-28 shrink-0 pt-1">
                  <LanguageIcon language={lang} className="h-4 w-4" />
                  <span className="text-sm font-medium capitalize">{lang}</span>
                </div>
                
                <div className="flex flex-wrap gap-2">
                  {packages.map((pkg) => (
                    <button
                      key={pkg}
                      onClick={() => handleExampleClick(lang as Language, pkg)}
                      disabled={isLoading}
                      className="px-2.5 py-1 text-xs font-mono rounded-md border bg-background hover:bg-muted transition-colors disabled:opacity-50"
                    >
                      {pkg}
                    </button>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
