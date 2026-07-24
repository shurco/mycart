# OpenBSD WebdriverIO Patch

## Problem
WebdriverIO and @puppeteer/browsers don't support OpenBSD out of the box, causing "The current platform is not supported" error.

## Solution
Patched `@puppeteer/browsers` to recognize OpenBSD as a Linux-like platform.

## Files Patched

### 1. Platform Detection (4 files)

**web/admin/node_modules/@puppeteer/browsers/lib/cjs/detectPlatform.js**
**web/admin/node_modules/@puppeteer/browsers/lib/esm/detectPlatform.js**
**web/site/node_modules/@puppeteer/browsers/lib/cjs/detectPlatform.js**
**web/site/node_modules/@puppeteer/browsers/lib/esm/detectPlatform.js**

Changed:
```javascript
switch (platform) {
    case 'darwin':
        return arch === 'arm64' ? BrowserPlatform.MAC_ARM : BrowserPlatform.MAC;
    case 'linux':
        return arch === 'arm64'
            ? BrowserPlatform.LINUX_ARM
            : BrowserPlatform.LINUX;
```

To:
```javascript
switch (platform) {
    case 'darwin':
        return arch === 'arm64' ? BrowserPlatform.MAC_ARM : BrowserPlatform.MAC;
    case 'linux':
    case 'openbsd':
    case 'freebsd':
        return arch === 'arm64'
            ? BrowserPlatform.LINUX_ARM
            : BrowserPlatform.LINUX;
```

### 2. Vitest Browser Configuration (2 files)

**web/admin/vitest.browser.config.ts**
**web/site/vitest.browser.config.ts**

Added OpenBSD-specific browser and chromedriver configuration:
```typescript
test: {
  browser: {
    enabled: true,
    provider: webdriverio({
      capabilities: {
        'goog:chromeOptions': {
          binary: '/usr/local/bin/chrome',  // OpenBSD Chromium path
          args: [
            '--headless',
            '--no-sandbox',
            '--disable-setuid-sandbox',
            '--disable-dev-shm-usage',
            '--disable-gpu'
          ]
        },
        'wdio:chromedriverOptions': {
          binary: '/usr/local/bin/chromedriver'  // OpenBSD chromedriver path
        }
      }
    }),
    instances: [{ browser: 'chrome' }],
    headless: true
  },
  include: ['src/**/*.browser.test.ts'],
  testTimeout: 30000
}
```

## Prerequisites on OpenBSD

Install Chromium and chromedriver:
```bash
pkg_add chromium
# chromedriver comes with chromium package
```

Verify installations:
```bash
which chrome           # /usr/local/bin/chrome
which chromedriver     # /usr/local/bin/chromedriver
chromedriver --version # Should show version
```

## Results

### ✅ Fixed
1. Platform detection error - OpenBSD now recognized
2. Chromedriver connection - Using local binaries
3. Browser launch - Chrome/Chromium starts successfully

### ⚠️ Remaining Issues
1. **Vite Module Resolution**: Test files fail to load with "Failed to fetch dynamically imported module"
   - Error: `http://localhost:PORT/home/wj/...` (absolute path issue)
   - This is a SvelteKit/Vite configuration issue, not WebdriverIO
   - Needs investigation of Vite's `server.fs.allow` or base URL configuration

## Testing

Start the server:
```bash
pushd web/admin/ && npx vite build && popd
pushd web/site/ && npx vite build && popd
go run ./cmd serve
```

Run browser tests:
```bash
cd web/admin
npm run test:browser

cd web/site
npm run test:browser
```

## Patch Persistence

**⚠️ Important**: These patches are in `node_modules` and will be lost on `npm install`.

### Option 1: Post-Install Script
Add to `package.json`:
```json
{
  "scripts": {
    "postinstall": "node scripts/patch-puppeteer-browsers.js"
  }
}
```

Create `web/admin/scripts/patch-puppeteer-browsers.js`:
```javascript
const fs = require('fs');
const path = require('path');

const files = [
  'node_modules/@puppeteer/browsers/lib/cjs/detectPlatform.js',
  'node_modules/@puppeteer/browsers/lib/esm/detectPlatform.js'
];

files.forEach(file => {
  const filePath = path.join(__dirname, '..', file);
  let content = fs.readFileSync(filePath, 'utf8');
  
  content = content.replace(
    /case 'linux':/g,
    "case 'linux':\n        case 'openbsd':\n        case 'freebsd':"
  );
  
  fs.writeFileSync(filePath, content, 'utf8');
  console.log(`Patched ${file}`);
});
```

### Option 2: Use patch-package
```bash
npm install --save-dev patch-package
npm run test:browser  # Let it fail
npx patch-package @puppeteer/browsers
```

Then add to package.json:
```json
{
  "scripts": {
    "postinstall": "patch-package"
  }
}
```

## Alternative: Environment Variable

Instead of patching, set chromedriver path via environment:
```bash
export CHROMEDRIVER_PATH=/usr/local/bin/chromedriver
npm run test:browser
```

But platform detection patch is still required.

## Status

- **Platform Detection**: ✅ Working
- **Chromedriver Launch**: ✅ Working  
- **Browser Connection**: ✅ Working
- **Test Execution**: ⚠️ Vite module resolution issue (next step)

## Next Steps

1. Fix Vite module resolution for browser tests
2. Consider using `patch-package` for permanent patches
3. Test actual browser test scenarios
4. Verify GREEN phase of TDD cycle
