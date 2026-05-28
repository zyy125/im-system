export interface ApiEnvelope<T> {
  code: string
  message: string
  data: T | null
}

export interface ApiErrorPayload {
  code: string
  message: string
  data: null
}

export class ApiError extends Error {
  readonly code: string
  readonly status: number

  constructor(payload: { code: string; message: string; status: number }) {
    super(payload.message)
    this.name = 'ApiError'
    this.code = payload.code
    this.status = payload.status
  }
}
