import {
  ArrowClockwise,
  ArrowUp,
  Books,
  DotsThree,
  GlobeHemisphereWest,
  List,
  Paperclip,
  PencilSimple,
  Plus,
  SignOut,
  Trash,
  WarningCircle,
  X,
} from '@phosphor-icons/react'
import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  type FormEvent,
  type KeyboardEvent,
} from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import type { StoredSession } from '../auth/types'
import {
  deleteConversation,
  getAgentConfig,
  listConversations,
  listMessages,
  renameConversation,
  streamChat,
} from './api'
import type {
  ChatAttachment,
  ChatMessage,
  ChatSubmission,
  ChatStreamEvent,
  Conversation,
  ResourceState,
  StreamState,
  ToolActivity,
} from './types'
import './ChatScreen.css'

const coveIcon = '/cove-mark.svg'
const maxAttachmentCount = 3
const maxAttachmentBytes = 1024 * 1024
const textFilePattern = /\.(?:txt|md|markdown|csv|json|log|ya?ml|xml|html?|css|jsx?|tsx?|py|go|rs|java|c|cpp|h|sh|sql)$/i

type ChatScreenProps = {
  session: StoredSession
  onLogout: () => void
}

function localMessage(role: 'user' | 'assistant', content: string): ChatMessage {
  return {
    id: `local-${role}-${Date.now()}-${Math.random().toString(16).slice(2)}`,
    role,
    content,
    meta_data: null,
    images: [],
    sender_persona_id: null,
    sender_name: null,
    feedback: null,
    created_at: new Date().toISOString(),
    pending: true,
    tools: [],
  }
}

function upsertConversation(items: Conversation[], event: { conversation_id: string; title: string }) {
  const existing = items.find((item) => item.id === event.conversation_id)
  const timestamp = new Date().toISOString()
  const next: Conversation = existing
    ? { ...existing, title: event.title, updated_at: timestamp }
    : {
        id: event.conversation_id,
        title: event.title,
        is_group: false,
        member_persona_ids: [],
        enable_tools: false,
        created_at: timestamp,
        updated_at: timestamp,
      }
  return [next, ...items.filter((item) => item.id !== next.id)]
}

function updateTool(tools: ToolActivity[], event: Extract<ChatStreamEvent, { type: string }>) {
  if (event.type !== 'tool_call' && event.type !== 'tool_result') {
    return tools
  }
  const toolEvent = event as {
    type: 'tool_call' | 'tool_result'
    tool_call_id: string
    tool: string
    error?: string
  }
  const status =
    toolEvent.type === 'tool_call' ? 'running' : toolEvent.error ? 'error' : 'complete'
  const activity: ToolActivity = {
    id: toolEvent.tool_call_id,
    tool: toolEvent.tool,
    status,
  }
  return [...tools.filter((item) => item.id !== activity.id), activity]
}

function isTextAttachment(file: File) {
  return file.type.startsWith('text/') || file.type === 'application/json' || textFilePattern.test(file.name)
}

