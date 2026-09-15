<script lang="ts">
  import { goto } from '$app/navigation'
  import { base } from '$app/paths'
  import { onDestroy, onMount } from 'svelte'
  import { browser } from '$app/environment'
  import Blank from '$lib/layouts/Blank.svelte'
  import FormInput from '$lib/components/form/Input.svelte'
  import FormSelect from '$lib/components/form/Select.svelte'
  import FormToggle from '$lib/components/form/Toggle.svelte'
  import FormButton from '$lib/components/form/Button.svelte'
  import { apiGet, apiPost } from '$lib/utils/api'
  import { showMessage } from '$lib/utils/message'
  import LanguageSelect from '$lib/components/LanguageSelect.svelte'
  import { translate } from '$lib/i18n'

  // Reactive translation function
  let t = $derived($translate)

  // Map backend error messages to translation keys
  function translateError(message: string): string {
    if (message.includes('already holds an installed cart')) {
      return t('install.errors.databaseAlreadyInstalled')
    }
    if (message.includes('connect to the selected database')) {
      return t('install.errors.databaseConnectionFailed')
    }
    if (message.includes('inspect the selected database')) {
      return t('install.errors.databaseInspectionFailed')
    }
    if (message.includes('migrations failed')) {
      return t('install.errors.migrationsFailed')
    }
    // Return original message if no translation found
    return message
  }

  let email = $state('')
  let password = $state('')
  let domain = $state('')
  let emailError = $state('')
  let passwordError = $state('')
  let domainError = $state('')

  // Database selection -------------------------------

  type Driver = 'sqlite' | 'postgres'

  let driver = $state<Driver>('sqlite')
  // A database the server already configured has already been decided; the
  // wizard installs into it rather than offering to move the cart elsewhere.
  let locked = $state(false)
  let lockedSummary = $state('')

  // PostgreSQL connection, either as a single string or as its parts.
  let useDSN = $state(false)
  let dsn = $state('')
  let pgHost = $state('localhost')
  let pgPort = $state('5432')
  let pgDatabase = $state('mycart')
  let pgUser = $state('mycart')
  let pgPassword = $state('')
  let pgSSLMode = $state('disable')

  let dbError = $state('')
  let tested = $state(false)
  let testing = $state(false)

  const sslModes = ['disable', 'require', 'verify-ca', 'verify-full']

  // Any change to the connection invalidates a previous successful test.
  $effect(() => {
    void driver
    void useDSN
    void dsn
    void pgHost
    void pgPort
    void pgDatabase
    void pgUser
    void pgPassword
    void pgSSLMode
    tested = false
  })

  // A locked database is the one the server is already running on: there is no
  // connection to test, so the button must not wait for one.
  let installableNow = $derived(locked || driver === 'sqlite' || tested)

  function buildDSN(): string {
    if (useDSN) {
      return dsn.trim()
    }
    const params = new URLSearchParams()
    if (pgSSLMode) params.set('sslmode', pgSSLMode)
    const user = encodeURIComponent(pgUser)
    const secret = encodeURIComponent(pgPassword)
    const credentials = pgPassword ? `${user}:${secret}` : user
    return `postgres://${credentials}@${pgHost}:${pgPort}/${pgDatabase}?${params.toString()}`
  }

  function databasePayload() {
    if (driver === 'sqlite') {
      return { driver: 'sqlite' }
    }
    return { driver: 'postgres', dsn: buildDSN() }
  }

  async function handleTestConnection() {
    dbError = ''
    testing = true
    try {
      const res = await apiPost('/api/install/db/test', databasePayload())
      if (res?.success) {
        tested = true
        showMessage(t('install.connectionOk'), 'connextSuccess')
      } else {
        tested = false
        const errorMsg = res?.message ? translateError(res.message) : t('install.connectionFailed')
        dbError = errorMsg
        showMessage(errorMsg, 'connextError')
      }
    } finally {
      testing = false
    }
  }

  let redirectTimer: ReturnType<typeof setTimeout> | undefined
  onDestroy(() => {
    if (redirectTimer !== undefined) clearTimeout(redirectTimer)
  })

  function validateEmail(value: string) {
    if (!value) {
      return t('install.emailRequired')
    }
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value)) {
      return t('install.emailInvalid')
    }
    return ''
  }

  function validatePassword(value: string) {
    if (!value) {
      return t('install.passwordRequired')
    }
    if (value.length < 6) {
      return t('install.passwordMinLength')
    }
    if (value.length > 72) {
      return t('install.passwordMaxLength')
    }
    return ''
  }

  function validateDomain(value: string) {
    if (!value) {
      return t('install.domainRequired')
    }
    // Basic domain validation
    const domainRegex = /^([a-z0-9]+(-[a-z0-9]+)*\.)+[a-z]{2,}$/i
    if (!domainRegex.test(value) && value !== 'localhost' && !value.startsWith('localhost:')) {
      return t('install.domainInvalid')
    }
    return ''
  }

  async function handleSubmit() {
    emailError = validateEmail(email)
    passwordError = validatePassword(password)
    domainError = validateDomain(domain)
    dbError = ''

    if (emailError || passwordError || domainError) {
      return
    }

    if (!locked && driver === 'postgres' && !tested) {
      dbError = t('install.testConnectionFirst')
      return
    }

    const body: Record<string, unknown> = { email, password, domain }
    if (!locked) {
      body.database = databasePayload()
    }

    try {
      const res = await apiPost(`/api/install`, body)
      if (res?.success) {
        showMessage(t('install.installedSuccessfully'), 'connextSuccess')
        // Redirect to signin page after successful installation
        redirectTimer = setTimeout(() => {
          goto(`${base}/signin`)
        }, 1000)
      } else {
        const errorMsg = res?.message ? translateError(res.message) : t('install.installationFailed')
        showMessage(errorMsg, 'connextError')
      }
    } catch (error) {
      showMessage(t('install.networkError'), 'connextError')
    }
  }

  onMount(async () => {
    // Set default domain from current location if in browser
    if (browser) {
      const url = new URL(window.location.href)
      domain = url.origin.replace(/^https?:\/\//, '')
    }

    const status = await apiGet('/api/install/status')
    const database = status?.result?.database
    if (database) {
      // The wizard may only choose the built-in default or a database it has
      // just connected to. The status endpoint masks the password, so the DSN it
      // reports cannot be submitted back — re-sending it would install into a
      // different database than the running one.
      locked = !!database.locked || database.source !== 'default'
      lockedSummary = database.dsn || database.driver
    }
  })
</script>

<Blank>
  <div class="content-center">
    <div class="header">
      <h1>🛒 {t('install.title')} myCart</h1>
      <p>{t('install.configureCart')}</p>
    </div>
    <form
      onsubmit={(e) => {
        e.preventDefault()
        handleSubmit()
      }}
      class="mx-auto mt-8 mb-0 max-w-md space-y-4"
    >
      <FormInput
        id="email"
        type="email"
        title={t('install.email')}
        ico="at-symbol"
        error={emailError}
        bind:value={email}
      />
      <FormInput
        id="password"
        type="password"
        title={t('install.password')}
        ico="finger-print"
        error={passwordError}
        bind:value={password}
      />
      <FormInput
        id="domain"
        type="text"
        title={t('install.domain')}
        ico="glob-alt"
        error={domainError}
        bind:value={domain}
        placeholder="example.com"
      />

      <fieldset class="space-y-4 rounded border border-gray-200 p-4">
        <legend class="px-2"><h3>{t('install.database')}</h3></legend>

        {#if locked}
          <p class="text-gray-500">
            {t('install.databaseLocked')}
            <code class="ml-1 text-xs break-all">{lockedSummary}</code>
          </p>
        {:else}
          <FormSelect
            id="database-driver"
            title={t('install.databaseDriver')}
            ico="circle-stack"
            options={{ sqlite: t('install.databaseSQLite'), postgres: t('install.databasePostgres') }}
            bind:value={driver}
          />

          {#if driver === 'postgres'}
            <FormToggle id="database-dsn-mode" label={t('install.useConnectionString')} bind:value={useDSN} />

            {#if useDSN}
              <FormInput
                id="database-dsn"
                type="text"
                title={t('install.connectionString')}
                ico="link"
                bind:value={dsn}
                placeholder="postgres://user:password@host:5432/mycart?sslmode=require"
              />
            {:else}
              <FormInput id="pg-host" type="text" title={t('install.host')} ico="server" bind:value={pgHost} />
              <FormInput id="pg-port" type="text" title={t('install.port')} ico="hashtag" bind:value={pgPort} />
              <FormInput
                id="pg-database"
                type="text"
                title={t('install.databaseName')}
                ico="circle-stack"
                bind:value={pgDatabase}
              />
              <FormInput id="pg-user" type="text" title={t('install.user')} ico="user" bind:value={pgUser} />
              <FormInput
                id="pg-password"
                type="password"
                title={t('install.dbPassword')}
                ico="lock-closed"
                bind:value={pgPassword}
              />
              <FormSelect
                id="pg-sslmode"
                title={t('install.sslMode')}
                ico="shield-check"
                options={sslModes}
                bind:value={pgSSLMode}
              />
            {/if}

            {#if dbError}
              <span class="error text-red-500">{dbError}</span>
            {/if}

            <FormButton
              type="button"
              variant="secondary"
              name={testing ? t('install.testingConnection') : t('install.testConnection')}
              disabled={testing}
              class="w-full"
              onclick={handleTestConnection}
            />
          {/if}
        {/if}
      </fieldset>

      <!-- Same reason and same place as on the sign-in page: setup is exactly
           where someone may be facing a language they cannot read. -->
      <div class="flex items-center justify-between gap-4">
        <FormButton
          type="submit"
          name={t('install.installButton')}
          variant="primary"
          ico="arrow-right"
          disabled={!installableNow}
        />
        <LanguageSelect />
      </div>
    </form>
  </div>
</Blank>
