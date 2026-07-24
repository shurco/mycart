#!/usr/bin/env node
/**
 * OpenBSD Native Module Patcher
 *
 * Patches @tailwindcss/oxide and lightningcss to support OpenBSD/FreeBSD
 * by adding platform detection and copying pre-built native binaries.
 *
 * Run after: npm install
 * Auto-run: Add "postinstall": "node scripts/postinstall-openbsd-natives.js" to package.json
 */

const fs = require('fs');
const path = require('path');
const os = require('os');

// Only run on BSD platforms
const platform = os.platform();
if (platform !== 'openbsd' && platform !== 'freebsd') {
  console.log('ℹ️  Not a BSD platform - skipping native module patches');
  process.exit(0);
}

console.log(`🔧 Patching native modules for ${platform}...`);

const NATIVE_LIBS_DIR = path.join(os.homedir(), '.local/lib/node-native-openbsd');
const WEB_DIRS = ['web/admin', 'web/site'];

let patchCount = 0;

// Patch @tailwindcss/oxide
function patchOxide(webDir) {
  const oxidePath = path.join(webDir, 'node_modules/@tailwindcss/oxide');
  const indexPath = path.join(oxidePath, 'index.js');
  const binaryPath = path.join(oxidePath, 'tailwindcss-oxide.openbsd-x64.node');

  if (!fs.existsSync(indexPath)) {
    console.log(`  ⏭️  Skipping ${webDir}/oxide - not installed`);
    return;
  }

  // Copy binary
  const sourceBinary = path.join(NATIVE_LIBS_DIR, 'tailwindcss-oxide.openbsd-x64.node');
  if (fs.existsSync(sourceBinary)) {
    fs.copyFileSync(sourceBinary, binaryPath);
    console.log(`  ✓ Copied oxide binary to ${webDir}`);
  } else {
    console.log(`  ⚠️  Binary not found: ${sourceBinary}`);
    console.log(`     Run: cargo build in tailwindcss repo first`);
    return;
  }

  // Patch index.js
  let content = fs.readFileSync(indexPath, 'utf8');

  if (content.includes("process.platform === 'openbsd'")) {
    console.log(`  ℹ️  ${webDir}/oxide already patched`);
    return;
  }

  const unsupportedPattern = /(\} else \{\s+loadErrors\.push\(new Error\(`Unsupported OS: \$\{process\.platform\}, architecture: \$\{process\.arch\}`\)\)\s+\})/;
  const openbsdSection = `} else if (process.platform === 'openbsd' || process.platform === 'freebsd') {
    if (process.arch === 'x64') {
      try {
        return require('./tailwindcss-oxide.openbsd-x64.node')
      } catch (e) {
        loadErrors.push(e)
      }
    } else {
      loadErrors.push(new Error(\`Unsupported architecture on BSD: \${process.arch}\`))
    }
  $1`;

  if (content.match(unsupportedPattern)) {
    content = content.replace(unsupportedPattern, openbsdSection);
    fs.writeFileSync(indexPath, content);
    console.log(`  ✓ Patched ${webDir}/oxide index.js`);
    patchCount++;
  }
}

// Copy lightningcss binary
function patchLightningCSS(webDir) {
  // Try flattened location first, then nested
  let lightningPath = path.join(webDir, 'node_modules/lightningcss');
  if (!fs.existsSync(lightningPath)) {
    lightningPath = path.join(webDir, 'node_modules/@tailwindcss/node/node_modules/lightningcss');
  }
  const binaryPath = path.join(lightningPath, 'lightningcss.openbsd-x64.node');

  if (!fs.existsSync(lightningPath)) {
    console.log(`  ⏭️  Skipping ${webDir}/lightningcss - not installed`);
    return;
  }

  const sourceBinary = path.join(NATIVE_LIBS_DIR, 'lightningcss.openbsd-x64.node');
  if (fs.existsSync(sourceBinary)) {
    fs.copyFileSync(sourceBinary, binaryPath);
    console.log(`  ✓ Copied lightningcss binary to ${webDir}`);
    patchCount++;
  } else {
    console.log(`  ⚠️  Binary not found: ${sourceBinary}`);
    console.log(`     Run: cargo build in lightningcss repo first`);
  }
}

// Apply patches
for (const webDir of WEB_DIRS) {
  if (fs.existsSync(webDir)) {
    patchOxide(webDir);
    patchLightningCSS(webDir);
  }
}

if (patchCount > 0) {
  console.log(`\n✅ Successfully applied ${patchCount} patches for ${platform}`);
} else {
  console.log('\n⚠️  No patches applied - binaries may not be available');
  console.log('   Save binaries to ~/.local/lib/node-native-openbsd/ first');
}
