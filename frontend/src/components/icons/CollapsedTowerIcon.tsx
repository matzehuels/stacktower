/**
 * Collapsed/fallen tower SVG icon.
 * Used for error states and 404 pages.
 */

interface CollapsedTowerIconProps {
  className?: string;
}

export function CollapsedTowerIcon({ className }: CollapsedTowerIconProps) {
  return (
    <svg
      viewBox="0 0 200 120"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={className}
    >
      {/* Ground line */}
      <path
        d="M10 100 L190 100"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        opacity="0.3"
      />
      
      {/* Dust clouds */}
      <ellipse cx="60" cy="95" rx="15" ry="8" fill="currentColor" opacity="0.1" />
      <ellipse cx="100" cy="92" rx="20" ry="10" fill="currentColor" opacity="0.08" />
      <ellipse cx="145" cy="96" rx="12" ry="6" fill="currentColor" opacity="0.1" />
      
      {/* Fallen tower blocks - rotated and scattered */}
      {/* Block 1 - tilted on ground */}
      <g transform="rotate(-75 45 85)">
        <rect x="30" y="70" width="30" height="20" rx="3" fill="currentColor" opacity="0.6" />
        <rect x="32" y="72" width="26" height="16" rx="2" fill="currentColor" opacity="0.2" />
      </g>
      
      {/* Block 2 - slightly tilted */}
      <g transform="rotate(-60 80 80)">
        <rect x="65" y="65" width="28" height="18" rx="3" fill="currentColor" opacity="0.5" />
        <rect x="67" y="67" width="24" height="14" rx="2" fill="currentColor" opacity="0.15" />
      </g>
      
      {/* Block 3 - flat on ground */}
      <g transform="rotate(-85 115 90)">
        <rect x="100" y="75" width="26" height="16" rx="3" fill="currentColor" opacity="0.55" />
        <rect x="102" y="77" width="22" height="12" rx="2" fill="currentColor" opacity="0.18" />
      </g>
      
      {/* Block 4 - bounced away */}
      <g transform="rotate(-45 150 75)">
        <rect x="138" y="62" width="24" height="14" rx="3" fill="currentColor" opacity="0.45" />
        <rect x="140" y="64" width="20" height="10" rx="2" fill="currentColor" opacity="0.12" />
      </g>
      
      {/* Block 5 - smallest, rolled away */}
      <g transform="rotate(15 170 88)">
        <rect x="162" y="80" width="18" height="12" rx="2" fill="currentColor" opacity="0.4" />
        <rect x="164" y="82" width="14" height="8" rx="1" fill="currentColor" opacity="0.1" />
      </g>
      
      {/* Impact lines */}
      <path d="M55 98 L50 90" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" opacity="0.2" />
      <path d="M58 97 L56 92" stroke="currentColor" strokeWidth="1" strokeLinecap="round" opacity="0.15" />
      <path d="M95 96 L92 88" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" opacity="0.2" />
      <path d="M130 97 L128 91" stroke="currentColor" strokeWidth="1" strokeLinecap="round" opacity="0.15" />
    </svg>
  );
}

