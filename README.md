<p align="center">
    <a href="#" target="_blank" rel="noopener">
        <img src="https://github.com/shurco/mycart/blob/main/.github/media/banner.png" alt="myCart - shopping-cart in 1 file" />
    </a>
</p>


<a href="https://github.com/shurco/mycart/releases"><img src="https://img.shields.io/github/v/release/shurco/mycart?sort=semver&label=Release&color=651FFF"></a>
<a href="https://goreportcard.com/report/github.com/shurco/mycart"><img src="https://goreportcard.com/badge/github.com/shurco/mycart"></a>
<a href="https://www.codefactor.io/repository/github/shurco/mycart"><img src="https://www.codefactor.io/repository/github/shurco/mycart/badge" alt="CodeFactor" /></a>
<a href="https://github.com/shurco/mycart/actions/workflows/release.yml"><img src="https://github.com/shurco/mycart/actions/workflows/release.yml/badge.svg"></a>
<a href="https://github.com/shurco/mycart/blob/master/LICENSE"><img src="https://img.shields.io/badge/License-MIT-yellow.svg"></a>

> [!Important]
> This repository is being renamed.  
>The project was originally published under the name litecart. I recently received a trademark-related claim regarding the use of this name. To avoid confusion and potential legal issues, the project will continue under a new name.  
>The codebase itself is not changing. Only the project name, repository name, package identifiers, and related references will be updated.  
>This repository will remain available for some time with redirects and notes pointing to the new location so existing users have time to migrate.  
>If your setup depends on the current repository name, please update your references once the rename is complete.  
>Thank you to everyone who has used the project and contributed feedback.  

> [!NOTE]
> **Disclaimer:** Not affiliated with any similarly named projects or brands.
> This is an independent open source project licensed under the MIT License.


## 🛒&nbsp;&nbsp;What is myCart?

myCart is an open source shopping-cart in 1 file of embedded database (SQLite), convenient dashboard UI and simple site.
Formerly known as **litecart** (legacy project name kept here for discoverability in search).

> [!WARNING]
> Current major version is zero (`v0.x.x`) to accommodate rapid development and fast iteration while getting early feedback from users. Please keep in mind that myCart is still under active development and therefore full backward compatibility is not guaranteed before reaching v1.0.0.

