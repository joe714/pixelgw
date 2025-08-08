import { useState } from 'react';
import { PixelDisplay } from '@/components/PixelDisplay';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';

/**
 * Debug page for testing PixelDisplay LED matrix alignment
 */
export function PixelDisplayDebug() {
  const [scale, setScale] = useState(7);

  // Create test pattern data URLs
  const createTestPattern = (width: number, height: number, pattern: string) => {
    const canvas = document.createElement('canvas');
    canvas.width = width;
    canvas.height = height;
    const ctx = canvas.getContext('2d')!;

    if (pattern === 'checkerboard') {
      // Checkerboard pattern
      for (let x = 0; x < width; x++) {
        for (let y = 0; y < height; y++) {
          const isLight = (Math.floor(x / 2) + Math.floor(y / 2)) % 2 === 0;
          ctx.fillStyle = isLight ? '#ffffff' : '#000000';
          ctx.fillRect(x, y, 1, 1);
        }
      }
    } else if (pattern === 'grid') {
      // Grid with dots at regular intervals
      ctx.fillStyle = '#222222';
      ctx.fillRect(0, 0, width, height);
      ctx.fillStyle = '#ffffff';
      for (let x = 2; x < width; x += 4) {
        for (let y = 2; y < height; y += 4) {
          ctx.fillRect(x, y, 1, 1);
        }
      }
    } else if (pattern === 'center-dots') {
      // Single dots in center of each "virtual LED"
      ctx.fillStyle = '#333333';
      ctx.fillRect(0, 0, width, height);
      ctx.fillStyle = '#00ff00';
      // Place dots that should align with LED centers when scaled
      for (let x = 0; x < width; x++) {
        for (let y = 0; y < height; y++) {
          ctx.fillRect(x, y, 1, 1);
        }
      }
    } else if (pattern === 'single-pixel') {
      // Single white pixel on black background
      ctx.fillStyle = '#000000';
      ctx.fillRect(0, 0, width, height);
      ctx.fillStyle = '#ffffff';
      ctx.fillRect(0, 0, 1, 1); // Top-left pixel
      ctx.fillRect(width-1, 0, 1, 1); // Top-right pixel
      ctx.fillRect(0, height-1, 1, 1); // Bottom-left pixel
      ctx.fillRect(width-1, height-1, 1, 1); // Bottom-right pixel
    }

    return canvas.toDataURL('image/png');
  };

  const patterns = {
    checkerboard: createTestPattern(16, 8, 'checkerboard'),
    grid: createTestPattern(16, 8, 'grid'),
    'center-dots': createTestPattern(8, 4, 'center-dots'),
    'single-pixel': createTestPattern(8, 4, 'single-pixel'),
  };

  const ledRadius = Math.floor((scale - 3) / 2);
  const centerPos = Math.floor(scale / 2);

  return (
    <div className="p-8 max-w-6xl mx-auto">
      <h1 className="text-3xl font-bold mb-6">PixelDisplay LED Matrix Alignment Debug</h1>
      
      <div className="mb-6 space-y-4">
        <div>
          <Label>Scale Factor: {scale}x</Label>
          <div className="flex gap-2 mt-2">
            {[5, 7, 9, 11].map(s => (
              <Button
                key={s}
                variant={scale === s ? 'default' : 'outline'}
                size="sm"
                onClick={() => setScale(s)}
              >
                {s}x
              </Button>
            ))}
          </div>
        </div>
        
        <div className="text-sm space-y-1 bg-gray-100 p-3 rounded">
          <p><strong>Current Settings:</strong></p>
          <p>Scale: {scale}x ({scale * 64}×{scale * 32} pixels)</p>
          <p>LED Radius: {ledRadius}px</p>
          <p>LED Center: ({centerPos}, {centerPos}) within each {scale}×{scale} tile</p>
          <p>LED Diameter: {ledRadius * 2}px</p>
          <p>Matrix Border: {(scale - (ledRadius * 2)) / 2}px on each side</p>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {Object.entries(patterns).map(([name, src]) => (
          <div key={name} className="space-y-2">
            <h3 className="text-lg font-semibold capitalize">{name.replace('-', ' ')}</h3>
            <div className="border-2 border-dashed border-gray-300 p-4 bg-gray-50">
              <PixelDisplay 
                src={src}
                scale={scale}
                showFrame={false}
                className="mx-auto"
              />
            </div>
            <div className="text-xs text-gray-600">
              {name === 'checkerboard' && 'Alternating pixels - LEDs should show clear on/off pattern'}
              {name === 'grid' && 'White dots every 4 pixels - should align with some LEDs'}
              {name === 'center-dots' && 'Solid fill - all LEDs should be bright'}
              {name === 'single-pixel' && 'Corner pixels - check LED alignment at edges'}
            </div>
          </div>
        ))}
      </div>

      <div className="mt-8 space-y-4">
        <h2 className="text-xl font-semibold">Alignment Analysis</h2>
        <div className="bg-yellow-50 border border-yellow-200 p-4 rounded">
          <h3 className="font-semibold mb-2">What to look for:</h3>
          <ul className="space-y-1 text-sm">
            <li>• <strong>Checkerboard:</strong> LEDs should show crisp on/off alternation</li>
            <li>• <strong>Grid:</strong> White dots should appear centered within LEDs</li>
            <li>• <strong>Center Dots:</strong> All LEDs should appear uniformly bright</li>
            <li>• <strong>Single Pixel:</strong> Corner LEDs should be perfectly positioned</li>
          </ul>
        </div>
        
        <div className="bg-red-50 border border-red-200 p-4 rounded">
          <h3 className="font-semibold mb-2">Current Issues:</h3>
          <ul className="space-y-1 text-sm">
            <li>• If patterns appear shifted, the mask positioning is incorrect</li>
            <li>• If LEDs have uneven borders, the centering calculation is wrong</li>
            <li>• If some pixels are cut off, the scale factor math needs adjustment</li>
          </ul>
        </div>
      </div>

      <div className="mt-6">
        <h3 className="text-lg font-semibold mb-2">CSS Debug Info</h3>
        <pre className="bg-gray-100 p-3 text-xs rounded overflow-x-auto">
{`maskImage: radial-gradient(
  circle ${ledRadius}px at ${centerPos}px ${centerPos}px, 
  transparent ${ledRadius}px, 
  black ${ledRadius + 1}px
)
maskSize: ${scale}px ${scale}px
maskPosition: 0px 0px`}
        </pre>
      </div>
    </div>
  );
}