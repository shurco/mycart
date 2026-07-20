# CLAUDE.md

Development and build instructions for myCart, with emphasis on local-only builds.

---

## Local Build Process

All components (backend + frontends) build locally on OpenBSD and other platforms.

### Prerequisites

- **Go**: 1.26+
- **Node.js**: 24+
- **npm** or **bun**: For frontend dependencies
- **Rust/Cargo**: Required for building native modules on OpenBSD (Tailwind CSS v4)

### Quick Build (All Platforms)

**Note:** The npm scripts use `bun` commands. If you don't have bun installed, use `npx vite build` instead of `npm run build` (see Troubleshooting section).

```bash
# Backend
go build -o mycart ./cmd/main.go

# Admin frontend
cd web/admin
npm install
npm run build        # requires bun, or use: npx vite build

# Site frontend
cd web/site
npm install
npm run build        # requires bun, or use: npx vite build
```

---

## OpenBSD-Specific Build Instructions

Tailwind CSS v4 and lightningcss don't ship pre-built binaries for OpenBSD. Native modules must be built from source.

### First-Time Setup on OpenBSD

#### 1. Build lightningcss Native Module

```bash
cd /tmp
git clone --depth 1 https://github.com/parcel-bundler/lightningcss.git
cd lightningcss
npm install --ignore-scripts
cargo build --release -p lightningcss_node

# Copy to both frontends
cp target/release/liblightningcss_node.so \
   /path/to/mycart/web/admin/node_modules/lightningcss/lightningcss.openbsd-x64.node

cp target/release/liblightningcss_node.so \
   /path/to/mycart/web/site/node_modules/lightningcss/lightningcss.openbsd-x64.node
```

#### 2. Build Tailwind Oxide Native Module

```bash
cd /tmp
git clone --depth 1 https://github.com/tailwindlabs/tailwindcss.git
cd tailwindcss/crates/node
cargo build --release

# Copy to both frontends
cp ../../target/release/libtailwind_oxide.so \
   /path/to/mycart/web/admin/node_modules/@tailwindcss/oxide/tailwindcss-oxide.openbsd-x64.node

cp ../../target/release/libtailwind_oxide.so \
   /path/to/mycart/web/site/node_modules/@tailwindcss/oxide/tailwindcss-oxide.openbsd-x64.node
```

#### 3. Patch @tailwindcss/oxide for OpenBSD

Add OpenBSD platform support to both `web/admin/node_modules/@tailwindcss/oxide/index.js` and `web/site/node_modules/@tailwindcss/oxide/index.js`.

Insert after the FreeBSD section (around line 262):

```javascript
  } else if (process.platform === 'openbsd') {
    if (process.arch === 'x64') {
      try {
        return require('./tailwindcss-oxide.openbsd-x64.node')
      } catch (e) {
        loadErrors.push(e)
      }
    } else {
      loadErrors.push(new Error(`Unsupported architecture on OpenBSD: ${process.arch}`))
    }
  } else if (process.platform === 'linux') {
```

#### 4. Build Frontends

```bash
cd web/admin
npm install
npx vite build

cd ../site
npm install
npx vite build
```

### Automation Script (OpenBSD)

Save as `scripts/build-openbsd-deps.sh`:

```bash
#!/bin/sh
set -e

MYCART_ROOT="$(pwd)"
TMP_DIR="/tmp/mycart-build-$$"

echo "Building native modules for OpenBSD..."

# Build lightningcss
mkdir -p "$TMP_DIR"
cd "$TMP_DIR"
git clone --depth 1 https://github.com/parcel-bundler/lightningcss.git
cd lightningcss
npm install --ignore-scripts
cargo build --release -p lightningcss_node

echo "Installing lightningcss to admin..."
cp target/release/liblightningcss_node.so \
   "$MYCART_ROOT/web/admin/node_modules/lightningcss/lightningcss.openbsd-x64.node"

echo "Installing lightningcss to site..."
cp target/release/liblightningcss_node.so \
   "$MYCART_ROOT/web/site/node_modules/lightningcss/lightningcss.openbsd-x64.node"

# Build tailwind oxide
cd "$TMP_DIR"
git clone --depth 1 https://github.com/tailwindlabs/tailwindcss.git
cd tailwindcss/crates/node
cargo build --release

echo "Installing oxide to admin..."
cp ../../target/release/libtailwind_oxide.so \
   "$MYCART_ROOT/web/admin/node_modules/@tailwindcss/oxide/tailwindcss-oxide.openbsd-x64.node"

echo "Installing oxide to site..."
cp ../../target/release/libtailwind_oxide.so \
   "$MYCART_ROOT/web/site/node_modules/@tailwindcss/oxide/tailwindcss-oxide.openbsd-x64.node"

# Patch oxide index.js files
echo "Patching oxide for OpenBSD support..."
for dir in "$MYCART_ROOT/web/admin" "$MYCART_ROOT/web/site"; do
  INDEX_JS="$dir/node_modules/@tailwindcss/oxide/index.js"
  if ! grep -q "process.platform === 'openbsd'" "$INDEX_JS"; then
    # Create patch content
    PATCH_CONTENT="  } else if (process.platform === 'openbsd') {
    if (process.arch === 'x64') {
      try {
        return require('./tailwindcss-oxide.openbsd-x64.node')
      } catch (e) {
        loadErrors.push(e)
      }
    } else {
      loadErrors.push(new Error(\`Unsupported architecture on OpenBSD: \${process.arch}\`))
    }
  } else if (process.platform === 'linux') {"
    
    # Replace the "} else if (process.platform === 'linux') {" line
    sed -i.bak "s|  } else if (process.platform === 'linux') {|$PATCH_CONTENT|" "$INDEX_JS"
    echo "  Patched $INDEX_JS"
  fi
done

cd "$MYCART_ROOT"
rm -rf "$TMP_DIR"

echo "Done! Native modules built and installed."
echo "You can now run: npm run build in web/admin and web/site"
```

