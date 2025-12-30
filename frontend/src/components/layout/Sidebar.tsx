/**
 * Application sidebar with navigation and user info.
 * Clean, minimal design.
 */

import { useState } from 'react';
import { Package, Github, Library, Compass, LogOut, Sun, Moon, Monitor, Lock, Layers } from 'lucide-react';
import type { GitHubUser } from '@/types/api';
import { GitHubLoginButton } from '@/components/GitHubLoginButton';
import { Button } from '@/components/ui/button';
import { CollapseButton } from '@/components/ui/collapse-button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
  DropdownMenuSub,
  DropdownMenuSubTrigger,
  DropdownMenuSubContent,
  DropdownMenuPortal,
} from '@/components/ui/dropdown-menu';
import { useTheme } from '@/hooks/useTheme';
import { cn } from '@/lib/utils';

export type Tab = 'packages' | 'repos' | 'library' | 'explore';

interface SidebarProps {
  user: GitHubUser | null | undefined;
  logout: () => void;
  login: () => void;
  activeTab: Tab;
  onNavigate: (tab: Tab) => void;
}

export function Sidebar({ user, logout, login, activeTab, onNavigate }: SidebarProps) {
  const [collapsed, setCollapsed] = useState(false);
  const { theme, setTheme } = useTheme();
  const isAuthenticated = !!user;

  return (
    <aside 
      className={cn(
        'bg-sidebar border-r border-sidebar-border flex flex-col transition-all duration-200 flex-shrink-0 relative',
        collapsed ? 'w-14' : 'w-52'
      )}
    >
      {/* Header with logo */}
      <div className={cn(
        'h-14 flex items-center border-b border-sidebar-border',
        collapsed ? 'justify-center px-2' : 'gap-2 px-3'
      )}>
        <Layers className="w-5 h-5 shrink-0" />
        {!collapsed && (
          <span className="text-sm font-semibold">Stacktower</span>
        )}
      </div>

      {/* Collapse toggle */}
      <CollapseButton
        onClick={() => setCollapsed(!collapsed)}
        collapsed={!collapsed}
        position="-right-3"
        top="top-16"
        title={collapsed ? 'Expand' : 'Collapse'}
      />

      {/* Navigation */}
      <nav className="flex-1 p-1.5 space-y-0.5">
        {/* Explore - always available */}
        <NavItem
          active={activeTab === 'explore'}
          onClick={() => onNavigate('explore')}
          icon={<Compass className="h-4 w-4" />}
          collapsed={collapsed}
        >
          Explore
        </NavItem>

        {/* Divider */}
        {!collapsed && (
          <div className="py-1.5">
            <div className="border-t border-sidebar-border" />
          </div>
        )}

        {/* Auth-required features */}
        <NavItem
          active={activeTab === 'packages'}
          onClick={() => onNavigate('packages')}
          icon={<Package className="h-4 w-4" />}
          collapsed={collapsed}
          locked={!isAuthenticated}
        >
          Packages
        </NavItem>
        <NavItem
          active={activeTab === 'repos'}
          onClick={() => onNavigate('repos')}
          icon={<Github className="h-4 w-4" />}
          collapsed={collapsed}
          locked={!isAuthenticated}
        >
          Repositories
        </NavItem>
        <NavItem
          active={activeTab === 'library'}
          onClick={() => onNavigate('library')}
          icon={<Library className="h-4 w-4" />}
          collapsed={collapsed}
          locked={!isAuthenticated}
        >
          Library
        </NavItem>
      </nav>

      {/* Bottom section */}
      <div className="border-t border-sidebar-border p-1.5">
        {isAuthenticated ? (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button
                className={cn(
                  'w-full flex items-center rounded-md p-1.5 hover:bg-sidebar-accent transition-colors',
                  collapsed ? 'justify-center' : 'gap-2'
                )}
              >
                <img 
                  src={user.avatar_url} 
                  alt={user.login} 
                  className="w-6 h-6 rounded-full flex-shrink-0"
                />
                {!collapsed && (
                  <span className="flex-1 text-left text-xs font-medium truncate">
                    {user.login}
                  </span>
                )}
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent 
              side={collapsed ? 'right' : 'top'} 
              align="start"
              className="w-48"
            >
              <div className="px-2 py-1.5">
                <p className="text-sm font-medium">{user.login}</p>
                {user.name && (
                  <p className="text-xs text-muted-foreground">{user.name}</p>
                )}
              </div>
              <DropdownMenuSeparator />
              
              <DropdownMenuSub>
                <DropdownMenuSubTrigger className="text-xs">
                  {theme === 'dark' ? (
                    <Moon className="h-3.5 w-3.5 mr-2" />
                  ) : theme === 'light' ? (
                    <Sun className="h-3.5 w-3.5 mr-2" />
                  ) : (
                    <Monitor className="h-3.5 w-3.5 mr-2" />
                  )}
                  Theme
                </DropdownMenuSubTrigger>
                <DropdownMenuPortal>
                  <DropdownMenuSubContent>
                    <DropdownMenuItem onClick={() => setTheme('light')} className="text-xs">
                      <Sun className="h-3.5 w-3.5 mr-2" />
                      Light
                      {theme === 'light' && <span className="ml-auto">✓</span>}
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => setTheme('dark')} className="text-xs">
                      <Moon className="h-3.5 w-3.5 mr-2" />
                      Dark
                      {theme === 'dark' && <span className="ml-auto">✓</span>}
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => setTheme('system')} className="text-xs">
                      <Monitor className="h-3.5 w-3.5 mr-2" />
                      System
                      {theme === 'system' && <span className="ml-auto">✓</span>}
                    </DropdownMenuItem>
                  </DropdownMenuSubContent>
                </DropdownMenuPortal>
              </DropdownMenuSub>
              
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={logout} className="text-xs text-destructive focus:text-destructive">
                <LogOut className="h-3.5 w-3.5 mr-2" />
                Sign out
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        ) : (
          <GitHubLoginButton login={login} compact={collapsed} className="w-full" />
        )}
      </div>
    </aside>
  );
}

// =============================================================================
// NavItem Component
// =============================================================================

interface NavItemProps {
  active: boolean;
  onClick: () => void;
  icon: React.ReactNode;
  children: React.ReactNode;
  collapsed?: boolean;
  locked?: boolean;
}

function NavItem({ active, onClick, icon, children, collapsed, locked }: NavItemProps) {
  return (
    <Button
      variant="ghost"
      onClick={onClick}
      title={collapsed ? String(children) : undefined}
      className={cn(
        'h-8 text-xs font-medium transition-colors',
        collapsed 
          ? 'w-full justify-center px-0' 
          : 'w-full justify-start gap-2 px-2',
        active
          ? 'bg-sidebar-accent text-foreground'
          : 'text-muted-foreground hover:bg-sidebar-accent hover:text-foreground',
        locked && 'opacity-50'
      )}
    >
      {icon}
      {!collapsed && (
        <span className="flex-1 flex items-center gap-2">
          {children}
          {locked && <Lock className="h-3 w-3 ml-auto" />}
        </span>
      )}
    </Button>
  );
}
