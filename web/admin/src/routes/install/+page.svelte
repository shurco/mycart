<script lang="ts">
  import { goto } from '$app/navigation'
  import { base } from '$app/paths'
  import { onDestroy, onMount } from 'svelte'
  import { browser } from '$app/environment'
  import Blank from '$lib/layouts/Blank.svelte'
  import FormInput from '$lib/components/form/Input.svelte'
  import FormButton from '$lib/components/form/Button.svelte'
  import { apiGet, apiPost } from '$lib/utils/api'
  import { showMessage } from '$lib/utils/message'
  import LanguageSelect from '$lib/components/LanguageSelect.svelte'
  import { translate } from '$lib/i18n'
  import DatabaseFields from './DatabaseFields.svelte'
  import { InstallDatabase } from './installDatabase.svelte'
  import { translateError, validateEmail, validatePassword, validateDomain } from './installValidation'

  // Reactive translation function
  let t = $derived($translate)

  let email = $state('')
  let password = $state('')
  let domain = $state('')
  let emailError = $state('')
  let passwordError = $state('')
  let domainError = $state('')

  const db = new InstallDatabase()

  async function handleTestConnection() {
    db.dbError = ''
    db.testing = true
    try {
      const res = await apiPost('/api/install/db/test', db.payload())
      if (res?.success) {
        db.tested = true
        showMessage(t('install.connectionOk'), 'connextSuccess')
      } else {
        db.tested = false
        const errorMsg = res?.message ? translateError(res.message, t) : t('install.connectionFailed')
        db.dbError = errorMsg
        showMessage(errorMsg, 'connextError')
      }
    } finally {
      db.testing = false
    }
  }

  let redirectTimer: ReturnType<typeof setTimeout> | undefined
  onDestroy(() => {
    if (redirectTimer !== undefined) clearTimeout(redirectTimer)
  })

  async function handleSubmit() {
    emailError = validateEmail(email, t)
    passwordError = validatePassword(password, t)
    domainError = validateDomain(domain, t)
    db.dbError = ''

    if (emailError || passwordError || domainError) {
      return
    }

    if (!db.locked && db.driver === 'postgres' && !db.tested) {
      db.dbError = t('install.testConnectionFirst')
      return
    }

    const body: Record<string, unknown> = { email, password, domain }
    if (!db.locked) {
      body.database = db.payload()
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
        const errorMsg = res?.message ? translateError(res.message, t) : t('install.installationFailed')
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
      db.locked = !!database.locked || database.source !== 'default'
      db.lockedSummary = database.dsn || database.driver
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

      <DatabaseFields {db} ontest={handleTestConnection} />

      <!-- Same reason and same place as on the sign-in page: setup is exactly
           where someone may be facing a language they cannot read. -->
      <div class="flex items-center justify-between gap-4">
        <FormButton
          type="submit"
          name={t('install.installButton')}
          variant="primary"
          ico="arrow-right"
          disabled={!db.installableNow}
        />
        <LanguageSelect />
      </div>
    </form>
  </div>
</Blank>
