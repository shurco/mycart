# Development on BSD Systems (OpenBSD/FreeBSD)

This guide covers BSD-specific setup requirements for developing with Tailwind CSS v4 and Patchright E2E testing.

## Overview

Tailwind CSS v4 migrated from PostCSS to Rust-based native binaries for better performance. Two native modules are required:

- **lightningcss** - CSS parsing, transformation, and minification
- **@tailwindcss/oxide** - Tailwind CSS v4 core engine

These packages don't provide pre-built binaries for OpenBSD/FreeBSD, so they must be built from source.

## Prerequisites

```bash
# Install Rust toolchain
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh

# Verify installation
cargo --version
rustc --version
```

## Building lightningcss

### 1. Clone and Build

```bash
# Clone repository
cd /tmp
git clone https://github.com/parcel-bundler/lightningcss.git
cd lightningcss

# Build the Node.js addon
cargo build --release -p lightningcss_node

# Build the CLI tool (optional)
cargo build --release --bin lightningcss --features cli
```

### 2. Install Binaries

```bash
# Create storage directory
mkdir -p ~/.local/lib/node-native-openbsd

# Copy Node.js addon
cp target/release/liblightningcss_node.so \
   ~/.local/lib/node-native-openbsd/lightningcss.openbsd-x64.node

# Copy CLI tool (optional)
cp target/release/lightningcss ~/.cargo/bin/
```

**Note:** The `.so` shared library is renamed to `.node` with platform suffix for Node.js NAPI-RS loading.

## Building @tailwindcss/oxide

### 1. Clone and Build

```bash
# Clone repository
cd /tmp
git clone https://github.com/tailwindlabs/tailwindcss.git
cd tailwindcss

# Build the oxide crate
cargo build --release -p tailwind-oxide
```

### 2. Install Binary

```bash
# Copy to storage directory
cp target/release/libtailwind_oxide.so \
   ~/.local/lib/node-native-openbsd/tailwindcss-oxide.openbsd-x64.node
```

## Installing Dependencies

### npm install with --legacy-peer-deps

Due to peer dependency conflicts between Vite 7 and `@sveltejs/vite-plugin-svelte@5.1.1` (which expects Vite 6), you must use the `--legacy-peer-deps` flag:

```bash
# In web/admin
cd web/admin
npm install --legacy-peer-deps

# In web/site
cd web/site
npm install --legacy-peer-deps
```

**Why this is needed:**
- Vite 7 is required for the latest features
- `@sveltejs/vite-plugin-svelte@5.1.1` declares a peer dependency on Vite 6
- The packages work correctly together despite the version mismatch
- `--legacy-peer-deps` bypasses npm's strict peer dependency checking

### SvelteKit Sync

After installing dependencies, run SvelteKit sync to generate TypeScript configs:

```bash
# In web/admin
cd web/admin
./node_modules/.bin/svelte-kit sync

# In web/site
cd web/site
./node_modules/.bin/svelte-kit sync
```

## Automated Installation

After building binaries once, use the postinstall script to automatically restore them after `npm install`:

### Run Manually

```bash
# From project root
node scripts/postinstall-openbsd-natives.js
```

### Add to package.json (Optional)

```json
{
  "scripts": {
    "postinstall": "node scripts/postinstall-openbsd-natives.js"
  }
}
```

## What the Postinstall Script Does

The script (`scripts/postinstall-openbsd-natives.js`) performs two operations:

### 1. Copy Native Binaries

Copies pre-built binaries from `~/.local/lib/node-native-openbsd/` to:

- `web/admin/node_modules/@tailwindcss/node/node_modules/lightningcss/lightningcss.openbsd-x64.node`
- `web/admin/node_modules/@tailwindcss/oxide/tailwindcss-oxide.openbsd-x64.node`
- `web/site/node_modules/@tailwindcss/node/node_modules/lightningcss/lightningcss.openbsd-x64.node`
- `web/site/node_modules/@tailwindcss/oxide/tailwindcss-oxide.openbsd-x64.node`

### 2. Patch JavaScript Loaders

**For @tailwindcss/oxide/index.js:**

Adds OpenBSD/FreeBSD platform detection before the "Unsupported OS" error:

```javascript
} else if (process.platform === 'openbsd' || process.platform === 'freebsd') {
  if (process.arch === 'x64') {
    try {
      return require('./tailwindcss-oxide.openbsd-x64.node')
    } catch (e) {
      loadErrors.push(e)
    }
  } else {
    loadErrors.push(new Error(`Unsupported architecture on BSD: ${process.arch}`))
  }
} else {
  loadErrors.push(new Error(`Unsupported OS: ${process.platform}, architecture: ${process.arch}`))
}
```

**For lightningcss:**

No patching required—lightningcss already supports custom platform binaries via NAPI-RS naming convention.

## Why Patching is Needed

