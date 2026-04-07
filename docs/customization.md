# Design Customization and Deployment

## Changing the Site Design

### Frontend Structure

The site frontend is located in the `web/site/` directory and uses:
- **SvelteKit** - framework for creating SPA
- **TailwindCSS v4** - for styling
- **TypeScript** - for type safety

### Main Files for Customization

#### 1. Global Styles

The `web/site/src/app.css` file contains global styles and TailwindCSS configuration:

```css
@import 'tailwindcss';
@plugin "@tailwindcss/forms";

/* Additional styles for product description */
.prod_desc > * + * {
  margin-top: 0.75em;
}
```

Here you can:
- Change the color scheme via TailwindCSS
- Add custom CSS classes
- Configure typography

#### 2. Components

Main components are located in `web/site/src/lib/components/`:

- **Header.svelte** - site header with logo, social networks, and cart
- **Footer.svelte** - site footer with page links
- **ProductCard.svelte** - product card
- **CookieConsent.svelte** - GDPR-compliant cookie consent banner
- **NotFoundPage.svelte** - 404 error page component
- **Overlay.svelte** - overlay component for loading and errors

Example of modifying Header:

```svelte
<!-- web/site/src/lib/components/Header.svelte -->
<header class="bg-white">
  <!-- Change bg-white to desired color, e.g., bg-blue-50 -->
  <div class="mx-auto flex h-16 max-w-screen-xl items-center gap-8 px-4 sm:px-6 lg:px-8">
    <!-- Your custom content -->
  </div>
</header>
```

#### 3. Layout

The main layout is located in `web/site/src/lib/layouts/MainLayout.svelte`. Here you can change the overall page structure.

#### 4. Pages

Pages are located in `web/site/src/routes/`:
- `+page.svelte` - home page
- `[slug]/+page.svelte` - content pages
- `products/[slug]/+page.svelte` - product page
- `cart/+page.svelte` - cart

### Design Modification Process

1. **Development Mode**

   Start the development server to view changes in real-time:

   ```bash
   # Start mycart server (in one terminal)
   ./mycart serve

   # Start frontend dev server (in another terminal)
   cd web/site
   bun run dev
   ```

   The site will be available at `http://localhost:5273`

2. **Making Changes**

   - Modify components in `web/site/src/lib/components/`
   - Modify styles in `web/site/src/app.css`
   - Modify pages in `web/site/src/routes/`

3. **Building for Production**

   After making changes, build the frontend:

   ```bash
   cd web/site
   bun run build
   ```

   The built files will be in `web/site/build/` and will be automatically embedded into the binary on the next mycart build.

4. **Rebuilding mycart**

   If you're using a compiled binary, rebuild it:

   ```bash
   go build -o mycart ./cmd/main.go
   ```

### Customization via TailwindCSS

You can configure colors, fonts, and other parameters via TailwindCSS. Create a `web/site/tailwind.config.js` file if it doesn't exist:

```javascript
/** @type {import('tailwindcss').Config} */
export default {
  content: ['./src/**/*.{html,js,svelte,ts}'],
  theme: {
    extend: {
      colors: {
        primary: {
          50: '#eff6ff',
          500: '#3b82f6',
          700: '#1d4ed8',
        },
      },
    },
  },
  plugins: [],
}
```

Then use these colors in components:

```svelte
<div class="bg-primary-500 text-white">
  Custom color
</div>
```

## Deployment on a Separate Server with Nginx

### Step 1: Build Frontend

```bash
cd web/site
bun install
bun run build
```

### Step 2: Configure Nginx

Create Nginx configuration `/etc/nginx/sites-available/mycart`:

```nginx
server {
    listen 80;
    server_name yourdomain.com;
    
    # Maximum upload file size
    client_max_body_size 20M;
    
    # Gzip compression
    gzip on;
    gzip_vary on;
    gzip_min_length 1024;
    gzip_types text/plain text/css text/xml text/javascript application/x-javascript application/xml+rss application/json;

    # Frontend static files
    root /path/to/mycart/web/site/build;
    index index.html;

    # SPA routing - all other requests to index.html
    location / {
        try_files $uri $uri/ /index.html;
    }

    # Static files
    location /assets {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}
```

### Step 3: Activate Configuration

```bash
sudo ln -s /etc/nginx/sites-available/mycart /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

### HTTPS Configuration

For production, it's recommended to use HTTPS. Use Let's Encrypt:

```bash
sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx -d yourdomain.com
```

Certbot will automatically update the Nginx configuration to use HTTPS.

## Complete Setup Example on a Separate Server

### Directory Structure

```
/var/www/
└── mycart-frontend/    # Built frontend
    └── build/           # Static files
```

### Deployment Commands

```bash
# 1. Create directory
sudo mkdir -p /var/www/mycart-frontend

# 2. Build frontend
cd web/site
bun install
bun run build
sudo cp -r build/* /var/www/mycart-frontend/

# 3. Set permissions
sudo chown -R www-data:www-data /var/www/mycart-frontend

# 4. Configure Nginx
sudo nano /etc/nginx/sites-available/mycart
# (paste configuration from section above)

# 5. Activate Nginx configuration
sudo ln -s /etc/nginx/sites-available/mycart /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

After this, your site will be available at the specified domain.
