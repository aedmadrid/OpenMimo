import React from 'react';
import omLogoUrl from '../assets/om-logo.svg';

interface Props {
  size?: number;
}

/**
 * Renders the official OM LOGO.svg with Apple-style squircle rounding on macOS & Linux
 * and clean subtle corners on Windows.
 */
export const AppLogo: React.FC<Props> = ({ size = 28 }) => {
  const plat = (window.__OPENMIMO_PLATFORM__ || 'macos').toLowerCase();
  const isAppleOrLinux = plat === 'macos' || plat === 'darwin' || plat === 'linux';
  return (
    <img
      src={omLogoUrl}
      alt="openmimo"
      width={size}
      height={size}
      draggable={false}
      style={{
        width: size,
        height: size,
        borderRadius: isAppleOrLinux ? '22.5%' : '5px',
        display: 'block',
        flexShrink: 0,
        boxShadow: '0 1px 3px rgba(0, 63, 255, 0.25)',
      }}
    />
  );
};