NAPI-RS (Rust-to-Node.js bindings) generates platform detection code automatically, but only for "officially supported" platforms (Windows, macOS, Linux). The generated `index.js` contains an if/else chain checking `process.platform`:

```javascript
if (process.platform === 'darwin') {
  // macOS loading
} else if (process.platform === 'linux') {
  // Linux loading
} else if (process.platform === 'win32') {
  // Windows loading
} else {
  // Error: Unsupported OS
}
```

**The patch adds OpenBSD/FreeBSD branches** before the final error, telling Node.js to load the `.openbsd-x64.node` binary.

## File Naming Convention

Native Node.js addons follow this pattern:

```
<package-name>.<platform>-<arch>.node
```

Examples:
- `lightningcss.openbsd-x64.node`
- `tailwindcss-oxide.openbsd-x64.node`
- `tailwindcss-oxide.linux-x64.node` (comparison)

## Verification

After running the postinstall script, verify builds work:

```bash
# Test admin build
cd web/admin
npm run build

# Test site build
cd web/site
npm run build
```

Expected output:
```
vite v7.3.6 building ssr environment for production...
✓ 305 modules transformed.
✓ built in X.XXs
```

## Troubleshooting

### npm ERESOLVE unable to resolve dependency tree

**Error:**
```
npm error ERESOLVE unable to resolve dependency tree
npm error peer vite@"^6.0.0" from @sveltejs/vite-plugin-svelte@5.1.1
```

**Cause:** Peer dependency conflict between Vite 7 and `@sveltejs/vite-plugin-svelte`.

**Solution:**
```bash
npm install --legacy-peer-deps
```

### "Cannot find module '../lightningcss.openbsd-x64.node'"

**Cause:** Binary not copied to node_modules.

**Solution:**
```bash
node scripts/postinstall-openbsd-natives.js
```

### "Cannot find native binding"

**Cause:** The oxide index.js wasn't patched.

**Solution:**
```bash
# Re-run postinstall (includes patching)
node scripts/postinstall-openbsd-natives.js
```

### Build fails with "Unsupported OS: openbsd"

**Cause:** Patch wasn't applied or was overwritten by npm install.

**Solution:**
```bash
# Always run postinstall after npm install
npm install --legacy-peer-deps
node scripts/postinstall-openbsd-natives.js
```

### Cargo build fails

**Common issues:**

1. **Missing Rust:** Install from https://rustup.rs
2. **Puppeteer error during npm install in lightningcss repo:**
   - Don't use `npm build` in the cloned repos
   - Use `cargo build` directly instead
3. **Old Rust version:** Update with `rustup update`

## Alternative: Manual Installation

If the postinstall script doesn't work, manually copy and patch:

```bash
# Copy binaries
cp ~/.local/lib/node-native-openbsd/lightningcss.openbsd-x64.node \
   web/admin/node_modules/@tailwindcss/node/node_modules/lightningcss/

cp ~/.local/lib/node-native-openbsd/tailwindcss-oxide.openbsd-x64.node \
   web/admin/node_modules/@tailwindcss/oxide/

# Patch oxide (lightningcss needs no patch)
# Edit web/admin/node_modules/@tailwindcss/oxide/index.js
# Add the OpenBSD/FreeBSD branch shown above
```

## Maintenance

After any `npm install` that updates `@tailwindcss/oxide` or `lightningcss`:

```bash
# Re-apply patches and binaries
npm install --legacy-peer-deps
node scripts/postinstall-openbsd-natives.js
```

**Note:** Always use `--legacy-peer-deps` when running `npm install` to avoid peer dependency conflicts between Vite 7 and SvelteKit plugins.

Consider adding to package.json scripts to run automatically.

## E2E Testing with Patchright

### Overview

Patchright (Playwright fork) requires platform-specific patches and ffmpeg setup for video recording on BSD systems.

### Patchright Platform Patches

The `scripts/patch-patchright-openbsd.js` script automatically patches Patchright after `npm install` to support OpenBSD/FreeBSD:

**Patches Applied:**
1. Cache directory detection (3 locations)
2. User agent platform mapping
3. Window insets configuration
4. Chrome channel executable path
5. Platform detection for BSD systems
6. ffmpeg platform mapping

These patches run automatically via the `postinstall` hook in `package.json`.

### ffmpeg Setup for Video Recording

#### The Problem

Playwright/Patchright expects a bundled ffmpeg binary at:
```
~/.cache/ms-playwright/ffmpeg-1011/ffmpeg-linux
```

However, Patchright **doesn't distribute ffmpeg binaries for Linux** (including OpenBSD which maps to `linux-x64`). The `patchright install ffmpeg` command fails because they expect Linux systems to have ffmpeg installed system-wide.

#### The Solution

Your system's ffmpeg is fully functional with all necessary codecs. Simply symlink it to where Patchright expects it:

