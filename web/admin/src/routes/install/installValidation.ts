type Translate = (key: string) => string

// Map backend error messages to translation keys
export function translateError(message: string, t: Translate): string {
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

export function validateEmail(value: string, t: Translate) {
  if (!value) {
    return t('install.emailRequired')
  }
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value)) {
    return t('install.emailInvalid')
  }
  return ''
}

export function validatePassword(value: string, t: Translate) {
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

export function validateDomain(value: string, t: Translate) {
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
