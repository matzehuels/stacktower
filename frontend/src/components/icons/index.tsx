/**
 * Custom icon components for the application.
 * 
 * Note: For standard icons, use Lucide React directly:
 *   import { Github, Package, Clock } from 'lucide-react';
 * 
 * This file contains only custom/branded icons.
 */

// Re-export language icons
export {
  LanguageIcon,
  LANGUAGE_ICONS,
  PythonIcon,
  JavaScriptIcon,
  TypeScriptIcon,
  RustIcon,
  GoIcon,
  RubyIcon,
  PhpIcon,
  JavaIcon,
} from './LanguageIcons';

// =============================================================================
// Logo
// =============================================================================

export function StacktowerLogo({ className = 'w-8 h-8' }: { className?: string }) {
  return (
    <div className={`${className} bg-primary rounded-lg flex items-center justify-center`}>
      <svg className="w-5 h-5 text-primary-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5">
        <rect x="3" y="14" width="6" height="7" rx="1" />
        <rect x="9" y="8" width="6" height="13" rx="1" />
        <rect x="15" y="3" width="6" height="18" rx="1" />
      </svg>
    </div>
  );
}

export function StacktowerLogoLarge({ className = 'w-16 h-16' }: { className?: string }) {
  return (
    <div className={`${className} bg-primary rounded-2xl flex items-center justify-center`}>
      <svg className="w-9 h-9 text-primary-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5">
        <rect x="3" y="14" width="6" height="7" rx="1" />
        <rect x="9" y="8" width="6" height="13" rx="1" />
        <rect x="15" y="3" width="6" height="18" rx="1" />
      </svg>
    </div>
  );
}
