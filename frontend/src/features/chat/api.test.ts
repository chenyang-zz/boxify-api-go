// @vitest-environment jsdom

import { beforeEach, describe, expect, it, vi } from 'vitest'

const mocks = vi.hoisted(() => ({
  authenticatedRequest: vi.fn(),
  refreshSession: vi.fn(),
}))

vi.mock('../auth/api', () => ({
  ApiError: class ApiError extends Error {
    status: number
    constructor(status: number, message: string) {
      super(message)
      this.status = status
    }
  },
  authenticatedRequest: mocks.authenticatedRequest,
  refreshSession: mocks.refreshSession,
}))

import { consumeChatStream, listConversations, listMessages, streamChat } from './api'
import type { ChatStreamEvent } from './types'

function chunkedStream(chunks: string[]): ReadableStream<Uint8Array> {
  const encoder = new TextEncoder()
  return new ReadableStream({
    start(controller) {
      for (const chunk of chunks) {
        controller.enqueue(encoder.encode(chunk))
      }
      controller.close()
    },
  })
}

beforeEach(() => {
  vi.restoreAllMocks()
  mocks.authenticatedRequest.mockReset()
  mocks.refreshSession.mockReset()
  window.localStorage.clear()
})

describe('chat API', () => {
  it('passes conversation and message requests through authenticatedRequest', async () => {
    mocks.authenticatedRequest.mockResolvedValueOnce({ list: [{ id: 'conversation-1' }] })
    await expect(listConversations()).resolves.toEqual({ list: [{ id: 'conversation-1' }] })
    expect(mocks.authenticatedRequest).toHaveBeenNthCalledWith(1, '/api/conversation')

    mocks.authenticatedRequest.mockResolvedValueOnce({ list: [{ id: 'message-1' }] })
    await expect(listMessages('conversation / 1')).resolves.toEqual({ list: [{ id: 'message-1' }] })
    expect(mocks.authenticatedRequest).toHaveBeenNthCalledWith(
      2,
      '/api/conversation/conversation%20%2F%201/messages',
    )
  })

  it('parses split SSE frames, multiline data, comments, and error events', async () => {
    const events: ChatStreamEvent[] = []
    const stream = chunkedStream([
      ': ping\n\n',
      'event: meta\ndata: {"type":"meta","conversation_id":"c1",',
      '"title":"新对话"}\n\n',
      'event: token\ndata: {"type":"token",\ndata: "text":"你',
      '好"}\n\n',
      'event: error\ndata: {"type":"error","content":"失败"}\n\n',
    ])

    await consumeChatStream(stream, (event) => events.push(event))

    expect(events).toEqual([
      { type: 'meta', conversation_id: 'c1', title: '新对话' },
      { type: 'token', text: '你好' },
      { type: 'error', content: '失败' },
    ])
  })

  it('refreshes once after an initial 401 and streams the retried response', async () => {
    window.localStorage.setItem(
      'cove.auth.session.v1',
      JSON.stringify({ accessToken: 'old', refreshToken: 'refresh', user: { id: 'u1' } }),
    )
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(new Response('', { status: 401 }))
      .mockImplementationOnce(() => {
        return Promise.resolve(
          new Response(chunkedStream(['event: done\ndata: {"type":"done","text":"m1"}\n\n']), {
            status: 200,
            headers: { 'Content-Type': 'text/event-stream' },
          }),
        )
      })
    vi.stubGlobal('fetch', fetchMock)
    mocks.refreshSession.mockImplementation(async () => {
      window.localStorage.setItem(
        'cove.auth.session.v1',
        JSON.stringify({ accessToken: 'new', refreshToken: 'refresh-2', user: { id: 'u1' } }),
      )
      return { accessToken: 'new' }
    })
    const events: ChatStreamEvent[] = []

    await streamChat({ message: '你好' }, new AbortController().signal, (event) => events.push(event))

    expect(mocks.refreshSession).toHaveBeenCalledTimes(1)
    expect(fetchMock).toHaveBeenCalledTimes(2)
    expect(new Headers(fetchMock.mock.calls[1][1]?.headers).get('Authorization')).toBe('Bearer new')
    expect(events).toEqual([{ type: 'done', text: 'm1' }])
  })

  it('propagates aborts without converting them to a network error', async () => {
    window.localStorage.setItem(
      'cove.auth.session.v1',
      JSON.stringify({ accessToken: 'token', refreshToken: 'refresh', user: { id: 'u1' } }),
    )
    const controller = new AbortController()
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation((_url: string, init: RequestInit) => {
        controller.abort()
        return Promise.reject(init.signal?.reason ?? new DOMException('Aborted', 'AbortError'))
      }),
    )

    await expect(streamChat({ message: '你好' }, controller.signal, vi.fn())).rejects.toMatchObject({
      name: 'AbortError',
    })
  })
})
