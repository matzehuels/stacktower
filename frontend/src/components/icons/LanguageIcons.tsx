/**
 * Programming language icons using official SVG logos.
 */

import { cn } from '@/lib/utils';

// Import official SVG logos
import pythonLogo from '@/assets/python.svg';
import javascriptLogo from '@/assets/javascript.svg';
import typescriptLogo from '@/assets/typescript.svg';
import rustLogo from '@/assets/rust.svg';
import goLogo from '@/assets/go.svg';
import rubyLogo from '@/assets/ruby.svg';
import phpLogo from '@/assets/php.svg';
import javaLogo from '@/assets/java.svg';

interface IconProps {
  className?: string;
}

// Python
export function PythonIcon({ className }: IconProps) {
  return <img src={pythonLogo} alt="Python" className={cn('h-4 w-4', className)} />;
}

// JavaScript
export function JavaScriptIcon({ className }: IconProps) {
  return <img src={javascriptLogo} alt="JavaScript" className={cn('h-4 w-4', className)} />;
}

// TypeScript
export function TypeScriptIcon({ className }: IconProps) {
  return <img src={typescriptLogo} alt="TypeScript" className={cn('h-4 w-4', className)} />;
}

// Rust
export function RustIcon({ className }: IconProps) {
  return <img src={rustLogo} alt="Rust" className={cn('h-4 w-4', className)} />;
}

// Go
export function GoIcon({ className }: IconProps) {
  return <img src={goLogo} alt="Go" className={cn('h-4 w-4', className)} />;
}

// Ruby
export function RubyIcon({ className }: IconProps) {
  return <img src={rubyLogo} alt="Ruby" className={cn('h-4 w-4', className)} />;
}

// PHP
export function PhpIcon({ className }: IconProps) {
  return <img src={phpLogo} alt="PHP" className={cn('h-4 w-4', className)} />;
}

// Java
export function JavaIcon({ className }: IconProps) {
  return <img src={javaLogo} alt="Java" className={cn('h-4 w-4', className)} />;
}

// Map language values to icon components
export const LANGUAGE_ICONS: Record<string, React.ComponentType<IconProps>> = {
  python: PythonIcon,
  javascript: JavaScriptIcon,
  typescript: TypeScriptIcon,
  rust: RustIcon,
  go: GoIcon,
  ruby: RubyIcon,
  php: PhpIcon,
  java: JavaIcon,
};

// Helper component to render the right icon for a language
export function LanguageIcon({ language, className }: { language: string; className?: string }) {
  const Icon = LANGUAGE_ICONS[language];
  if (!Icon) return null;
  return <Icon className={className} />;
}
