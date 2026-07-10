// @vitest-environment jsdom

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { clearSession, login, refreshSession, restoreSession } from './api'
import type { ApiEnvelope, AuthResponse, StoredSession, UserResponse } from './types'

const storageKey = 'cove.auth.session.v1'

function jsonResponse<T>(envelope: ApiEnvelope<T>, status = 200): Response {
  return new Response(JSON.stringify(envelope), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function authResponse(overrides: Partial<AuthResponse> = {}): AuthResponse {
  return {
    user_id: 'user-1',
    username: 'linhai',
    email: 'linhai@example.com',
    access_token: 'access-1',
    refresh_token: 'refresh-1',
    ...overrides,
  }
}

function storedSession(overrides: Partial<StoredSession> = {}): StoredSession {
  return {
    accessToken: 'old-access',
    refreshToken: 'old-refresh',
    user: {
      id: 'user-1',
      username: 'linhai',
      nickname: null,
      email: 'linhai@example.com',
      avatar: null,
    },
    ...overrides,
  }
}

beforeEach(() => {
  clearSession()
  vi.restoreAllMocks()
})

describe('auth session API', () => {
  it('stores tokens and basic user details after login', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse<AuthResponse>({ code: 0, message: 'ok', data: authResponse() }),
    )
    vi.stubGlobal('fetch', fetchMock)

    const session = await login({ login: 'linhai', password: 'secret123' })

    expect(session.accessToken).toBe('access-1')
    expect(session.user.username).toBe('linhai')
    expect(JSON.parse(window.localStorage.getItem(storageKey) ?? '{}')).toEqual(session)
  })

  it('deduplicates concurrent refresh requests and rotates both tokens', async () => {
    window.localStorage.setItem(storageKey, JSON.stringify(storedSession()))
    let resolveResponse: ((response: Response) => void) | undefined
    const pendingResponse = new Promise<Response>((resolve) => {
      resolveResponse = resolve
    })
    const fetchMock = vi.fn().mockReturnValue(pendingResponse)
    vi.stubGlobal('fetch', fetchMock)

    const firstRefresh = refreshSession()
    const secondRefresh = refreshSession()
    expect(firstRefresh).toBe(secondRefresh)

    resolveResponse?.(
      jsonResponse<AuthResponse>({
        code: 0,
        message: 'ok',
        data: authResponse({ access_token: 'access-2', refresh_token: 'refresh-2' }),
      }),
    )
    const [firstSession, secondSession] = await Promise.all([firstRefresh, secondRefresh])

    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(firstSession?.accessToken).toBe('access-2')
    expect(secondSession?.refreshToken).toBe('refresh-2')
  })

  it('refreshes once after a 401 and then restores the current user', async () => {
    window.localStorage.setItem(storageKey, JSON.stringify(storedSession()))
    const user: UserResponse = {
      id: 'user-1',
      username: 'linhai',
      nickname: '林海',
      email: 'linhai@example.com',
      avatar: null,
      created_at: '2026-07-10T00:00:00Z',
    }
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse({ code: 40101, message: '请先登录' }, 401))
      .mockResolvedValueOnce(
        jsonResponse<AuthResponse>({
          code: 0,
          message: 'ok',
          data: authResponse({ access_token: 'access-2', refresh_token: 'refresh-2' }),
        }),
      )
      .mockResolvedValueOnce(jsonResponse<UserResponse>({ code: 0, message: 'ok', data: user }))
    vi.stubGlobal('fetch', fetchMock)

    const restored = await restoreSession()

    expect(fetchMock).toHaveBeenCalledTimes(3)
    expect(restored?.accessToken).toBe('access-2')
    expect(restored?.user.nickname).toBe('林海')
  })

  it('clears the stored session when refresh fails', async () => {
    window.localStorage.setItem(storageKey, JSON.stringify(storedSession()))
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(jsonResponse({ code: 40101, message: '登录状态已失效' }, 401)),
    )

    await expect(refreshSession()).rejects.toThrow('登录状态已失效')
    expect(window.localStorage.getItem(storageKey)).toBeNull()
  })
})
