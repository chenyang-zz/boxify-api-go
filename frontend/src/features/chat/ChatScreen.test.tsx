// @vitest-environment jsdom

import { act, cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { StoredSession } from '../auth/types'
import type { ChatStreamEvent } from './types'

const mocks = vi.hoisted(() => ({
  deleteConversation: vi.fn(),
  getAgentConfig: vi.fn(),
  listConversations: vi.fn(),
  listMessages: vi.fn(),
  renameConversation: vi.fn(),
  streamChat: vi.fn(),
}))

vi.mock('./api', () => ({
  deleteConversation: mocks.deleteConversation,
  getAgentConfig: mocks.getAgentConfig,
  listConversations: mocks.listConversations,
  listMessages: mocks.listMessages,
  renameConversation: mocks.renameConversation,
  streamChat: mocks.streamChat,
}))

import { ChatScreen } from './ChatScreen'

const session: StoredSession = {
  accessToken: 'access',
  refreshToken: 'refresh',
  user: {
    id: 'user-1',
    username: 'linhai',
    nickname: '林海',
    email: 'linhai@example.com',
    avatar: null,
  },
}

beforeEach(() => {
  vi.restoreAllMocks()
  sessionStorage.clear()
  mocks.listConversations.mockReset().mockResolvedValue({ list: [] })
  mocks.listMessages.mockReset().mockResolvedValue({ list: [] })
  mocks.getAgentConfig.mockReset().mockResolvedValue({ enable_knowledge: false })
  mocks.renameConversation.mockReset()
  mocks.deleteConversation.mockReset()
  mocks.streamChat.mockReset()
  Element.prototype.scrollIntoView = vi.fn()
  Element.prototype.scrollTo = vi.fn()
  window.scrollTo = vi.fn()
})

afterEach(() => {
  cleanup()
})

describe('ChatScreen', () => {
  it('tracks the visual viewport so the keyboard cannot scroll the whole app away', async () => {
    const listeners = new Map<string, EventListener>()
    const viewport = {
      height: 844,
      width: 390,
      offsetTop: 0,
      addEventListener: vi.fn((type: string, listener: EventListener) => listeners.set(type, listener)),
      removeEventListener: vi.fn((type: string) => listeners.delete(type)),
    }
    vi.stubGlobal('visualViewport', viewport)

    const { container, unmount } = render(<ChatScreen session={session} onLogout={vi.fn()} />)
    const app = container.querySelector<HTMLElement>('.chat-app')
    expect(container.querySelector('.message-scroll')?.classList.contains('message-scroll--empty')).toBe(true)
    expect(app?.dataset.keyboardOpen).toBe('false')
    expect(app?.style.getPropertyValue('--chat-keyboard-height')).toBe('0px')
    expect(app?.style.getPropertyValue('--chat-content-shift')).toBe('0px')

    viewport.height = 516
    viewport.offsetTop = 286
    act(() => listeners.get('resize')?.(new Event('resize')))
    expect(app?.dataset.keyboardOpen).toBe('true')
    expect(app?.style.getPropertyValue('--chat-keyboard-height')).toBe('328px')
    expect(app?.style.getPropertyValue('--chat-content-shift')).toBe('148px')

    viewport.height = 844
    act(() => listeners.get('resize')?.(new Event('resize')))
    expect(app?.dataset.keyboardOpen).toBe('false')
    expect(app?.style.getPropertyValue('--chat-content-shift')).toBe('0px')
    await waitFor(() => {
      expect(Element.prototype.scrollTo).toHaveBeenCalledWith({ top: 0 })
    })

    unmount()
    expect(document.documentElement.classList.contains('chat-document')).toBe(false)
    expect(viewport.removeEventListener).toHaveBeenCalledWith('resize', expect.any(Function))
    vi.unstubAllGlobals()
  })

  it('shows the personalized empty state and toggles the mobile drawer', async () => {
    const user = userEvent.setup()
    render(<ChatScreen session={session} onLogout={vi.fn()} />)

    expect(await screen.findByRole('heading', { name: '你好，林海' })).toBeTruthy()
    const drawer = screen.getByRole('complementary')
    expect(drawer.classList.contains('chat-drawer--open')).toBe(false)

    await user.click(screen.getByRole('button', { name: '打开会话列表' }))
    expect(drawer.classList.contains('chat-drawer--open')).toBe(true)
    await user.click(screen.getAllByRole('button', { name: '关闭会话列表' })[1])
    expect(drawer.classList.contains('chat-drawer--open')).toBe(false)
  })

  it('uses a neutral user name when nickname is empty', async () => {
    render(
      <ChatScreen
        session={{ ...session, user: { ...session.user, nickname: null } }}
        onLogout={vi.fn()}
      />,
    )

    expect(await screen.findByRole('heading', { name: '你好，用户' })).toBeTruthy()
    expect(screen.queryByRole('heading', { name: '你好，linhai' })).toBeNull()
  })

  it('closes the account menu when the user clicks outside or presses Escape', async () => {
    const user = userEvent.setup()
    render(<ChatScreen session={session} onLogout={vi.fn()} />)
    await screen.findByRole('heading', { name: '你好，林海' })

    const trigger = screen.getByRole('button', { name: '打开账户菜单' })
    await user.click(trigger)
    expect(screen.getByRole('menu')).toBeTruthy()

    await user.click(screen.getByRole('heading', { name: '你好，林海' }))
    expect(screen.queryByRole('menu')).toBeNull()

    await user.click(trigger)
    await user.keyboard('{Escape}')
    expect(screen.queryByRole('menu')).toBeNull()
  })

  it('focuses the composer without allowing WKWebView to scroll the page', async () => {
    const user = userEvent.setup()
    render(<ChatScreen session={session} onLogout={vi.fn()} />)
    const composer = await screen.findByRole('textbox', { name: '发送给 Cove 的消息' })
    const focus = vi.spyOn(composer, 'focus')

    await user.pointer({ target: composer, keys: '[MouseLeft]' })

    expect(focus).toHaveBeenCalledWith({ preventScroll: true })
  })

  it('protects form-edge taps when the textarea remains focused after the keyboard closes', async () => {
    render(<ChatScreen session={session} onLogout={vi.fn()} />)
    const textarea = await screen.findByRole('textbox', { name: '发送给 Cove 的消息' })
    const form = textarea.closest('form')
    expect(form).toBeTruthy()

    textarea.focus()
    const focus = vi.spyOn(textarea, 'focus')
    fireEvent.pointerDown(form as HTMLFormElement)

    expect(focus).toHaveBeenCalledWith({ preventScroll: true })
  })

  it('loads the latest conversation and its message history', async () => {
    mocks.listConversations.mockResolvedValue({
      list: [
        {
          id: 'conversation-1',
          title: '周末安排',
          is_group: false,
          member_persona_ids: [],
          enable_tools: false,
          created_at: '2026-07-10T08:00:00Z',
          updated_at: '2026-07-11T08:00:00Z',
        },
      ],
    })
    mocks.listMessages.mockResolvedValue({
      list: [
        {
          id: 'message-1',
          role: 'assistant',
          content: '我们可以先安排上午。',
          meta_data: null,
          images: [],
          sender_persona_id: null,
          sender_name: null,
          feedback: null,
          created_at: '2026-07-11T08:01:00Z',
        },
      ],
    })

    render(<ChatScreen session={session} onLogout={vi.fn()} />)

    expect(await screen.findByText('我们可以先安排上午。')).toBeTruthy()
    expect(document.querySelector('.message-scroll')?.classList.contains('message-scroll--empty')).toBe(false)
    expect(mocks.listMessages).toHaveBeenCalledWith('conversation-1')
    expect(screen.getAllByText('周末安排')).toHaveLength(2)
  })

  it('creates a conversation from meta and renders streamed markdown tokens', async () => {
    const user = userEvent.setup()
    mocks.streamChat.mockImplementation(
      async (
        _input: unknown,
        _signal: AbortSignal,
        onEvent: (event: ChatStreamEvent) => void,
      ) => {
        onEvent({ type: 'meta', conversation_id: 'conversation-2', title: '学习计划' })
        onEvent({ type: 'token', text: '**先确定目标**' })
        onEvent({ type: 'done', text: 'message-2' })
      },
    )
    render(<ChatScreen session={session} onLogout={vi.fn()} />)
    await screen.findByRole('heading', { name: '你好，林海' })

    const composer = screen.getByRole('textbox', { name: '发送给 Cove 的消息' })
    await user.type(composer, '帮我制定学习计划')
    await user.click(screen.getByRole('button', { name: '发送消息' }))

    expect(await screen.findByText('先确定目标')).toHaveProperty('tagName', 'STRONG')
    expect(mocks.streamChat).toHaveBeenCalledWith(
      { message: '帮我制定学习计划', enable_knowledge: false },
      expect.any(AbortSignal),
      expect.any(Function),
    )
    expect(screen.getAllByText('学习计划').length).toBeGreaterThan(0)
    expect(mocks.listMessages).not.toHaveBeenCalled()
  })

  it('blocks duplicate sends while a stream is pending and supports retry after failure', async () => {
    const user = userEvent.setup()
    let emit: ((event: ChatStreamEvent) => void) | undefined
    let finish: (() => void) | undefined
    mocks.streamChat
      .mockImplementationOnce(
        (_input: unknown, _signal: AbortSignal, onEvent: (event: ChatStreamEvent) => void) => {
          emit = onEvent
          return new Promise<void>((resolve) => {
            finish = resolve
          })
        },
      )
      .mockImplementationOnce(
        async (_input: unknown, _signal: AbortSignal, onEvent: (event: ChatStreamEvent) => void) => {
          onEvent({ type: 'done', text: 'message-retry' })
        },
      )

    render(<ChatScreen session={session} onLogout={vi.fn()} />)
    await screen.findByRole('heading', { name: '你好，林海' })
    const composer = screen.getByRole('textbox', { name: '发送给 Cove 的消息' })
    await user.type(composer, '请再试一次')
    await user.click(screen.getByRole('button', { name: '发送消息' }))

    expect((composer as HTMLTextAreaElement).disabled).toBe(true)
    expect(mocks.streamChat).toHaveBeenCalledTimes(1)
    act(() => {
      emit?.({ type: 'error', content: '服务暂时不可用' })
      finish?.()
    })
    const retry = await screen.findByRole('button', { name: '重新发送' })
    await user.click(retry)

    await waitFor(() => expect(mocks.streamChat).toHaveBeenCalledTimes(2))
  })

  it('attaches text files, toggles knowledge, removes attachments, and retries the full submission', async () => {
    const user = userEvent.setup()
    mocks.getAgentConfig.mockResolvedValue({ enable_knowledge: true })
    mocks.streamChat
      .mockImplementationOnce(
        async (_input: unknown, _signal: AbortSignal, onEvent: (event: ChatStreamEvent) => void) => {
          onEvent({ type: 'error', content: '暂时失败' })
        },
      )
      .mockImplementationOnce(
        async (_input: unknown, _signal: AbortSignal, onEvent: (event: ChatStreamEvent) => void) => {
          onEvent({ type: 'done', text: 'message-retry' })
        },
      )

    const { container } = render(<ChatScreen session={session} onLogout={vi.fn()} />)
    await screen.findByRole('heading', { name: '你好，林海' })
    const knowledge = await screen.findByRole('button', { name: '使用知识库' })
    await waitFor(() => expect(knowledge.getAttribute('aria-pressed')).toBe('true'))
    await user.click(knowledge)

    const input = container.querySelector<HTMLInputElement>('input[type="file"]')
    const firstFile = new File(['first'], 'first.md', { type: 'text/markdown' })
    Object.defineProperty(firstFile, 'text', { value: vi.fn().mockResolvedValue('first') })
    fireEvent.change(input as HTMLInputElement, { target: { files: [firstFile] } })
    expect(await screen.findByText('first.md')).toBeTruthy()
    await user.click(screen.getByRole('button', { name: '移除附件 first.md' }))
    expect(screen.queryByText('first.md')).toBeNull()

    const notesFile = new File(['notes'], 'notes.md', { type: 'text/markdown' })
    Object.defineProperty(notesFile, 'text', { value: vi.fn().mockResolvedValue('# Notes') })
    fireEvent.change(input as HTMLInputElement, { target: { files: [notesFile] } })
    expect(await screen.findByText('notes.md')).toBeTruthy()

    await user.type(screen.getByRole('textbox', { name: '发送给 Cove 的消息' }), '总结附件')
    await user.click(screen.getByRole('button', { name: '发送消息' }))
    await user.click(await screen.findByRole('button', { name: '重新发送' }))

    const expectedInput = {
      message: '总结附件',
      attachments: [{ file_name: 'notes.md', text: '# Notes' }],
      enable_knowledge: false,
    }
    expect(mocks.streamChat).toHaveBeenNthCalledWith(
      1,
      expectedInput,
      expect.any(AbortSignal),
      expect.any(Function),
    )
    expect(mocks.streamChat).toHaveBeenNthCalledWith(
      2,
      expectedInput,
      expect.any(AbortSignal),
      expect.any(Function),
    )
  })

  it('enforces attachment type, size, and count limits', async () => {
    const { container } = render(<ChatScreen session={session} onLogout={vi.fn()} />)
    await screen.findByRole('heading', { name: '你好，林海' })
    const input = container.querySelector<HTMLInputElement>('input[type="file"]')
    const files = ['one.md', 'two.md', 'three.md', 'four.md'].map((name) => {
      const file = new File(['text'], name, { type: 'text/markdown' })
      Object.defineProperty(file, 'text', { value: vi.fn().mockResolvedValue(name) })
      return file
    })

    fireEvent.change(input as HTMLInputElement, { target: { files } })

    expect((await screen.findByRole('alert')).textContent).toContain('最多添加 3 个附件。')
    expect(screen.getByText('one.md')).toBeTruthy()
    expect(screen.getByText('two.md')).toBeTruthy()
    expect(screen.getByText('three.md')).toBeTruthy()
    expect(screen.queryByText('four.md')).toBeNull()
  })

  it('renames and deletes conversations through the row menu and closes it outside', async () => {
    const user = userEvent.setup()
    const conversation = {
      id: 'conversation-1',
      title: '旧名称',
      is_group: false,
      member_persona_ids: [],
      enable_tools: false,
      created_at: '2026-07-10T08:00:00Z',
      updated_at: '2026-07-11T08:00:00Z',
    }
    mocks.listConversations.mockResolvedValue({ list: [conversation] })
    mocks.renameConversation.mockResolvedValue({ ...conversation, title: '新名称' })
    mocks.deleteConversation.mockResolvedValue(undefined)

    render(<ChatScreen session={session} onLogout={vi.fn()} />)
    await screen.findAllByText('旧名称')
    const manage = screen.getByRole('button', { name: '管理会话：旧名称' })
    await user.click(manage)
    expect(screen.getByRole('menu')).toBeTruthy()
    await user.click(screen.getByRole('heading', { name: '你好，林海' }))
    expect(screen.queryByRole('menu')).toBeNull()

    await user.click(manage)
    await user.click(screen.getByRole('menuitem', { name: '重命名' }))
    const titleInput = screen.getByRole('textbox', { name: '会话名称' })
    await user.clear(titleInput)
    await user.type(titleInput, '新名称')
    await user.click(screen.getByRole('button', { name: '保存' }))
    await waitFor(() => expect(mocks.renameConversation).toHaveBeenCalledWith('conversation-1', '新名称'))

    await user.click(await screen.findByRole('button', { name: '管理会话：新名称' }))
    await user.click(screen.getByRole('menuitem', { name: '删除' }))
    await user.click(screen.getByRole('button', { name: '删除' }))

    await waitFor(() => expect(mocks.deleteConversation).toHaveBeenCalledWith('conversation-1'))
    expect(await screen.findByRole('heading', { name: '你好，林海' })).toBeTruthy()
    expect(screen.queryByText('新名称')).toBeNull()
  })
})