### Video Example
![Example](https://github.com/shurco/mycart/blob/main/.github/media/demo.gif)

### Admin Panel Screenshots
<p align="center">
  <img src="https://github.com/shurco/mycart/blob/main/.github/media/screenshots/products.png" width="270">
  <img src="https://github.com/shurco/mycart/blob/main/.github/media/screenshots/product-edit.png" width="270">
  <img src="https://github.com/shurco/mycart/blob/main/.github/media/screenshots/carts.png" width="270">
  <img src="https://github.com/shurco/mycart/blob/main/.github/media/screenshots/pages.png" width="270">
  <img src="https://github.com/shurco/mycart/blob/main/.github/media/screenshots/settings.png" width="270">
</p>


## 🏆&nbsp;&nbsp;Features

🚀 **Simple and Fast**: Enjoy a one-click installation process that gets your store up and running quickly, saving you time and effort.  

💰 **Support for Popular Payment Systems**: Accept payments seamlessly with support for popular payment systems, ensuring a smooth checkout experience for your customers.  

🔑 **Sell Files and License Keys**: Whether you're selling digital files or license keys, myCart has you covered, providing flexibility in the types of products you can offer.  

⚙️ **Lightweight and Efficient**: myCart utilizes SQLite as its embedded database, eliminating the need for heavy databases like MySQL, PostgreSQL, or MongoDB. This results in a lightweight website that performs exceptionally well.  

☁️ **Easily Customizable**: Modify and customize your myCart website effortlessly to match your branding and unique requirements, making it truly your own.  

🧞‍♂️ **Convenient Administration Panel**: With a user-friendly dashboard UI, myCart offers a hassle-free administration panel, allowing you to manage your store, inventory, and orders with ease.  

⚡️ **Hardware Compatibility**: Whether you're running myCart on a powerful server or a modest hardware setup, rest assured that it will work seamlessly, providing a consistent shopping experience for your customers.  

🔒 **Built-in HTTPS Support**: Prioritizing security, myCart comes with built-in support for HTTPS, ensuring the safety of your customers' data.

🆓 **Free Products Support**: Offer free products to your customers by setting the product price to 0. Free products are automatically processed without requiring payment system integration, making it perfect for free downloads, samples, or promotional content.

🌐 **Multi-language Support**: Built-in internationalization (i18n) support allows you to create multilingual stores. By default, myCart includes support for English and Chinese languages. The language switcher is available in both the admin panel and the public site, making it easy to manage content in multiple languages and provide a localized shopping experience for your customers.


## ⬇️&nbsp;&nbsp;Installation

`mycart` is engineered for easy installation and operation, requiring just a single command from your terminal. Besides the conventional installation method, `mycart` can also be set up and operated via HomeBrew, Docker, or any other container orchestration tools like Docker Compose, Docker Swarm, Rancher, or Kubernetes.

#### <img width="20" src="/.github/media/platforms/apple.svg">&nbsp;Install on macOS
The fastest method to install `mycart` on macOS involves using Homebrew. This will install the command-line tools and the `mycart` server as a combined executable. If you don't utilize Homebrew, adhere to the Linux instructions below for `mycart` installation.
```shell
brew install shurco/tap/mycart
```

Alternately, you can configure the tap and install the package separately:
``` shell
$ brew tap shurco/tap
$ brew install mycart
```


#### <img width="20" src="/.github/media/platforms/linux.svg">&nbsp;Install on Linux 
The most straightforward and recommended method to start using `mycart` on Unix operating systems involves installing and utilizing the `mycart` command-line tool. Execute the given command in your terminal and adhere to the instructions displayed on the screen.

```bash
curl -L https://raw.githubusercontent.com/shurco/mycart/main/scripts/install | sh
```

#### <img width="20" src="/.github/media/platforms/windows.svg">&nbsp;Install on Windows
The simplest and most recommended method to start using `mycart` on Windows is by installing and utilizing the `mycart` command-line tool. Execute the given command in your terminal and adhere to the instructions displayed on the screen.
```bash
curl -L https://raw.githubusercontent.com/shurco/mycart/main/scripts/install | sh
```
or download and unzip the [latest version](https://github.com/shurco/mycart/releases/latest) for Windows.


#### <img width="20" src="/.github/media/platforms/docker.svg">&nbsp;Run using Docker
Docker enables the management and operation of a `mycart` instance without requiring the installation of any command-line tools. The `mycart` Docker container includes all necessary command-line tools  or even for server execution.

For [Docker Hub](https://hub.docker.com/r/shurco/mycart):
```bash
docker run \
  -v ./lc_base:/lc_base \
  -v ./lc_digitals:/lc_digitals \
  -v ./lc_uploads:/lc_uploads \
  -v ./site:/site \
  --rm shurco/mycart:latest init

docker run \
  --name mycart \
  --restart unless-stopped \
  -p '8080:8080' \
  -v ./lc_base:/lc_base \
  -v ./lc_digitals:/lc_digitals \
  -v ./lc_uploads:/lc_uploads \
  -v ./site:/site \
  shurco/mycart:latest
```
or if use [Github Packages Hub](https://github.com/shurco/mycart/pkgs/container/mycart):

```bash
docker run \
  -v ./lc_base:/lc_base \
  -v ./lc_digitals:/lc_digitals \
  -v ./lc_uploads:/lc_uploads \
  -v ./site:/site \
  --rm ghcr.io/shurco/mycart:latest init

docker run \
  --name mycart \
  --restart unless-stopped \
  -p '8080:8080' \
  -v ./lc_base:/lc_base \
  -v ./lc_digitals:/lc_digitals \
  -v ./lc_uploads:/lc_uploads \
  -v ./site:/site \
  ghcr.io/shurco/mycart:latest
```

#### <img width="20" src="/.github/media/platforms/docker.svg">&nbsp;Run using Docker Compose
Docker Compose provides a convenient way to manage multiple containers and services. The project includes several Docker Compose configurations for different use cases.

**Production Setup** (`docker/docker-compose.yml`):
This configuration includes the myCart application and an Nginx reverse proxy:

```bash
cd docker
docker-compose up -d
```

This setup includes:
- **mycart**: The main application container with all required volumes
- **nginx**: Reverse proxy server that listens on port 80 and forwards requests to myCart

The Nginx configuration is located in `docker/nginx/nginx.conf` and can be customized as needed.

**Development Setup** (`docker/docker-compose_dev.yml`):
For local development, you can use the development configuration which includes MailHog for email testing:

```bash
cd docker
docker-compose -f docker-compose.yml -f docker-compose_dev.yml up -d
```

This adds:
- **mailhog**: Email testing tool accessible at `http://localhost:8025` for viewing sent emails

**Combined Usage**:
To run both production and development services together:

```bash
cd docker
docker-compose -f docker-compose.yml -f docker-compose_dev.yml up -d
```

**Initialization**:
Before starting the services, initialize the application:

```bash
docker-compose run --rm mycart init
```

**Stopping Services**:
```bash
docker-compose down
```

#### <img width="20" src="/.github/media/platforms/k8s.svg">&nbsp;Run using Kubernetes
An example manifest for running on Kubernetes can be found in the `/k8s/` folder (thanks <a href="https://github.com/vuisme" target="_blank">@vuisme</a>)


## 🔄&nbsp;&nbsp;Migrating from litecart

If you are upgrading from a version that was published under the old name **litecart**, see the **[Migration Guide](./docs/migration-from-litecart.md)** for step-by-step instructions covering binary, Docker, Docker Compose, Kubernetes, Homebrew, and Go module updates. Your data and database are fully compatible — no schema migration is required.

## ⬇️&nbsp;&nbsp;Updating
> [!WARNING]
> Before any update, be sure to make a backup of the *./lc_base* folder and the *./site* folder.

#### Update on macOS / Linux / Windows
The easiest way to update `mycart` to the latest version is to execute the command:

```bash
./mycart update
```

If there were changes in the database structure during the update, it is necessary to perform migration. To do this, you need to run the command from the `mycart` folder:
```bash
./mycart migrate
```


#### <img width="20" src="/.github/media/platforms/docker.svg">&nbsp; Update using Docker
Our mantra is to make updating a seamless experience. Simply download the new image and launch the container as you normally would. For example, if use [Docker Hub](https://hub.docker.com/r/shurco/mycart):

```bash
docker stop mycart
docker pull shurco/mycart:latest # download new image
docker rename mycart mycart-backup # do image backup
docker run \
  --name mycart \
  --restart unless-stopped \
  -p '8080:8080' \
  -v ./lc_base:/lc_base \
  -v ./lc_digitals:/lc_digitals \
  -v ./lc_uploads:/lc_uploads \
  -v ./site:/site \
  shurco/mycart:latest
```

If there were changes in the database structure during the update, it is necessary to perform migration. To do this, you need to run the command from the `mycart` folder:
```bash
docker run \
-v ./lc_base:/lc_base \
-v ./site:/site \
--rm shurco/mycart migrate
```

#### <img width="20" src="/.github/media/platforms/k8s.svg">&nbsp;Run using Kubernetes
An example manifest for running on Kubernetes can be found in the `/k8s/` folder (thanks <a href="https://github.com/vuisme" target="_blank">@vuisme</a>)

## 🚀&nbsp;&nbsp;Getting started
Getting started with `mycart` is as easy as starting up the `mycart` server

Default run for Linux/macOS:
```bash
./mycart serve
```

For Windows:
```
mycart.exe serve
```

When launched for the first time, necessary folders will be created in the directory with the executable file. The default links for access are:  
- [http://localhost:8080](http://localhost:8080) - website  
- [http://localhost:8080/_/](http://localhost:8080/_/) - control panel  

If you need to run on a different port, use the flag `--http`:
```
./mycart serve --http 0.0.0.0:8088
```

> [!NOTE]
> Ports <= 1024 are privileged ports. You can't use them unless you're root or have the explicit permission to use them. See this answer for an explanation or wikipedia or something you trust more. Use:
> **sudo setcap 'cap_net_bind_service=+ep' /path_to/mycart**

## 📚&nbsp;&nbsp;Commands
Usage:
```
./mycart [command] [flags]
```

Available commands:
```
init        Creating the basic structure
migrate     Migrate on the latest version of database schema
serve       Starts the web server (default to 0.0.0.0:8080)
update      Updating the application to the latest version
```

Global flags `./mycart [flags]`:
```
-h, --help      help for mycart
-v, --version   version for mycart
```

Serve flags `./mycart serve [flags]`:
```
--http string    server address (default "0.0.0.0:8080")
--https string   https server address (auto TLS)
--no-site        disable create site
```

## 🏦&nbsp;&nbsp;Adding payment systems
#### Stripe
Stripe is a popular payment system that allows you to accept online payments from customers. It provides various tools and APIs for processing payments, including the ability to accept credit and debit cards, digital wallets, and bank transfers. Stripe ensures payment security, currency processing, and support for various payment methods.

To obtain the Secret key in Stripe, follow these steps:

1. Log in to your <a href="https://dashboard.stripe.com" target="_blank">Stripe account</a> on the official Stripe website. If you don't have an account, <a href="https://dashboard.stripe.com/register" target="_blank">register</a> for one.
2. In the top right corner, select the <a href="https://dashboard.stripe.com/developers" target="_blank">Developers section</a>.
3. In the dropdown menu, choose "<a href="https://dashboard.stripe.com/apikeys" target="_blank">API Keys</a>".
4. In the "Standard keys" section, you will find your "Secret key".

> [!WARNING]
> Please note that the "Secret key" is confidential information that should be kept secure.


#### PayPal
PayPal is an online payment system that allows individuals and businesses to send and receive money over the internet. It enables payments for goods and services, as well as transfers between users. PayPal provides a secure and convenient way to make electronic payments.

To obtain a Client ID and Secret Key for using the PayPal API, you need to follow these steps:

1. To use the API, you will need a PayPal business account.
2. Go to the <a href="https://developer.paypal.com/" target="_blank">PayPal Developer</a> website and sign in with your PayPal business account credentials.
3. In the Dashboard, find the "My Apps & Credentials" section and create a new application by clicking the "Create App" button.
4. On the application page, you will see your Client ID. It will be visible immediately after creating the application. To see the Secret Key, click on the "Show" button under the "Secret" label.

> [!WARNING]
> Please note that the "Secret key" is confidential information that should be kept secure.


#### SpectroCoin
<a href="https://spectrocoin.com/en/invite?referralId=b2n87748" target="_blank">SpectroCoin</a> is a payment system and cryptocurrency wallet that allows users to send and receive payments in various currencies, including cryptocurrencies such as Bitcoin, Ethereum, and others. It also offers currency exchange operations between different currencies and the ability to deposit and withdraw funds to bank accounts. <a href="https://spectrocoin.com/en/invite?referralId=b2n87748" target="_blank">SpectroCoin</a> ensures the security of payments and cryptocurrency storage, as well as offering additional features such as debit cards.

To obtain a "Merchant ID", "Project (API) ID" and "Private key" in <a href="https://spectrocoin.com/en/invite?referralId=b2n87748" target="_blank">SpectroCoin</a>, follow these steps:

1. Register on <a href="https://spectrocoin.com/en/invite?referralId=b2n87748" target="_blank">SpectroCoin</a> if you don't have an account yet.
2. Log in to your <a href="https://spectrocoin.com/en/invite?referralId=b2n87748" target="_blank">SpectroCoin</a> account.
3. Go to the "Business" section in the navigation menu.
4. Navigate to the "New project" section in the navigation menu.
5. Fill in the project name and make sure to enable the "Public key" section. A window with a "Private key" will appear, copy and save it. You can activate other options if needed.
6. After filling in the details, you will be redirected to the projects page. Go to the created project and in the header, copy the "Merchant ID" and "Project (API) ID".

> [!WARNING]
> Please note that creating a project may require you to complete the verification process for your <a href="https://spectrocoin.com/en/invite?referralId=b2n87748" target="_blank">SpectroCoin</a> account.  
> Please note that the "Private key" is confidential information that should be kept secure.


#### Coinbase
Coinbase Commerce is a cryptocurrency payment platform that allows businesses to accept payments in various cryptocurrencies including Bitcoin, Ethereum, Litecoin, and many others. It provides a simple and secure way to integrate crypto payments into your store.

To obtain an API Key for using the Coinbase Commerce API, follow these steps:

1. Create or log in to your <a href="https://commerce.coinbase.com" target="_blank">Coinbase Commerce</a> account.
2. Navigate to the **Settings** section.
3. Under the **API Keys** tab, click **Create an API Key**.
4. Copy the generated API Key and save it securely.

> [!WARNING]
> Please note that the "API Key" is confidential information that should be kept secure.


#### Dummy Payment
Dummy Payment is a built-in payment provider that comes pre-configured with myCart. It is designed for processing free products (products with a price of $0) and does not require any external payment system integration or API keys.

**How it works:**
- Automatically activated when a customer's cart contains only free products (total amount = $0)
- No payment processing occurs - orders are immediately marked as paid
- Customers only need to provide their email address to complete the checkout
- Perfect for free downloads, samples, promotional content, and lead generation

**Key features:**
- **No configuration required**: Works out of the box, no setup needed
- **No API keys needed**: Unlike other payment providers, Dummy Payment doesn't require any external accounts or credentials
- **Instant processing**: Orders are processed immediately without waiting for payment confirmation
- **Full feature support**: All standard features work with Dummy Payment, including email delivery, digital file downloads, license keys, and webhooks

**When to use:**
- Selling free products or samples
- Offering promotional content
- Collecting email addresses for lead generation
- Testing the checkout process during development

> [!NOTE]
> Dummy Payment is only used for carts containing exclusively free products. If a cart contains both free and paid products, customers must use a regular payment system (Stripe, PayPal, or SpectroCoin) to complete the purchase.


## 🆓&nbsp;&nbsp;Free Products

myCart supports free products, allowing you to offer digital content, samples, or promotional materials at no cost to your customers.

### Creating Free Products

To create a free product:

1. In the admin panel, navigate to **Products** and create a new product
2. Set the **Amount** field to `0` (zero)
3. The product will automatically display as "free" in both the admin panel and on the website

### How Free Products Work

- **Automatic Processing**: When a customer adds only free products to their cart (total amount = 0), the checkout process automatically uses the built-in **Dummy Payment** provider (see [Dummy Payment](#dummy-payment-built-in) section for details)
- **No Payment Required**: Free products bypass all external payment system integrations - customers can complete their purchase with just an email address
- **Instant Access**: After checkout, customers immediately receive access to free products via email, just like paid products
- **Mixed Carts**: If a cart contains both free and paid products, customers must use a regular payment system (Stripe, PayPal, or SpectroCoin) to complete the purchase

### Use Cases

Free products are perfect for:
- Free downloads and samples
- Promotional content and giveaways
- Trial versions of digital products
- Free resources and documentation
- Lead generation (collecting email addresses)

### Technical Details

- Free products are identified by `amount = 0` in the database
- The **Dummy Payment** provider is automatically selected for carts with `amountTotal = 0`
- All standard features work with free products: email delivery, digital file downloads, license keys, and webhooks
- Free products are included in order history and cart management just like paid products
- No external payment processing occurs - orders are immediately marked as paid when using Dummy Payment


## 🧩&nbsp;&nbsp;For developers
The backend is developed in Go language. The frontend admin panel operates on SvelteKit and TailwindCSS.  

There are a number of scripts (in the ./scripts folder) that simplify development:  
`./scripts/golang` - Installs or updates a previously installed version of go (if needed).  
`./scripts/migration` - Helps to work with migrations. For instance, the `./scripts/migration dev up` command will apply new migrations from folder ./migrations, then implement the migrations from folder ./fixtures.  
`./scripts/sqlite` - Optimizes the existing database.  
`./scripts/tools` - Sets up the necessary environment for development (if needed).  
`./scripts/webscripts` - For the basic site, updates vue3 to the latest version.  
`./scripts/clear` - Removing hung golang or vite processes.  

> [!NOTE]
> I recommend running the `./scripts/migration dev up` command. It will add test data to the database, which makes it easier to work with. For example, it will create products, transfer test images and create a test user for access to the admin panel:  
> login - user@mail.com  
> password - Pass123

#### Admin panel (frontend)
To develop the web interface of the admin panel, you need to start the myCart server (for example, execute the command from the project root `go run ./cmd/main.go serve`).
All the code is located in the folder ./web/admin. The command `cd ./web/admin && bun run dev` will start the development server for the admin panel web interface. By default, it will be available at http://localhost:5173/_/.

#### Base site (frontend)
To develop the web interface of the base site, you need to start the myCart server (for example, execute the command from the project root `go run ./cmd/main.go serve`).  
Run `./scripts/webscripts` to download and install the necessary JavaScript libraries into the folder ./web/site/public/assets/js/.
To update the styles, it is necessary to execute the command `cd ./web/site && bun run build`.  
If you actively change styles, you can run the command `cd ./web/site && bun run dev`. It will monitor changes in files and automatically update the style file.

#### Customization and Deployment
For detailed information on how to customize the site design and deploy it on a separate server with Nginx, see [Customization and Deployment Guide](./docs/customization.md).

## 🗺️&nbsp;&nbsp;ToDo
`mycart` has a [roadmap](https://github.com/users/shurco/projects/2) and I try to work on issues in specific order and such PRs often come in out of nowhere and skew all initial planning with tedious back-and-forth communication.

- [x] Product in the form of files
- [x] Product in the form of license keys
- [ ] Product returned via API to another site (example license keys)
- [x] <a href="#stripe">Payment Stripe</a>
- [x] <a href="#paypal">Payment PayPal</a>
- [ ] Payment Square
- [ ] Payment Adyen
- [ ] Payment Checkout
- [ ] Payment via Webhook
- [x] <a href="#spectrocoin">Support for payment using crypto (SpectroCoin)</a>
- [x] <a href="#coinbase">Coinbase Commerce crypto payments</a>
- [x] Support WebHook (<a href="https://github.com/msalbrain" target="_blank">@nicksnyder</a> in <a href="https://github.com/shurco/mycart/pull/61" target="_blank">#61</a>)
- [x] <a href="#dummy-payment">Dummy Payment</a> (<a href="https://github.com/majiayu000" target="_blank">@majiayu000</a> in <a href="https://github.com/shurco/mycart/pull/261" target="_blank">#261</a>)


## 👍&nbsp;&nbsp;Contribute

If you want to say **thank you** and/or support the active development of `mycart`:

1. Add a [GitHub Star](https://github.com/shurco/mycart/stargazers) to the project.
2. Tweet about the project [on your Twitter](https://twitter.com/intent/tweet?text=%F0%9F%9B%92%20myCart%20-%20shopping-cart%20in%201%20file%20on%20%23Go%20https%3A%2F%2Fgithub.com%2Fshurco%2Fmycart).
3. Write a review or tutorial on [Medium](https://medium.com/), [Dev.to](https://dev.to/) or personal blog.
4. Support the project by donating a [cup of coffee](https://github.com/sponsors/shurco).

You can learn more about how you can contribute to this project in the [contribution guide](https://github.com/shurco/mycart/blob/master/.github/CONTRIBUTING.md).
