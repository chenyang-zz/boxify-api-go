export type Conversation = {
  id: string
  title: string
  is_group: boolean
  member_persona_ids: string[]
  enable_tools: boolean
  created_at: string
  updated_at: string
}

export type MessageMetadata = {
  image_keys?: string[]
  sender_name?: string
  interrupted?: boolean
}

export type ChatMessage = {
  id: string
  role: 'user' | 'assistant' | 'system' | string
  content: string
  meta_data: MessageMetadata | null
  images: string[]
  sender_persona_id: string | null
  sender_name: string | null
  feedback: string | null
  created_at: string
  pending?: boolean
  tools?: ToolActivity[]
}

export type ChatStreamRequest = {
  conversation_id?: string
  message: string
  greeting?: string
  skill_id?: string
  image_keys?: string[]
  attachments?: Array<{ file_name: string; text?: string }>
  enable_knowledge?: boolean
  enable_memory?: boolean
  enable_web_search?: boolean
}

export type ChatMetaEvent = {
  type: 'meta'
  conversation_id: string
  title: string
}

export type ChatTextEvent = {
  type: 'token' | 'done'
  text: string
}

export type ChatErrorEvent = {
  type: 'error'
  content: string
}

export type ChatToolEvent = {
  type: 'tool_call' | 'tool_result'
  tool: string
  input?: Record<string, unknown>
  observation?: string
  error?: string
  iteration: number
  tool_call_id: string
}

export type ChatUnknownEvent = {
  type: string
  [key: string]: unknown
}

export type ChatStreamEvent =
  | ChatMetaEvent
  | ChatTextEvent
  | ChatErrorEvent
  | ChatToolEvent
  | ChatUnknownEvent

export type ToolActivity = {
  id: string
  tool: string
  status: 'running' | 'complete' | 'error'
}

export type ResourceState = 'idle' | 'loading' | 'ready' | 'error'

export type StreamState =
  | { status: 'idle' }
  | { status: 'streaming' }
  | { status: 'error'; message: string; prompt: string }