export function ChatScreen({ session, onLogout }: ChatScreenProps) {
  const [conversations, setConversations] = useState<Conversation[]>([])
  const [conversationState, setConversationState] = useState<ResourceState>('loading')
  const [conversationError, setConversationError] = useState('')
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [messageState, setMessageState] = useState<ResourceState>('idle')
  const [messageError, setMessageError] = useState('')
  const [streamState, setStreamState] = useState<StreamState>({ status: 'idle' })
  const [draft, setDraft] = useState('')
  const [attachments, setAttachments] = useState<ChatAttachment[]>([])
  const [attachmentError, setAttachmentError] = useState('')
  const [knowledgeEnabled, setKnowledgeEnabled] = useState(false)
  const [knowledgeState, setKnowledgeState] = useState<ResourceState>('loading')
  const [knowledgeError, setKnowledgeError] = useState('')
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [accountOpen, setAccountOpen] = useState(false)
  const [conversationMenuId, setConversationMenuId] = useState<string | null>(null)
  const [renameTarget, setRenameTarget] = useState<Conversation | null>(null)
  const [renameTitle, setRenameTitle] = useState('')
  const [deleteTarget, setDeleteTarget] = useState<Conversation | null>(null)
  const [conversationActionError, setConversationActionError] = useState('')
  const [conversationActionPending, setConversationActionPending] = useState(false)
  const abortRef = useRef<AbortController | null>(null)
  const skipHistoryForRef = useRef<string | null>(null)
  const viewportRootRef = useRef<HTMLElement | null>(null)
  const accountMenuRef = useRef<HTMLDivElement | null>(null)
  const conversationMenuRef = useRef<HTMLDivElement | null>(null)
  const messageScrollRef = useRef<HTMLDivElement | null>(null)
  const textareaRef = useRef<HTMLTextAreaElement | null>(null)
  const fileInputRef = useRef<HTMLInputElement | null>(null)
  const hasMessagesRef = useRef(false)
  const keyboardHeightRef = useRef(Number(sessionStorage.getItem('cove-keyboard-height')) || 0)
  const keyboardPreparationTimerRef = useRef<number | null>(null)

  const displayName = session.user.nickname?.trim() || '用户'
  const activeConversation = conversations.find((item) => item.id === selectedId)
  const isEmptyConversation = messageState === 'ready' && messages.length === 0

  const loadConversations = useCallback(async (selectFirst = false) => {
    setConversationState('loading')
    setConversationError('')
    try {
      const response = await listConversations()
      const sorted = [...response.list].sort(
        (a, b) => Date.parse(b.updated_at) - Date.parse(a.updated_at),
      )
      setConversations(sorted)
      setConversationState('ready')
      if (selectFirst && sorted.length > 0) {
        setSelectedId((current) => current ?? sorted[0].id)
      }
    } catch (error: unknown) {
      setConversationState('error')
      setConversationError(error instanceof Error ? error.message : '会话加载失败。')
    }
  }, [])

  const loadHistory = useCallback(async (conversationId: string) => {
    setMessageState('loading')
    setMessageError('')
    try {
      const response = await listMessages(conversationId)
      setMessages(response.list)
      setMessageState('ready')
    } catch (error: unknown) {
      setMessageState('error')
      setMessageError(error instanceof Error ? error.message : '消息加载失败。')
    }
  }, [])

  const loadKnowledgeConfig = useCallback(async () => {
    setKnowledgeState('loading')
    setKnowledgeError('')
    try {
      const config = await getAgentConfig()
      setKnowledgeEnabled(Boolean(config.enable_knowledge))
      setKnowledgeState('ready')
    } catch (error: unknown) {
      setKnowledgeState('error')
      setKnowledgeError(error instanceof Error ? error.message : '知识库配置加载失败。')
    }
  }, [])

  useEffect(() => {
    void loadConversations(true)
    void loadKnowledgeConfig()
  }, [loadConversations, loadKnowledgeConfig])

  useEffect(() => {
    if (!selectedId) {
      setMessages([])
      setMessageState('ready')
      return
    }
    if (skipHistoryForRef.current === selectedId) {
      skipHistoryForRef.current = null
      setMessageState('ready')
      return
    }
    void loadHistory(selectedId)
  }, [loadHistory, selectedId])

  useEffect(() => {
    hasMessagesRef.current = messages.length > 0
    const messageScroll = messageScrollRef.current
    messageScroll?.scrollTo({ top: messageScroll.scrollHeight, behavior: 'smooth' })
  }, [messages])

  useEffect(() => {
    return () => {
      abortRef.current?.abort()
      if (keyboardPreparationTimerRef.current !== null) {
        window.clearTimeout(keyboardPreparationTimerRef.current)
      }
    }
  }, [])

  useEffect(() => {
    if (!accountOpen) {
      return
    }

    function closeAccountMenu(event: PointerEvent) {
      if (!accountMenuRef.current?.contains(event.target as Node)) {
        setAccountOpen(false)
      }
    }

    function closeAccountMenuWithEscape(event: globalThis.KeyboardEvent) {
      if (event.key === 'Escape') {
        setAccountOpen(false)
      }
    }

    document.addEventListener('pointerdown', closeAccountMenu)
    document.addEventListener('keydown', closeAccountMenuWithEscape)
    return () => {
      document.removeEventListener('pointerdown', closeAccountMenu)
      document.removeEventListener('keydown', closeAccountMenuWithEscape)
    }
  }, [accountOpen])

  useEffect(() => {
    if (!conversationMenuId) {
      return
    }

    function closeConversationMenu(event: PointerEvent) {
      if (!conversationMenuRef.current?.contains(event.target as Node)) {
        setConversationMenuId(null)
      }
    }

    function closeConversationMenuWithEscape(event: globalThis.KeyboardEvent) {
      if (event.key === 'Escape') {
        setConversationMenuId(null)
      }
    }

    document.addEventListener('pointerdown', closeConversationMenu)
    document.addEventListener('keydown', closeConversationMenuWithEscape)
    return () => {
      document.removeEventListener('pointerdown', closeConversationMenu)
      document.removeEventListener('keydown', closeConversationMenuWithEscape)
    }
  }, [conversationMenuId])

  useEffect(() => {
    if (!renameTarget && !deleteTarget) {
      return
    }
    function closeDialogWithEscape(event: globalThis.KeyboardEvent) {
      if (event.key === 'Escape' && !conversationActionPending) {
        setRenameTarget(null)
        setDeleteTarget(null)
        setConversationActionError('')
      }
    }
    document.addEventListener('keydown', closeDialogWithEscape)
    return () => document.removeEventListener('keydown', closeDialogWithEscape)
  }, [conversationActionPending, deleteTarget, renameTarget])

  useLayoutEffect(() => {
    document.documentElement.classList.add('chat-document')
    window.scrollTo(0, 0)
    return () => document.documentElement.classList.remove('chat-document')
  }, [])

  useLayoutEffect(() => {
    const root = viewportRootRef.current
    const viewport = window.visualViewport
    if (!root || !viewport) {
      return
    }
    const activeRoot = root
    const activeViewport = viewport
    let layoutHeight = Math.max(window.innerHeight, activeViewport.height)
    let layoutWidth = activeViewport.width

    function syncVisualViewport() {
      const widthChanged = Math.abs(activeViewport.width - layoutWidth) > 1
      if (widthChanged) {
        layoutHeight = activeViewport.height
        layoutWidth = activeViewport.width
      }

      layoutHeight = Math.max(layoutHeight, window.innerHeight, activeViewport.height)
      const keyboardHeight = Math.max(0, layoutHeight - activeViewport.height)
      const contentShift = Math.round(Math.min(160, keyboardHeight * 0.45))
      const keyboardOpen = keyboardHeight > 20
      if (!keyboardOpen && activeViewport.height > layoutHeight) {
        layoutHeight = activeViewport.height
      }

      activeRoot.style.setProperty('--chat-keyboard-height', `${keyboardHeight}px`)
      activeRoot.style.setProperty('--chat-content-shift', `${contentShift}px`)
      activeRoot.dataset.keyboardOpen = String(keyboardOpen)
      if (keyboardOpen) {
        keyboardHeightRef.current = keyboardHeight
        sessionStorage.setItem('cove-keyboard-height', String(keyboardHeight))
        if (keyboardPreparationTimerRef.current !== null) {
          window.clearTimeout(keyboardPreparationTimerRef.current)
          keyboardPreparationTimerRef.current = null
        }
        window.requestAnimationFrame(() => {
          const messageScroll = messageScrollRef.current
          messageScroll?.scrollTo({ top: messageScroll.scrollHeight })
        })
      } else if (!hasMessagesRef.current) {
        window.requestAnimationFrame(() => {
          messageScrollRef.current?.scrollTo({ top: 0 })
        })
      }
    }

    syncVisualViewport()
    activeViewport.addEventListener('resize', syncVisualViewport)
    return () => {
      activeViewport.removeEventListener('resize', syncVisualViewport)
    }
  }, [])

  function focusComposerWithoutScroll(textarea: HTMLTextAreaElement) {
    const root = viewportRootRef.current
    const viewport = window.visualViewport
    if (root && viewport && viewport.width < 900) {
      const anticipatedHeight =
        keyboardHeightRef.current || Math.min(360, Math.max(260, window.innerHeight * 0.38))
      const anticipatedContentShift = Math.round(Math.min(160, anticipatedHeight * 0.45))
      const heightBeforeFocus = viewport.height
      root.style.setProperty('--chat-keyboard-height', `${anticipatedHeight}px`)
      root.dataset.keyboardOpen = 'true'
      void root.offsetHeight

      if (keyboardPreparationTimerRef.current !== null) {
        window.clearTimeout(keyboardPreparationTimerRef.current)
      }
      keyboardPreparationTimerRef.current = window.setTimeout(() => {
        if (viewport.height >= heightBeforeFocus - 20) {
          root.style.setProperty('--chat-keyboard-height', '0px')
          root.style.setProperty('--chat-content-shift', '0px')
          root.dataset.keyboardOpen = 'false'
        }
        keyboardPreparationTimerRef.current = null
      }, 650)
      window.requestAnimationFrame(() => {
        root.style.setProperty('--chat-content-shift', `${anticipatedContentShift}px`)
      })
    }
    textarea.focus({ preventScroll: true })
  }

  function handleComposerSurfacePress(event: {
    target: EventTarget | null
    preventDefault: () => void
  }) {
    const root = viewportRootRef.current
    const textarea = textareaRef.current
    const target = event.target
    if (
      !root ||
      !textarea ||
      !(target instanceof Element) ||
      target.closest('button') ||
      root.dataset.keyboardOpen === 'true'
    ) {
      return
    }

    event.preventDefault()
    focusComposerWithoutScroll(textarea)
  }

  function startNewConversation() {
    abortRef.current?.abort()
    abortRef.current = null
    setSelectedId(null)
    setMessages([])
    setMessageState('ready')
    setMessageError('')
    setStreamState({ status: 'idle' })
    setAttachments([])
    setAttachmentError('')
    setConversationMenuId(null)
    setDrawerOpen(false)
    window.requestAnimationFrame(() => {
      const textarea = textareaRef.current
      if (textarea) {
        focusComposerWithoutScroll(textarea)
      }
    })
  }

  function selectConversation(conversationId: string) {
    if (conversationId === selectedId) {
      setDrawerOpen(false)
      return
    }
    abortRef.current?.abort()
    abortRef.current = null
    setStreamState({ status: 'idle' })
    setAttachments([])
    setAttachmentError('')
    setConversationMenuId(null)
    setSelectedId(conversationId)
    setDrawerOpen(false)
  }

  function beginRename(conversation: Conversation) {
    setConversationMenuId(null)
    setConversationActionError('')
    setRenameTitle(conversation.title || '新对话')
    setRenameTarget(conversation)
  }

  async function submitRename(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const title = renameTitle.trim()
    if (!renameTarget || !title || title.length > 256 || conversationActionPending) {
      if (!title) {
        setConversationActionError('请输入会话名称。')
      } else if (title.length > 256) {
        setConversationActionError('会话名称不能超过 256 个字符。')
      }
      return
    }

    setConversationActionPending(true)
    setConversationActionError('')
    try {
      const updated = await renameConversation(renameTarget.id, title)
      setConversations((current) =>
        current.map((conversation) => (conversation.id === updated.id ? updated : conversation)),
      )
      setRenameTarget(null)
    } catch (error: unknown) {
      setConversationActionError(error instanceof Error ? error.message : '重命名失败。')
    } finally {
      setConversationActionPending(false)
    }
  }

  async function confirmDeleteConversation() {
    if (!deleteTarget || conversationActionPending) {
      return
    }
    setConversationActionPending(true)
    setConversationActionError('')
    try {
      await deleteConversation(deleteTarget.id)
      setConversations((current) => current.filter((conversation) => conversation.id !== deleteTarget.id))
      if (selectedId === deleteTarget.id) {
        abortRef.current?.abort()
        abortRef.current = null
        setSelectedId(null)
        setMessages([])
        setMessageState('ready')
        setStreamState({ status: 'idle' })
        setAttachments([])
        setAttachmentError('')
        setDrawerOpen(false)
      }
      setDeleteTarget(null)
    } catch (error: unknown) {
      setConversationActionError(error instanceof Error ? error.message : '删除会话失败。')
    } finally {
      setConversationActionPending(false)
    }
  }

  async function handleAttachmentSelection(files: FileList | null) {
    if (!files?.length) {
      return
    }
    const next = [...attachments]
    let errorMessage = ''
    for (const file of Array.from(files)) {
      if (next.length >= maxAttachmentCount) {
        errorMessage = `最多添加 ${maxAttachmentCount} 个附件。`
        break
      }
      if (!isTextAttachment(file)) {
        errorMessage = `${file.name} 不是支持的文本文件。`
        continue
      }
      if (file.size > maxAttachmentBytes) {
        errorMessage = `${file.name} 超过 1 MiB。`
        continue
      }
      if (next.some((attachment) => attachment.file_name === file.name)) {
        errorMessage = `${file.name} 已经添加。`
        continue
      }
      const text = await file.text()
      if (!text) {
        errorMessage = `${file.name} 是空文件。`
        continue
      }
      next.push({ file_name: file.name, text })
    }
    setAttachments(next)
    setAttachmentError(errorMessage)
    if (fileInputRef.current) {
      fileInputRef.current.value = ''
    }
  }

  function updateAssistant(id: string, updater: (message: ChatMessage) => ChatMessage) {
    setMessages((current) => current.map((message) => (message.id === id ? updater(message) : message)))
  }

  async function sendMessage(submission?: ChatSubmission) {
    const request: ChatSubmission = submission ?? {
      message: draft.trim(),
      attachments: attachments.map((attachment) => ({ ...attachment })),
      enableKnowledge: knowledgeEnabled,
    }
    const text = request.message.trim()
    if (!text || streamState.status === 'streaming') {
      return
    }

    const conversationIdAtSend = selectedId
    const userMessage = localMessage('user', text)
    const assistantMessage = localMessage('assistant', '')
    const controller = new AbortController()
    abortRef.current = controller
    setDraft('')
    setAttachments([])
    setAttachmentError('')
    setMessageError('')
    setMessageState('ready')
    setStreamState({ status: 'streaming' })
    setMessages((current) => [...current, userMessage, assistantMessage])
    let reachedTerminalEvent = false

    try {
      await streamChat(
        {
          ...(conversationIdAtSend ? { conversation_id: conversationIdAtSend } : {}),
          message: text,
          ...(request.attachments.length > 0 ? { attachments: request.attachments } : {}),
          enable_knowledge: request.enableKnowledge,
        },
        controller.signal,
        (event) => {
          if (event.type === 'meta') {
            const meta = event as { type: 'meta'; conversation_id: string; title: string }
            if (meta.conversation_id !== conversationIdAtSend) {
              skipHistoryForRef.current = meta.conversation_id
              setSelectedId(meta.conversation_id)
            }
            setConversations((current) => upsertConversation(current, meta))
            return
          }
          if (event.type === 'token') {
            const token = event as { type: 'token'; text: string }
            updateAssistant(assistantMessage.id, (message) => ({
              ...message,
              content: message.content + token.text,
            }))
            return
          }
          if (event.type === 'tool_call' || event.type === 'tool_result') {
            updateAssistant(assistantMessage.id, (message) => ({
              ...message,
              tools: updateTool(message.tools ?? [], event),
            }))
            return
          }
          if (event.type === 'done') {
            reachedTerminalEvent = true
            const done = event as { type: 'done'; text: string }
            updateAssistant(assistantMessage.id, (message) => ({
              ...message,
              id: done.text || message.id,
              pending: false,
            }))
            setMessages((current) =>
              current.map((message) =>
                message.id === userMessage.id ? { ...message, pending: false } : message,
              ),
            )
            setStreamState({ status: 'idle' })
            return
          }
          if (event.type === 'error') {
            reachedTerminalEvent = true
            const streamError = event as { type: 'error'; content: string }
            updateAssistant(assistantMessage.id, (message) => ({ ...message, pending: false }))
            setStreamState({ status: 'error', message: streamError.content, submission: request })
            return
          }
          if (import.meta.env.DEV) {
            console.debug('Ignored chat stream event', event)
          }
        },
      )
      if (!reachedTerminalEvent && !controller.signal.aborted) {
        updateAssistant(assistantMessage.id, (item) => ({ ...item, pending: false }))
        setStreamState({
          status: 'error',
          message: '消息流提前结束，请重新发送。',
          submission: request,
        })
      }
    } catch (error: unknown) {
      if (!controller.signal.aborted) {
        const message = error instanceof Error ? error.message : '回复中断，请稍后重试。'
        updateAssistant(assistantMessage.id, (item) => ({ ...item, pending: false }))
        setStreamState({ status: 'error', message, submission: request })
      }
    } finally {
      if (abortRef.current === controller) {
        abortRef.current = null
      }
    }
  }

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    void sendMessage()
  }

  function handleComposerKeyDown(event: KeyboardEvent<HTMLTextAreaElement>) {
    if (event.key === 'Enter' && !event.shiftKey && !event.nativeEvent.isComposing) {
      event.preventDefault()
      void sendMessage()
    }
  }

  function handleDraftChange(value: string) {
    setDraft(value)
    const textarea = textareaRef.current
    if (textarea) {
      textarea.style.height = 'auto'
      textarea.style.height = `${Math.min(textarea.scrollHeight, 132)}px`
    }
  }

  return (
    <main className="chat-app" ref={viewportRootRef}>
      <button
        className={drawerOpen ? 'chat-drawer-scrim chat-drawer-scrim--visible' : 'chat-drawer-scrim'}
        type="button"
        aria-label="关闭会话列表"
        tabIndex={drawerOpen ? 0 : -1}
        onClick={() => setDrawerOpen(false)}
      />

      <aside className={drawerOpen ? 'chat-drawer chat-drawer--open' : 'chat-drawer'}>
        <header className="chat-drawer__header">
          <div className="brand-lockup">
            <img className="brand-lockup__icon" src={coveIcon} alt="" />
            <span>Cove</span>
          </div>
          <button className="icon-button chat-drawer__close" type="button" aria-label="关闭会话列表" onClick={() => setDrawerOpen(false)}>
            <X size={20} weight="bold" />
          </button>
        </header>

        <button className="new-chat-button" type="button" onClick={startNewConversation}>
          <Plus size={18} weight="bold" />
          <span>新对话</span>
        </button>

        <div className="conversation-list" aria-label="历史会话">
          <p className="conversation-list__label">最近对话</p>
          {conversationState === 'loading' && (
            <div className="conversation-skeleton" aria-label="正在加载会话">
              <span />
              <span />
              <span />
            </div>
          )}
          {conversationState === 'error' && (
            <div className="drawer-error" role="alert">
              <p>{conversationError}</p>
              <button type="button" onClick={() => void loadConversations(false)}>
                <ArrowClockwise size={16} /> 重试
              </button>
            </div>
          )}
          {conversationState === 'ready' && conversations.length === 0 && (
            <p className="conversation-list__empty">发送第一条消息后，会话会保存在这里。</p>
          )}
          {conversations.map((conversation) => (
            <div
              className={conversation.id === selectedId ? 'conversation-row conversation-row--active' : 'conversation-row'}
              key={conversation.id}
            >
              <button className="conversation-row__select" type="button" onClick={() => selectConversation(conversation.id)}>
                <span>{conversation.title || '新对话'}</span>
                <time dateTime={conversation.updated_at}>
                  {new Intl.DateTimeFormat('zh-CN', { month: 'numeric', day: 'numeric' }).format(
                    new Date(conversation.updated_at),
                  )}
                </time>
              </button>
              <div
                className="conversation-row__menu-wrap"
                ref={conversationMenuId === conversation.id ? conversationMenuRef : undefined}
              >
                <button
                  className="conversation-row__menu-trigger"
                  type="button"
                  aria-label={`管理会话：${conversation.title || '新对话'}`}
                  aria-expanded={conversationMenuId === conversation.id}
                  onClick={() => setConversationMenuId((current) => current === conversation.id ? null : conversation.id)}
                >
                  <DotsThree size={18} weight="bold" />
                </button>
                {conversationMenuId === conversation.id && (
                  <div className="conversation-row__menu" role="menu">
                    <button
                      type="button"
                      role="menuitem"
                      disabled={streamState.status === 'streaming' && selectedId === conversation.id}
                      onClick={() => beginRename(conversation)}
                    >
                      <PencilSimple size={16} /> 重命名
                    </button>
                    <button
                      className="conversation-row__delete"
                      type="button"
                      role="menuitem"
                      disabled={streamState.status === 'streaming' && selectedId === conversation.id}
                      onClick={() => { setConversationMenuId(null); setConversationActionError(''); setDeleteTarget(conversation) }}
                    >
                      <Trash size={16} /> 删除
                    </button>
                  </div>
                )}
              </div>
            </div>
          ))}
        </div>

        <div className="drawer-account">
          <span className="drawer-account__avatar">{displayName.slice(0, 1).toUpperCase()}</span>
          <span className="drawer-account__name">{displayName}</span>
        </div>
      </aside>

      {renameTarget && (
        <div
          className="conversation-dialog-backdrop"
          onPointerDown={(event) => {
            if (event.target === event.currentTarget && !conversationActionPending) {
              setRenameTarget(null)
              setConversationActionError('')
            }
          }}
        >
          <form className="conversation-dialog" role="dialog" aria-modal="true" aria-labelledby="rename-conversation-title" onSubmit={submitRename}>
            <h2 id="rename-conversation-title">重命名会话</h2>
            <input
              autoFocus
              value={renameTitle}
              maxLength={256}
              aria-label="会话名称"
              disabled={conversationActionPending}
              onChange={(event) => setRenameTitle(event.target.value)}
            />
            {conversationActionError && <p role="alert">{conversationActionError}</p>}
            <div className="conversation-dialog__actions">
              <button type="button" disabled={conversationActionPending} onClick={() => { setRenameTarget(null); setConversationActionError('') }}>取消</button>
              <button className="conversation-dialog__primary" type="submit" disabled={conversationActionPending || !renameTitle.trim()}>保存</button>
            </div>
          </form>
        </div>
      )}

      {deleteTarget && (
        <div
          className="conversation-dialog-backdrop"
          onPointerDown={(event) => {
            if (event.target === event.currentTarget && !conversationActionPending) {
              setDeleteTarget(null)
              setConversationActionError('')
            }
          }}
        >
          <section className="conversation-dialog" role="dialog" aria-modal="true" aria-labelledby="delete-conversation-title">
            <h2 id="delete-conversation-title">删除会话？</h2>
            <p>“{deleteTarget.title || '新对话'}”及其消息将被永久删除。</p>
            {conversationActionError && <p className="conversation-dialog__error" role="alert">{conversationActionError}</p>}
            <div className="conversation-dialog__actions">
              <button type="button" disabled={conversationActionPending} onClick={() => { setDeleteTarget(null); setConversationActionError('') }}>取消</button>
              <button className="conversation-dialog__danger" type="button" disabled={conversationActionPending} onClick={() => void confirmDeleteConversation()}>删除</button>
            </div>
          </section>
        </div>
      )}

      <section className="chat-workspace">
        <header className="chat-header">
          <button className="icon-button chat-header__menu" type="button" aria-label="打开会话列表" onClick={() => { setAccountOpen(false); setDrawerOpen(true) }}>
            <List size={22} />
          </button>
          <div className="chat-header__title">
            <strong>{activeConversation?.title || '新对话'}</strong>
            <span>{streamState.status === 'streaming' ? 'Cove 正在回复' : 'Cove AI'}</span>
          </div>
          <div className="account-menu" ref={accountMenuRef}>
            <button className="icon-button" type="button" aria-label="打开账户菜单" aria-expanded={accountOpen} onClick={() => setAccountOpen((open) => !open)}>
              <DotsThree size={24} weight="bold" />
            </button>
            {accountOpen && (
              <div className="account-menu__popover" role="menu">
                <div>
                  <strong>{displayName}</strong>
                  <span>{session.user.email || `@${session.user.username}`}</span>
                </div>
                <button type="button" onClick={onLogout}>
                  <SignOut size={18} />
                  退出登录
                </button>
              </div>
            )}
          </div>
        </header>

        <div
          className={isEmptyConversation ? 'message-scroll message-scroll--empty' : 'message-scroll'}
          ref={messageScrollRef}
          aria-busy={messageState === 'loading'}
        >
          <div className="message-column" role="log" aria-live="polite" aria-relevant="additions text">
            {messageState === 'loading' && (
              <div className="message-skeleton" aria-label="正在加载消息">
                <span />
                <span />
                <span />
              </div>
            )}
            {messageState === 'error' && (
              <div className="message-error" role="alert">
                <WarningCircle size={24} />
                <p>{messageError}</p>
                {selectedId && (
                  <button type="button" onClick={() => void loadHistory(selectedId)}>
                    重新加载
                  </button>
                )}
              </div>
            )}
            {messageState === 'ready' && messages.length === 0 && (
              <div className="chat-empty">
                <img src={coveIcon} alt="" />
                <h1>你好，{displayName}</h1>
                <p>把正在思考的事告诉我，我们一起理清。</p>
                <div className="prompt-suggestions">
                  <button type="button" onClick={() => handleDraftChange('帮我把今天最重要的三件事理清楚')}>
                    梳理今天的重点
                  </button>
                  <button type="button" onClick={() => handleDraftChange('帮我制定一个可执行的学习计划')}>
                    制定学习计划
                  </button>
                </div>
              </div>
            )}
            {messages.map((message) => (
              <article className={`message message--${message.role}`} key={message.id}>
                {message.role === 'assistant' && (
                  <img className="message__avatar" src={coveIcon} alt="Cove" />
                )}
                <div className="message__body">
                  {message.tools && message.tools.length > 0 && (
                    <div className="tool-activity">
                      {message.tools.map((tool) => (
                        <span className={`tool-activity__item tool-activity__item--${tool.status}`} key={tool.id}>
                          {tool.status === 'running' ? '正在使用' : tool.status === 'error' ? '工具失败' : '已使用'} {tool.tool}
                        </span>
                      ))}
                    </div>
                  )}
                  {message.role === 'assistant' ? (
                    message.content ? (
                      <ReactMarkdown
                        remarkPlugins={[remarkGfm]}
                        skipHtml
                        components={{
                          a: ({ href, children }) => (
                            <a href={href} target="_blank" rel="noreferrer noopener">
                              {children}
                            </a>
                          ),
                        }}
                      >
                        {message.content}
                      </ReactMarkdown>
                    ) : (
                      message.pending && <span className="thinking-indicator" aria-label="Cove 正在思考"><i /><i /><i /></span>
                    )
                  ) : (
                    <p>{message.content}</p>
                  )}
                  {message.role === 'assistant' && message.pending && message.content && (
                    <span className="stream-cursor" aria-hidden="true" />
                  )}
                </div>
              </article>
            ))}
            {streamState.status === 'error' && (
              <div className="stream-error" role="alert">
                <WarningCircle size={19} />
                <span>{streamState.message}</span>
                <button type="button" onClick={() => void sendMessage(streamState.submission)}>
                  重新发送
                </button>
              </div>
            )}
            <div />
          </div>
        </div>

        <footer className="composer-area">
          <form
            className="composer"
            onSubmit={handleSubmit}
            onTouchStartCapture={handleComposerSurfacePress}
            onPointerDownCapture={handleComposerSurfacePress}
          >
            <textarea
              ref={textareaRef}
              rows={1}
              value={draft}
              placeholder="问问 Cove..."
              aria-label="发送给 Cove 的消息"
              autoComplete="off"
              autoCorrect="off"
              autoCapitalize="sentences"
              spellCheck={false}
              enterKeyHint="send"
              disabled={streamState.status === 'streaming'}
              onTouchStart={(event) => {
                if (document.activeElement !== event.currentTarget) {
                  event.preventDefault()
                  focusComposerWithoutScroll(event.currentTarget)
                }
              }}
              onPointerDown={(event) => {
                if (document.activeElement !== event.currentTarget) {
                  event.preventDefault()
                  focusComposerWithoutScroll(event.currentTarget)
                }
              }}
              onFocus={() => setAccountOpen(false)}
              onChange={(event) => handleDraftChange(event.target.value)}
              onKeyDown={handleComposerKeyDown}
            />
            <input
              className="composer__file-input"
              ref={fileInputRef}
              type="file"
              multiple
              accept="text/*,.md,.markdown,.csv,.json,.log,.yaml,.yml,.xml,.html,.css,.js,.jsx,.ts,.tsx,.py,.go,.rs,.java,.c,.cpp,.h,.sh,.sql"
              tabIndex={-1}
              aria-hidden="true"
              onChange={(event) => void handleAttachmentSelection(event.target.files)}
            />
            {attachments.length > 0 && (
              <div className="composer-attachments" aria-label="已添加附件">
                {attachments.map((attachment) => (
                  <span className="composer-attachment" key={attachment.file_name}>
                    <Paperclip size={13} />
                    <span>{attachment.file_name}</span>
                    <button
                      type="button"
                      aria-label={`移除附件 ${attachment.file_name}`}
                      disabled={streamState.status === 'streaming'}
                      onClick={() => setAttachments((current) => current.filter((item) => item.file_name !== attachment.file_name))}
                    >
                      <X size={13} weight="bold" />
                    </button>
                  </span>
                ))}
              </div>
            )}
            {(attachmentError || knowledgeError) && (
              <p className="composer__error" role="alert">{attachmentError || knowledgeError}</p>
            )}
            <div className="composer__toolbar">
              <div>
                <button
                  className="composer-tool"
                  type="button"
                  aria-label="添加文本附件"
                  title="添加文本附件"
                  disabled={streamState.status === 'streaming' || attachments.length >= maxAttachmentCount}
                  onClick={() => fileInputRef.current?.click()}
                >
                  <Paperclip size={18} />
                </button>
                <button
                  className={knowledgeEnabled ? 'composer-tool composer-tool--active' : 'composer-tool'}
                  type="button"
                  aria-label={knowledgeState === 'error' ? '重新加载知识库配置' : '使用知识库'}
                  aria-pressed={knowledgeEnabled}
                  title={knowledgeState === 'error' ? '配置加载失败，点击重试' : knowledgeEnabled ? '知识库已开启' : '知识库已关闭'}
                  disabled={streamState.status === 'streaming' || knowledgeState === 'loading'}
                  onClick={() => {
                    if (knowledgeState === 'error') {
                      void loadKnowledgeConfig()
                    } else {
                      setKnowledgeEnabled((enabled) => !enabled)
                    }
                  }}
                >
                  <Books size={18} />
                </button>
                <button className="composer-tool" type="button" disabled aria-label="联网搜索，服务端暂未接入" title="联网搜索服务端暂未接入">
                  <GlobeHemisphereWest size={18} />
                </button>
              </div>
              <button className="send-button" type="submit" aria-label="发送消息" disabled={!draft.trim() || streamState.status === 'streaming'}>
                <ArrowUp size={19} weight="bold" />
              </button>
            </div>
          </form>
          <p>Cove 可能会出错，请核对重要信息。</p>
        </footer>
      </section>
    </main>
  )
}
