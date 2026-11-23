export type Driver = 'sqlite' | 'postgres'

export const sslModes = ['disable', 'require', 'verify-ca', 'verify-full']

/** The database half of the install wizard: what the operator picked and whether it was tested. */
export class InstallDatabase {
  driver = $state<Driver>('sqlite')
  // A database the server already configured has already been decided; the
  // wizard installs into it rather than offering to move the cart elsewhere.
  locked = $state(false)
  lockedSummary = $state('')

  // PostgreSQL connection, either as a single string or as its parts.
  useDSN = $state(false)
  dsn = $state('')
  pgHost = $state('localhost')
  pgPort = $state('5432')
  pgDatabase = $state('mycart')
  pgUser = $state('mycart')
  pgPassword = $state('')
  pgSSLMode = $state('disable')

  dbError = $state('')
  tested = $state(false)
  testing = $state(false)

  // A locked database is the one the server is already running on: there is no
  // connection to test, so the button must not wait for one.
  installableNow = $derived(this.locked || this.driver === 'sqlite' || this.tested)

  constructor() {
    // Any change to the connection invalidates a previous successful test.
    $effect(() => {
      void this.driver
      void this.useDSN
      void this.dsn
      void this.pgHost
      void this.pgPort
      void this.pgDatabase
      void this.pgUser
      void this.pgPassword
      void this.pgSSLMode
      this.tested = false
    })
  }

  buildDSN(): string {
    if (this.useDSN) {
      return this.dsn.trim()
    }
    const params = new URLSearchParams()
    if (this.pgSSLMode) params.set('sslmode', this.pgSSLMode)
    const user = encodeURIComponent(this.pgUser)
    const secret = encodeURIComponent(this.pgPassword)
    const credentials = this.pgPassword ? `${user}:${secret}` : user
    return `postgres://${credentials}@${this.pgHost}:${this.pgPort}/${this.pgDatabase}?${params.toString()}`
  }

  payload() {
    if (this.driver === 'sqlite') {
      return { driver: 'sqlite' }
    }
    return { driver: 'postgres', dsn: this.buildDSN() }
  }
}