```bash
# Install ffmpeg (if not already installed)
doas pkg_add ffmpeg

# Create Patchright cache directory
mkdir -p ~/.cache/ms-playwright/ffmpeg-1011

# Symlink system ffmpeg to expected location
ln -sf /usr/local/bin/ffmpeg ~/.cache/ms-playwright/ffmpeg-1011/ffmpeg-linux
```

#### Verification

```bash
# Verify symlink
ls -la ~/.cache/ms-playwright/ffmpeg-1011/

# Should show:
# lrwxr-xr-x  1 user  group  21 Jul 24 08:55 ffmpeg-linux -> /usr/local/bin/ffmpeg
```

#### Video Recording Configuration

In `playwright.config.ts`, video recording options:

```typescript
video: 'off',                 // Disabled - no video recording
video: 'on',                  // Always record (requires ffmpeg)
video: 'retain-on-failure',   // Only save videos when tests fail (recommended)
video: 'on-first-retry',      // Only record on retries
```

**Recommended:** `retain-on-failure` - saves disk space while keeping videos for debugging failures.

### System Requirements for E2E Testing

```bash
# Required packages
doas pkg_add chromium ffmpeg

# Verify installations
chrome --version    # Should show Chromium version
ffmpeg -version     # Should show ffmpeg 8.x with codecs
```

### Running E2E Tests

```bash
# Build and run all tests
npm run test:e2e

# Run without rebuilding
npm run test:e2e:nobuild

# Run with UI (interactive mode)
npm run test:e2e:ui

# Run with debugging
npm run test:e2e:debug

# View test report
npm run test:e2e:report
```

### Troubleshooting E2E Tests

#### "Executable doesn't exist at .../ffmpeg-linux"

**Cause:** ffmpeg symlink not created.

**Solution:**
```bash
mkdir -p ~/.cache/ms-playwright/ffmpeg-1011
ln -sf /usr/local/bin/ffmpeg ~/.cache/ms-playwright/ffmpeg-1011/ffmpeg-linux
```

#### "ERROR: Patchright does not support ffmpeg on linux-x64"

**Cause:** This is expected - Patchright doesn't distribute ffmpeg for Linux.

**Solution:** Use the symlink approach above instead of `patchright install ffmpeg`.

#### Tests fail to start Chrome

**Cause:** Chromium not installed or wrong path.

**Solution:**
```bash
doas pkg_add chromium
which chrome  # Should show /usr/local/bin/chrome
```

#### Patchright patches not applied

**Cause:** npm install was run but postinstall hook didn't execute.

**Solution:**
```bash
npm run postinstall
# Should show: "✅ Successfully applied X patches for openbsd"
```

### Why System ffmpeg Works

OpenBSD's ffmpeg (version 8.x) includes all codecs Playwright needs:

- **Video:** H.264 (libx264), H.265 (libx265), VP8/VP9 (libvpx), AV1 (libaom, libdav1d, libsvtav1)
- **Audio:** AAC, Opus, Vorbis, MP3 (libmp3lame)
- **Containers:** MP4, WebM, Matroska

This is maintained by the OpenBSD ports team and kept up-to-date, making it more reliable than bundled binaries.

## Related Files

- `scripts/postinstall-openbsd-natives.js` - Automated patch and binary installer for Tailwind
- `scripts/patch-patchright-openbsd.js` - Patchright platform patches for E2E testing
- `playwright.config.ts` - E2E test configuration
- `~/.local/lib/node-native-openbsd/` - Storage for pre-built Tailwind binaries
- `~/.cache/ms-playwright/` - Patchright cache directory

## Technical Background

### Why Tailwind v4 Needs Native Binaries

Tailwind CSS v4 architecture:

- **v3:** PostCSS plugin (pure JavaScript, slow)
- **v4:** Rust-based engine (compiled native code, ~100x faster)

The Rust code is compiled to platform-specific shared libraries (`.so` on BSD/Linux, `.dylib` on macOS, `.dll` on Windows) and wrapped as Node.js native addons (`.node` files) using NAPI-RS.

### Why .so → .node Renaming

Node.js expects native addons with `.node` extension. The Rust build produces:

- `liblightningcss_node.so` (Rust/Cargo naming)
- `libtailwind_oxide.so` (Rust/Cargo naming)

NAPI-RS loader expects:

- `<package>.<platform>-<arch>.node` (Node.js naming)

The copy step renames and moves these files to match Node.js conventions.

### NAPI-RS Loading Mechanism

1. Package's `index.js` checks `process.platform` and `process.arch`
2. Constructs filename: `<package>.${platform}-${arch}.node`
3. Attempts `require('./constructed-filename')`
4. If not found, tries fallback platforms
5. If all fail, throws "Cannot find native binding"

Our patch adds step 2.5: "If BSD, load `.openbsd-x64.node`"
