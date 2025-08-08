import { useState } from 'react';
import { PixelDisplay, PixelDisplayGrid } from '@/components/PixelDisplay';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { Separator } from '@/components/ui/separator';

/**
 * Demo page showcasing the PixelDisplay component capabilities
 */
export function PixelDisplayDemo() {
  const [scale, setScale] = useState(4);
  const [brightness, setBrightness] = useState(100);
  const [showScanLines, setShowScanLines] = useState(false);
  const [frameTheme, setFrameTheme] = useState<'black' | 'white' | 'wood' | 'metal'>('black');
  const [showFrame, setShowFrame] = useState(true);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(false);

  // Demo image URL - replace with actual channel image
  const demoImageUrl = "/api/channels/default";

  // Sample grid data
  const gridDisplays = [
    { id: '1', src: demoImageUrl, name: 'Living Room' },
    { id: '2', src: demoImageUrl, name: 'Office' },
    { id: '3', src: demoImageUrl, name: 'Kitchen' },
    { id: '4', src: demoImageUrl, name: 'Bedroom' },
    { id: '5', src: demoImageUrl, name: 'Garage' },
    { id: '6', src: demoImageUrl, name: 'Basement' },
  ];

  return (
    <div className="p-8 max-w-7xl mx-auto">
      <h1 className="text-3xl font-bold mb-6">PixelDisplay Component Demo</h1>
      
      {/* Main Interactive Demo */}
      <section className="mb-12">
        <h2 className="text-xl font-semibold mb-4">Interactive Display</h2>
        
        <div className="flex gap-8">
          {/* Display Preview */}
          <div className="flex-1 flex justify-center items-center bg-gray-100 dark:bg-gray-900 rounded-lg p-8">
            <PixelDisplay
              src={demoImageUrl}
              scale={scale}
              showFrame={showFrame}
              frameTheme={frameTheme}
              showScanLines={showScanLines}
              brightness={brightness}
              loading={loading}
              error={error}
              onClick={() => console.log('Display clicked!')}
            />
          </div>
          
          {/* Controls */}
          <div className="w-80 space-y-4">
            <div>
              <Label>Scale: {scale}x</Label>
              <input
                type="range"
                min="1"
                max="8"
                value={scale}
                onChange={(e) => setScale(Number(e.target.value))}
                className="w-full"
              />
            </div>
            
            <div>
              <Label>Brightness: {brightness}%</Label>
              <input
                type="range"
                min="0"
                max="100"
                value={brightness}
                onChange={(e) => setBrightness(Number(e.target.value))}
                className="w-full"
              />
            </div>
            
            <div>
              <Label>Frame Theme</Label>
              <div className="flex gap-2 mt-2">
                {(['black', 'white', 'wood', 'metal'] as const).map((theme) => (
                  <Button
                    key={theme}
                    variant={frameTheme === theme ? 'default' : 'outline'}
                    size="sm"
                    onClick={() => setFrameTheme(theme)}
                  >
                    {theme}
                  </Button>
                ))}
              </div>
            </div>
            
            <div className="space-y-2">
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={showFrame}
                  onChange={(e) => setShowFrame(e.target.checked)}
                />
                <span>Show Frame</span>
              </label>
              
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={showScanLines}
                  onChange={(e) => setShowScanLines(e.target.checked)}
                />
                <span>Show Scan Lines</span>
              </label>
            </div>
            
            <Separator />
            
            <div>
              <Label>Test States</Label>
              <div className="flex gap-2 mt-2">
                <Button
                  variant={loading ? 'default' : 'outline'}
                  size="sm"
                  onClick={() => {
                    setLoading(!loading);
                    setError(false);
                  }}
                >
                  Loading
                </Button>
                <Button
                  variant={error ? 'destructive' : 'outline'}
                  size="sm"
                  onClick={() => {
                    setError(!error);
                    setLoading(false);
                  }}
                >
                  Error
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => {
                    setLoading(false);
                    setError(false);
                  }}
                >
                  Reset
                </Button>
              </div>
            </div>
          </div>
        </div>
      </section>
      
      <Separator className="my-8" />
      
      {/* Frame Theme Showcase */}
      <section className="mb-12">
        <h2 className="text-xl font-semibold mb-4">Frame Themes</h2>
        <div className="flex gap-4 justify-around">
          {(['black', 'white', 'wood', 'metal'] as const).map((theme) => (
            <div key={theme} className="text-center">
              <PixelDisplay
                src={demoImageUrl}
                scale={2}
                frameTheme={theme}
              />
              <p className="mt-2 text-sm capitalize">{theme}</p>
            </div>
          ))}
        </div>
      </section>
      
      <Separator className="my-8" />
      
      {/* Scale Comparison */}
      <section className="mb-12">
        <h2 className="text-xl font-semibold mb-4">Scale Comparison</h2>
        <div className="flex gap-4 items-end justify-around">
          {[1, 2, 3, 4, 6].map((s) => (
            <div key={s} className="text-center">
              <PixelDisplay
                src={demoImageUrl}
                scale={s}
                showFrame={false}
                className="border border-gray-300"
              />
              <p className="mt-2 text-sm">{s}x Scale</p>
            </div>
          ))}
        </div>
      </section>
      
      <Separator className="my-8" />
      
      {/* Grid Layout Demo */}
      <section className="mb-12">
        <h2 className="text-xl font-semibold mb-4">Grid Layout</h2>
        <PixelDisplayGrid
          displays={gridDisplays}
          scale={2}
          columns={3}
          frameTheme="white"
        />
      </section>
      
      <Separator className="my-8" />
      
      {/* Inline Usage Examples */}
      <section className="mb-12">
        <h2 className="text-xl font-semibold mb-4">Inline Usage</h2>
        <div className="space-y-4">
          <div className="flex items-center gap-4">
            <PixelDisplay
              src={demoImageUrl}
              scale={1}
              showFrame={false}
              className="border"
            />
            <span>Minimal inline display (1x scale, no frame)</span>
          </div>
          
          <div className="flex items-center gap-4">
            <PixelDisplay
              src={demoImageUrl}
              scale={2}
              showFrame={false}
              showScanLines={true}
              brightness={70}
              className="rounded-lg border-2 border-blue-500"
            />
            <span>Retro style with scan lines and reduced brightness</span>
          </div>
          
          <div className="flex items-center gap-4">
            <PixelDisplay
              src={demoImageUrl}
              scale={2}
              frameTheme="metal"
              showPowerLED={true}
              powerOn={true}
            />
            <span>Metal frame with power LED indicator</span>
          </div>
        </div>
      </section>
    </div>
  );
}