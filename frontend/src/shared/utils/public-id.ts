export const normalizePublicIdInput = (value: string) =>
  value.replace(/[^\d]/g, '')

export const parsePublicId = (value: string) => {
  if (!/^\d+$/.test(value)) {
    return null
  }

  const parsed = Number(value)
  return Number.isSafeInteger(parsed) && parsed > 0 ? parsed : null
}
