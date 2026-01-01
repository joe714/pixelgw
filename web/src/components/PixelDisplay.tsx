import React, { useState, useEffect } from 'react';
import { cn } from '@/lib/utils';

interface PixelDisplayProps {
  /** URL or base64 encoded WebP image source */
  src: string;
  /** Display scale factor (default: 4) */
  scale?: number;
  /** Show device frame (default: true) */
  showFrame?: boolean;
  /** Frame color theme */
  frameTheme?: 'black' | 'white' | 'wood' | 'metal';
  /** Show power LED indicator */
  showPowerLED?: boolean;
  /** Power LED state */
  powerOn?: boolean;
  /** Additional CSS classes */
  className?: string;
  /** Alt text for accessibility */
  alt?: string;
  /** Show scan lines effect */
  showScanLines?: boolean;
  /** Brightness level (0-100) */
  brightness?: number;
  /** Click handler */
  onClick?: () => void;
  /** Loading state */
  loading?: boolean;
  /** Error state */
  error?: boolean;
}

/**
 * PixelDisplay - Simulated pixel display device component
 * 
 * Renders WebP images with a retro pixel display aesthetic,
 * simulating the appearance of a Tidbyt or similar LED matrix display.
 */
export const PixelDisplay: React.FC<PixelDisplayProps> = ({
  src,
  scale = 4,
  showFrame = true,
  frameTheme = 'black',
  showPowerLED = true,
  powerOn = true,
  className,
  alt = 'Pixel display',
  showScanLines = false,
  brightness = 100,
  onClick,
  loading = false,
  error = false,
}) => {
  const [imageError, setImageError] = useState(false);

  // Native resolution of Tidbyt displays
  const NATIVE_WIDTH = 64;
  const NATIVE_HEIGHT = 32;

  const displayWidth = NATIVE_WIDTH * scale;
  const displayHeight = NATIVE_HEIGHT * scale;

  useEffect(() => {
    setImageError(false);
  }, [src]);

  const frameStyles = {
    black: 'bg-gray-900 border-gray-800',
    white: 'bg-gray-100 border-gray-200',
    wood: 'bg-amber-900 border-amber-800',
    metal: 'bg-gradient-to-b from-gray-400 to-gray-600 border-gray-500',
  };

  const ledColors = {
    on: 'bg-green-500 animate-pulse',
    off: 'bg-gray-600',
    error: 'bg-red-500 animate-pulse',
  };

  const handleImageLoad = () => {
    setImageError(false);
  };

  const handleImageError = () => {
    setImageError(true);
  };

  const renderContent = () => {
    if (loading) {
      return (
        <div className="flex items-center justify-center w-full h-full bg-gray-800">
          <div className="text-gray-500 text-xs">Loading...</div>
        </div>
      );
    }

    if (error || imageError) {
      return (
        <div className="flex items-center justify-center w-full h-full bg-gray-800">
          <div className="text-red-500 text-xs">Error</div>
        </div>
      );
    }

    return (
      <>
        <img
          src={src}
          alt={alt}
          className="w-full h-full object-contain"
          style={{
            imageRendering: 'pixelated',
            filter: `brightness(${brightness}%)`,
          }}
          onLoad={handleImageLoad}
          onError={handleImageError}
        />
        
        {/* Scan lines effect overlay */}
        {showScanLines && (
          <div 
            className="absolute inset-0 pointer-events-none"
            style={{
              background: `repeating-linear-gradient(
                0deg,
                transparent,
                transparent 2px,
                rgba(0, 0, 0, 0.1) 2px,
                rgba(0, 0, 0, 0.1) 4px
              )`,
            }}
          />
        )}

        {/* LED matrix overlay - creates black grid with circular LED cutouts */}
        <div
          className="absolute inset-0 pointer-events-none"
          style={{
            backgroundImage: `radial-gradient(circle at center, transparent 0%, transparent 35%, black 45%, black 100%)`,
            backgroundSize: `${scale}px ${scale}px`,
            backgroundRepeat: 'repeat',
          }}
        />
      </>
    );
  };

  const display = (
    <div
      className={cn(
        'relative overflow-hidden bg-black',
        onClick && 'cursor-pointer hover:opacity-90 transition-opacity',
        className
      )}
      style={{
        width: `${displayWidth}px`,
        height: `${displayHeight}px`,
      }}
      onClick={onClick}
    >
      {renderContent()}
    </div>
  );

  if (!showFrame) {
    return display;
  }

  return (
    <div
      className={cn(
        'inline-block rounded-lg p-4 shadow-xl',
        frameStyles[frameTheme]
      )}
    >
      {/* Power LED */}
      {showPowerLED && (
        <div className="flex justify-end mb-2">
          <div
            className={cn(
              'w-2 h-2 rounded-full',
              error || imageError
                ? ledColors.error
                : powerOn
                ? ledColors.on
                : ledColors.off
            )}
          />
        </div>
      )}

      {/* Display screen with bezel */}
      <div className="relative rounded border-2 border-black bg-black p-1">
        {display}
      </div>

      {/* Device branding */}
      <div className="mt-2 text-center">
        <span className="text-xs opacity-50">PixelGW Display</span>
      </div>
    </div>
  );
};

/**
 * PixelDisplayGrid - Grid layout for multiple displays
 */
interface PixelDisplayGridProps {
  displays: Array<{
    id: string;
    src: string;
    name?: string;
  }>;
  scale?: number;
  columns?: number;
  showFrames?: boolean;
  frameTheme?: PixelDisplayProps['frameTheme'];
}

export const PixelDisplayGrid: React.FC<PixelDisplayGridProps> = ({
  displays,
  scale = 2,
  columns = 3,
  showFrames = true,
  frameTheme = 'black',
}) => {
  return (
    <div
      className="grid gap-4"
      style={{
        gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))`,
      }}
    >
      {displays.map((display) => (
        <div key={display.id} className="flex flex-col items-center">
          <PixelDisplay
            src={display.src}
            scale={scale}
            showFrame={showFrames}
            frameTheme={frameTheme}
          />
          {display.name && (
            <div className="mt-2 text-sm text-gray-600">{display.name}</div>
          )}
        </div>
      ))}
    </div>
  );
};