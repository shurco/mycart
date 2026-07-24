#!/usr/bin/env node
/**
 * Patchright OpenBSD/FreeBSD Platform Support Patcher
 *
 * Automatically applies platform detection patches to Patchright after npm install.
 * Only runs on OpenBSD and FreeBSD - silently skips on other platforms.
 *
 * Patches Applied:
 * 1. Cache directory detection (3 locations)
 * 2. User agent platform mapping
 * 3. Window insets configuration
 * 4. Chrome channel executable path
 */

const fs = require('fs');
const path = require('path');
const os = require('os');

// Only run on BSD platforms
const platform = os.platform();
if (platform !== 'openbsd' && platform !== 'freebsd') {
  // Silent exit on Linux/macOS/Windows - patches not needed
  process.exit(0);
}

console.log(`🔧 Detected ${platform} - applying Patchright patches...`);

const bundlePath = path.join(__dirname, '../node_modules/patchright-core/lib/coreBundle.js');

// Check if Patchright is installed
if (!fs.existsSync(bundlePath)) {
  console.log('ℹ️  Patchright not installed - skipping patches');
  process.exit(0);
}

let content = fs.readFileSync(bundlePath, 'utf8');
let patchCount = 0;

// Patch 1: Cache directory detection (3 locations)
// Lines ~29986, ~52808, ~70503
const cachePattern1 = /if \(process\.platform === "linux"\)\s*return process\.env\.XDG_CACHE_HOME/g;
const cacheReplacement1 = 'if (process.platform === "linux" || process.platform === "openbsd" || process.platform === "freebsd")\n        return process.env.XDG_CACHE_HOME';
const cacheMatches1 = content.match(cachePattern1);
if (cacheMatches1) {
  content = content.replace(cachePattern1, cacheReplacement1);
  patchCount += cacheMatches1.length;
  console.log(`  ✓ Patched cache directory detection (${cacheMatches1.length} locations)`);
}

const cachePattern2 = /if \(process\.platform === "linux"\)\s*localCacheDir = process\.env\.XDG_CACHE_HOME/g;
const cacheReplacement2 = 'if (process.platform === "linux" || process.platform === "openbsd" || process.platform === "freebsd")\n        localCacheDir = process.env.XDG_CACHE_HOME';
const cacheMatches2 = content.match(cachePattern2);
if (cacheMatches2) {
  content = content.replace(cachePattern2, cacheReplacement2);
  patchCount += cacheMatches2.length;
  console.log(`  ✓ Patched daemon cache directory (${cacheMatches2.length} location)`);
}

// Patch 2: User agent platform detection (two separate patches)
// Line ~11386 - Platform check
const userAgentPlatformPattern = /} else if \(process\.platform === "linux"\) \{/g;
const userAgentPlatformReplacement = '} else if (process.platform === "linux" || process.platform === "openbsd" || process.platform === "freebsd") {';
if (content.match(userAgentPlatformPattern)) {
  content = content.replace(userAgentPlatformPattern, userAgentPlatformReplacement);
  patchCount++;
  console.log('  ✓ Patched user agent platform check');
}

// Line ~11392 - OS identifier fallback
const userAgentOsPattern = /osIdentifier = "linux";(\s*\})/g;
const userAgentOsReplacement = 'osIdentifier = process.platform === "openbsd" ? "openbsd" : process.platform === "freebsd" ? "freebsd" : "linux";$1';
if (content.match(userAgentOsPattern)) {
  content = content.replace(userAgentOsPattern, userAgentOsReplacement);
  patchCount++;
  console.log('  ✓ Patched user agent OS identifier');
}

// Patch 3: Window insets configuration
// Line ~38888
const insetsPattern = /else if \(process\.platform === "linux"\)\s*insets = \{ width: 8, height: 85 \};/;
const insetsReplacement = 'else if (process.platform === "linux" || process.platform === "openbsd" || process.platform === "freebsd")\n              insets = { width: 8, height: 85 };';

if (content.match(insetsPattern)) {
  content = content.replace(insetsPattern, insetsReplacement);
  patchCount++;
  console.log('  ✓ Patched window insets configuration');
}

// Patch 4: Chrome channel executable path
// Line ~30350+
const chromeChannelPattern = /"linux": "\/opt\/google\/chrome\/chrome",(\s*)"darwin": "\/Applications\/Google Chrome\.app/;
const chromeChannelReplacement = `"linux": "/opt/google/chrome/chrome",
          "openbsd": "/usr/local/bin/chrome",
          "freebsd": "/usr/local/bin/chrome",$1"darwin": "/Applications/Google Chrome.app`;

if (content.match(chromeChannelPattern)) {
  content = content.replace(chromeChannelPattern, chromeChannelReplacement);
  patchCount++;
  console.log('  ✓ Patched Chrome channel executable path');
}

// Patch 5: Platform detection for ffmpeg support
// Line ~7900+ in calculatePlatform() - Add OpenBSD/FreeBSD before final return
const platformCalcPattern = /(if \(platform === "win32"\)\s*return \{ hostPlatform: "win64", isOfficiallySupportedPlatform: true \};)\s*(return \{ hostPlatform: "<unknown>", isOfficiallySupportedPlatform: false \};)/;
const platformCalcReplacement = `$1
  if (platform === "openbsd" || platform === "freebsd") {
    const archSuffix = "-" + import_os3.default.arch();
    return { hostPlatform: "linux-x64", isOfficiallySupportedPlatform: false };
  }
  $2`;

if (content.match(platformCalcPattern)) {
  content = content.replace(platformCalcPattern, platformCalcReplacement);
  patchCount++;
  console.log('  ✓ Patched platform detection for BSD systems');
}

// Patch 6: ffmpeg platform mapping - Add BSD entries
// Line ~29616+ in ffmpeg dependencies object
const ffmpegPattern = /("ffmpeg": \{\s*"<unknown>": void 0,\s*"linux-x64": \["ffmpeg-linux"\],)/;
const ffmpegReplacement = `$1
        "openbsd-x64": ["ffmpeg-linux"],
        "freebsd-x64": ["ffmpeg-linux"],`;

if (content.match(ffmpegPattern)) {
  content = content.replace(ffmpegPattern, ffmpegReplacement);
  patchCount++;
  console.log('  ✓ Patched ffmpeg platform mapping for BSD');
}

// Write patched content back
if (patchCount > 0) {
  fs.writeFileSync(bundlePath, content, 'utf8');
  console.log(`\n✅ Successfully applied ${patchCount} patches for ${platform}`);
  console.log('   Patchright is now ready for E2E testing!\n');
} else {
  console.log('⚠️  No patches applied - file may already be patched or has changed');
  console.log('   Tests may still work if patches were applied manually\n');
}