---

## Development Workflow

### Local Development (All Platforms)

**Note:** The `package.json` scripts use `bun` by default. If you don't have bun installed, see the "Running without Bun" section below.

```bash
# Terminal 1: Backend
go run ./cmd serve

# Terminal 2: Admin frontend (with hot reload)
cd web/admin
bun run dev  # or see below if bun not installed

# Terminal 3: Site frontend (with hot reload)
cd web/site
bun run dev  # or see below if bun not installed
```

#### Running without Bun

If you don't have bun installed, you have two options:

**Option 1: Install Bun (Recommended)**
```bash
curl -fsSL https://bun.sh/install | bash
# Restart your terminal, then use npm/bun run dev as normal
```

**Option 2: Use npx directly**
```bash
# Admin
cd web/admin
npx vite dev

# Site
cd web/site
npx vite dev

# Build
npx vite build

# Other commands
npx svelte-kit sync
```

This bypasses the npm scripts and runs vite directly.

Access:
- Admin panel: `http://localhost:5173/_/` (dev) or `http://localhost:8080/_/` (prod)
- Storefront: `http://localhost:5174/` (dev) or `http://localhost:8080/` (prod)

### Production Build

```bash
# Build all components
./scripts/build-all.sh  # or manually:

go build -o mycart ./cmd/main.go
cd web/admin && npm run build && cd ../..
cd web/site && npm run build && cd ../..

# Single binary includes embedded frontends
./mycart serve
```

---

## Platform Notes

### Linux / macOS / Windows

Pre-built native modules are available. Standard npm/bun workflow applies:

```bash
cd web/admin && npm install && npm run build
cd web/site && npm install && npm run build
```

### OpenBSD

Requires building native modules from source (see above). After first-time setup:

```bash
# When node_modules are deleted, rebuild native modules
./scripts/build-openbsd-deps.sh

# Then normal build
cd web/admin && npm install && npm run build
cd web/site && npm install && npm run build
```

### FreeBSD

May work with FreeBSD binaries (untested). If not, follow OpenBSD instructions.

---

## Troubleshooting

### "sh: bun: not found" when running `npm run dev`

The package.json scripts are configured to use bun. You have two options:

**Option 1: Install Bun**
```bash
curl -fsSL https://bun.sh/install | bash
# Restart terminal
```

**Option 2: Run vite directly**
```bash
npx vite dev        # for dev
npx vite build      # for build
npx vite preview    # for preview
```

**Note:** We keep bun in package.json for existing users. Use npx as a workaround if you prefer not to install bun.

### "Cannot find module 'lightningcss.openbsd-x64.node'"

The native module wasn't built. Run:
```bash
./scripts/build-openbsd-deps.sh
```

### "unsupported TLS program header"

FreeBSD binaries are incompatible with OpenBSD. Must build from source.

### Build works but `npm install` deletes native modules

Native modules must be rebuilt after `npm install`. Add to your workflow:
```bash
npm install && ./scripts/build-openbsd-deps.sh
```

Or use `npm ci` instead of `npm install` to preserve node_modules.

---

## CI/CD Considerations

For automated builds:

1. **Detect platform** early in build script
2. **Cache compiled native modules** between builds (OpenBSD only)
3. **Build order**: Backend → Native modules → Frontends
4. **Artifact**: Single `mycart` binary with embedded frontends

Example build script:
```bash
#!/bin/sh
set -e

# Backend
go build -o mycart ./cmd/main.go

# Frontends (platform-aware)
if [ "$(uname)" = "OpenBSD" ]; then
  ./scripts/build-openbsd-deps.sh
fi

cd web/admin && npm install && npm run build && cd ../..
cd web/site && npm install && npm run build && cd ../..

echo "Build complete: ./mycart"
```

---

## References

- **AGENTS.md**: High-level development guide
- **DEV_SETUP.md**: Initial setup for contributors
- **READY_TO_TEST.md**: Testing the PortOne payment integration
- `web/AGENTS.md`: Frontend-specific conventions
