<script lang="ts">
  import FormInput from '$lib/components/form/Input.svelte'
  import FormSelect from '$lib/components/form/Select.svelte'
  import FormToggle from '$lib/components/form/Toggle.svelte'
  import FormButton from '$lib/components/form/Button.svelte'
  import { translate } from '$lib/i18n'
  import { sslModes, type InstallDatabase } from './installDatabase.svelte'

  // Reactive translation function
  let t = $derived($translate)

  interface Props {
    db: InstallDatabase
    ontest: () => void
  }

  let { db, ontest }: Props = $props()
</script>

<fieldset class="space-y-4 rounded border border-gray-200 p-4">
  <legend class="px-2"><h3>{t('install.database')}</h3></legend>

  {#if db.locked}
    <p class="text-gray-500">
      {t('install.databaseLocked')}
      <code class="ml-1 text-xs break-all">{db.lockedSummary}</code>
    </p>
  {:else}
    <FormSelect
      id="database-driver"
      title={t('install.databaseDriver')}
      ico="circle-stack"
      options={{ sqlite: t('install.databaseSQLite'), postgres: t('install.databasePostgres') }}
      bind:value={db.driver}
    />

    {#if db.driver === 'postgres'}
      <FormToggle id="database-dsn-mode" label={t('install.useConnectionString')} bind:value={db.useDSN} />

      {#if db.useDSN}
        <FormInput
          id="database-dsn"
          type="text"
          title={t('install.connectionString')}
          ico="link"
          bind:value={db.dsn}
          placeholder="postgres://user:password@host:5432/mycart?sslmode=require"
        />
      {:else}
        <FormInput id="pg-host" type="text" title={t('install.host')} ico="server" bind:value={db.pgHost} />
        <FormInput id="pg-port" type="text" title={t('install.port')} ico="hashtag" bind:value={db.pgPort} />
        <FormInput
          id="pg-database"
          type="text"
          title={t('install.databaseName')}
          ico="circle-stack"
          bind:value={db.pgDatabase}
        />
        <FormInput id="pg-user" type="text" title={t('install.user')} ico="user" bind:value={db.pgUser} />
        <FormInput
          id="pg-password"
          type="password"
          title={t('install.dbPassword')}
          ico="lock-closed"
          bind:value={db.pgPassword}
        />
        <FormSelect
          id="pg-sslmode"
          title={t('install.sslMode')}
          ico="shield-check"
          options={sslModes}
          bind:value={db.pgSSLMode}
        />
      {/if}

      {#if db.dbError}
        <span class="error text-red-500">{db.dbError}</span>
      {/if}

      <FormButton
        type="button"
        variant="secondary"
        name={db.testing ? t('install.testingConnection') : t('install.testConnection')}
        disabled={db.testing}
        class="w-full"
        onclick={ontest}
      />
    {/if}
  {/if}
</fieldset>
