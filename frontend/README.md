# Stacktower Frontend

React + TypeScript + Vite frontend for the Stacktower dependency visualization tool.

Built with **shadcn/ui** components and **Tailwind CSS v4**.

## Features

- 🌗 **Dark/Light Mode** - System preference detection with manual toggle
- 🎨 **shadcn/ui** - Beautiful, accessible components built on Radix UI
- ⚡ **React Query** - Automatic caching and background refetching
- 🔒 **GitHub OAuth** - Secure authentication

## Architecture Overview

```
src/
├── components/           # UI components
│   ├── ui/              # shadcn/ui primitives
│   ├── icons/           # Custom brand icons
│   ├── layout/          # Page layouts (Sidebar, LoginScreen)
│   └── *.tsx            # Feature components
├── config/              # Configuration
│   ├── constants.ts     # App-wide constants
│   └── env.ts           # Environment variables
├── hooks/               # React hooks
│   ├── queries/         # React Query hooks
│   └── useTheme.ts      # Theme hook
├── lib/                 # Non-React utilities
│   ├── api/             # API client and endpoints
│   └── utils.ts         # shadcn utilities (cn)
├── providers/           # React context providers
├── types/               # TypeScript type definitions
└── App.tsx              # Root component
```

## Key Patterns

### Importing

Use path aliases for clean imports:

```tsx
// UI components (shadcn/ui)
import { Button, Card, Input, Select, Skeleton } from '@/components/ui';

// Data fetching hooks
import { useCurrentUser, useHistory, useRenderMutation } from '@/hooks';

// Theme
import { useTheme } from '@/hooks';

// Icons (use Lucide for standard icons)
import { Github, Package, Clock } from 'lucide-react';

// Custom icons
import { StacktowerLogo } from '@/components/icons';
```

### UI Components (shadcn/ui)

```tsx
import { Button, Input, Card, CardContent, Skeleton } from '@/components/ui';
import { AlertDialog, AlertDialogAction, AlertDialogContent } from '@/components/ui';

// Buttons
<Button>Primary</Button>
<Button variant="secondary">Secondary</Button>
<Button variant="outline">Outline</Button>
<Button variant="destructive">Destructive</Button>
<Button variant="ghost" size="icon"><Github /></Button>

// Cards
<Card>
  <CardHeader>
    <CardTitle>Title</CardTitle>
    <CardDescription>Description</CardDescription>
  </CardHeader>
  <CardContent>Content</CardContent>
</Card>

// Loading states
<Skeleton className="h-4 w-32" />

// Dialogs (instead of confirm())
<AlertDialog>
  <AlertDialogTrigger asChild>
    <Button>Delete</Button>
  </AlertDialogTrigger>
  <AlertDialogContent>...</AlertDialogContent>
</AlertDialog>
```

### Theme Toggle

The app supports dark/light/system themes:

```tsx
import { useTheme } from '@/hooks';
import { ThemeToggle } from '@/components/ui';

// Use the pre-built toggle
<ThemeToggle />

// Or build your own
const { theme, setTheme, resolvedTheme } = useTheme();
setTheme('dark');  // 'dark' | 'light' | 'system'
```

### Toast Notifications (instead of alert())

```tsx
import { toast } from 'sonner';

toast.success('Saved successfully');
toast.error('Something went wrong', { description: error.message });
toast.loading('Processing...');
```

### Data Fetching

```tsx
import { useRepos, useHistory, useRenderMutation } from '@/hooks';

function MyComponent() {
  // Queries (automatic fetching)
  const { data: repos, isLoading, error } = useRepos();
  
  // Mutations (on-demand)
  const { render, isLoading, job } = useRenderMutation();
  
  const handleSubmit = () => {
    render({ package: 'flask', language: 'python' });
  };
}
```

## Development

```bash
# Install dependencies
npm install

# Start dev server (port 3000)
npm run dev

# Type check
npx tsc --noEmit

# Lint
npm run lint

# Build for production
npm run build
```

## Adding shadcn Components

```bash
# Add a new component
npx shadcn@latest add dialog

# List available components
npx shadcn@latest add
```

## Environment Variables

Create a `.env.local` file for local overrides:

```bash
# API base URL (optional, defaults to /api/v1 which is proxied)
VITE_API_BASE_URL=/api/v1
```

## Adding New Features

### New API Endpoint

1. Add types to `types/api.ts`
2. Add endpoint function to `lib/api/endpoints.ts`
3. Export from `lib/api/index.ts`
4. Create React Query hook in `hooks/queries/`
5. Export from `hooks/queries/index.ts`

### New UI Component

Use shadcn/ui components where possible:
```bash
npx shadcn@latest add [component-name]
```

For custom components:
1. Create in `components/`
2. Export from `components/index.ts`
3. Use shadcn primitives and `cn()` utility

## Tech Stack

- **React 19** - UI framework
- **TypeScript** - Type safety
- **Vite** - Build tool
- **TanStack Query** - Server state management
- **Tailwind CSS v4** - Styling
- **shadcn/ui** - Component library (Radix UI + Tailwind)
- **Lucide React** - Icons
- **Sonner** - Toast notifications
