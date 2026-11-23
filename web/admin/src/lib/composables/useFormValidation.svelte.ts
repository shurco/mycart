import { validateFields, validators, type ValidationRule } from '$lib/utils/validation'
import { MIN_NAME_LENGTH, MIN_SLUG_LENGTH, ERROR_MESSAGES } from '$lib/constants/validation'

export function useFormValidation() {
  let errors = $state<Record<string, string>>({})

  function validate(data: Record<string, any>, rules: ValidationRule[]): boolean {
    errors = validateFields(data, rules)
    return Object.keys(errors).length === 0
  }

  function validateNameAndSlug(data: { name: string; slug: string }): boolean {
    return validate(data, [
      { field: 'name', ...validators.minLength(MIN_NAME_LENGTH, ERROR_MESSAGES.NAME_TOO_SHORT) },
      { field: 'slug', ...validators.minLength(MIN_SLUG_LENGTH, ERROR_MESSAGES.SLUG_TOO_SHORT) }
    ])
  }

  function clearErrors() {
    errors = {}
  }

  function setError(field: string, message: string) {
    errors[field] = message
  }

  return {
    get errors() {
      return errors
    },
    validate,
    validateNameAndSlug,
    clearErrors,
    setError
  }
}
