# OpenBSD Patch Automation

## Overview

The WebdriverIO OpenBSD patches have been generalized and published to:
**https://github.com/nikescar/vitest-webdriverio/tree/openbsd-support**

## Quick Setup

### Option 1: Use the Patch Script (Recommended)

1. **Download the patch script:**
```bash
curl -o scripts/patch-puppeteer-platforms.js \
  https://raw.githubusercontent.com/nikescar/vitest-webdriverio/openbsd-support/scripts/patch-puppeteer-platforms.js
```

2. **Add to package.json:**
```json
{
  "scripts": {
    "postinstall": "node scripts/patch-puppeteer-platforms.js"
  }
}
```

3. **Run once:**
```bash
npm install
```

The patch will now be automatically applied after every `npm install`.

### Option 2: Copy from Repository

For site and admin:

```bash
# Create scripts directory
mkdir -p web/site/scripts
mkdir -p web/admin/scripts

# Copy patch script
cp /tmp/vitest-webdriverio/scripts/patch-puppeteer-platforms.js web/site/scripts/
cp /tmp/vitest-webdriverio/scripts/patch-puppeteer-platforms.js web/admin/scripts/

# Update package.json for both
```

**web/site/package.json:**
```json
{
  "scripts": {
    "postinstall": "node scripts/patch-puppeteer-platforms.js",
    "test:browser": "vitest --config vitest.browser.config.ts"
  }
}
```

**web/admin/package.json:**
```json
{
  "scripts": {
    "postinstall": "node scripts/patch-puppeteer-platforms.js",
    "test:browser": "vitest --config vitest.browser.config.ts"
  }
}
```

## What Gets Patched

The script automatically patches these files:
- `node_modules/@puppeteer/browsers/lib/cjs/detectPlatform.js`
- `node_modules/@puppeteer/browsers/lib/esm/detectPlatform.js`

### Supported Platforms
- OpenBSD ✅
- FreeBSD ✅
- NetBSD ✅
- DragonFlyBSD ✅
- SunOS ✅

## Current Manual Patches

The following files are currently manually patched and will be lost on `npm install`:

### Site
- `web/site/node_modules/@puppeteer/browsers/lib/cjs/detectPlatform.js`
- `web/site/node_modules/@puppeteer/browsers/lib/esm/detectPlatform.js`

### Admin
- `web/admin/node_modules/@puppeteer/browsers/lib/cjs/detectPlatform.js`
- `web/admin/node_modules/@puppeteer/browsers/lib/esm/detectPlatform.js`

## Migration Steps

To migrate to automatic patching:

1. **Create scripts directory:**
```bash
mkdir -p web/site/scripts
mkdir -p web/admin/scripts
```

2. **Copy the patch script to both:**
```bash
# From this repo if cloned
cp /tmp/vitest-webdriverio/scripts/patch-puppeteer-platforms.js web/site/scripts/
cp /tmp/vitest-webdriverio/scripts/patch-puppeteer-platforms.js web/admin/scripts/

# Or download directly
curl -o web/site/scripts/patch-puppeteer-platforms.js \
  https://raw.githubusercontent.com/nikescar/vitest-webdriverio/openbsd-support/scripts/patch-puppeteer-platforms.js

curl -o web/admin/scripts/patch-puppeteer-platforms.js \
  https://raw.githubusercontent.com/nikescar/vitest-webdriverio/openbsd-support/scripts/patch-puppeteer-platforms.js
```

3. **Make scripts executable:**
```bash
chmod +x web/site/scripts/patch-puppeteer-platforms.js
chmod +x web/admin/scripts/patch-puppeteer-platforms.js
```

4. **Update package.json files:**

Edit `web/site/package.json`:
```json
{
  "scripts": {
    "dev": "bun --bunx vite dev",
    "build": "bun --bunx vite build",
    "preview": "bun --bunx vite preview",
    "check": "bun --bunx svelte-kit sync",
    "format": "prettier --write .",
    "update": "bun update",
    "postinstall": "node scripts/patch-puppeteer-platforms.js",
    "test": "vitest",
    "test:watch": "vitest --watch",
    "test:ui": "vitest --ui",
    "test:coverage": "vitest --coverage",
    "test:browser": "vitest --config vitest.browser.config.ts",
    "test:browser:ui": "vitest --config vitest.browser.config.ts --ui"
  }
}
```

Edit `web/admin/package.json` similarly.

5. **Test the automation:**
```bash
# Remove node_modules to test
rm -rf web/site/node_modules web/admin/node_modules

# Reinstall - patches should apply automatically
cd web/site && npm install
cd ../admin && npm install
```

6. **Verify patches applied:**
```bash
# Check if patches worked
cd web/site
npm run test:browser -- --run

cd ../admin
npm run test:browser -- --run
```

## Verification

After setup, verify the patch is applied:

```bash
cd web/site
node scripts/patch-puppeteer-platforms.js
```

Expected output:
```
🔧 Patching @puppeteer/browsers for BSD/Unix platform support
Current platform: openbsd
Adding support for: openbsd, freebsd, netbsd, dragonflybsd, sunos

✓ node_modules/@puppeteer/browsers/lib/cjs/detectPlatform.js already patched
✓ node_modules/@puppeteer/browsers/lib/esm/detectPlatform.js already patched

📊 Patch Summary:
  ✓ Successfully patched: 2

✅ Your platform (openbsd) is now supported!
```

## Benefits of Automation

1. **No Manual Patching**: Patches apply automatically after `npm install`
2. **Consistent**: Same patch applied across all environments
3. **Maintainable**: Update script once, affects all projects
4. **Portable**: Works on any BSD system automatically
5. **Backup**: Script creates backups before patching

## Troubleshooting

### Script Not Running

If the postinstall script doesn't run:

```bash
# Run manually
cd web/site
node scripts/patch-puppeteer-platforms.js

cd ../admin
node scripts/patch-puppeteer-platforms.js
```

### Patches Lost After Install

Ensure `postinstall` is in the scripts section:

```bash
# Check if postinstall exists
cd web/site
npm run postinstall

# Should output patch information
```

### Script Not Found

Ensure the script exists:

```bash
ls -la web/site/scripts/patch-puppeteer-platforms.js
ls -la web/admin/scripts/patch-puppeteer-platforms.js
```

## Additional Resources

- **Full Documentation**: https://github.com/nikescar/vitest-webdriverio/blob/openbsd-support/docs/BSD_PLATFORMS.md
- **Examples**: https://github.com/nikescar/vitest-webdriverio/tree/openbsd-support/examples/bsd
- **Original Patch**: `OPENBSD_WEBDRIVERIO_PATCH.md` in this repository

## Future Updates

When the `openbsd-support` branch is merged or if you want to use the official package:

```bash
# The patch script will be included in the package
npm install @vitest/browser-webdriverio

# Just add postinstall to package.json
{
  "scripts": {
    "postinstall": "node node_modules/@vitest/browser-webdriverio/scripts/patch-puppeteer-platforms.js"
  }
}
```

## Contributing

Found issues or improvements? Contribute to:
https://github.com/nikescar/vitest-webdriverio
