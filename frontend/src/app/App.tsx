import { useEffect, useState } from 'react'
import './App.css'
import { AuthScreen } from '../features/auth/AuthScreen'
import { clearSession, restoreSession } from '../features/auth/api'
import type { StoredSession } from '../features/auth/types'
import { ChatScreen } from '../features/chat/ChatScreen'

const coveIcon = '/cove-mark.svg'

type AuthState =
  | { status: 'restoring' }
  | { status: 'anonymous' }
  | { status: 'authenticated'; session: StoredSession }

type AuthenticatedAppProps = {
  session: StoredSession
  onLogout: () => void
}

function AuthenticatedApp({ session, onLogout }: AuthenticatedAppProps) {
  return <ChatScreen session={session} onLogout={onLogout} />
}

function App() {
  const [authState, setAuthState] = useState<AuthState>({ status: 'restoring' })

  useEffect(() => {
    let active = true
    restoreSession().then((session) => {
      if (!active) {
        return
      }
      setAuthState(session ? { status: 'authenticated', session } : { status: 'anonymous' })
    })
    return () => {
      active = false
    }
  }, [])

  function handleLogout() {
    clearSession()
    setAuthState({ status: 'anonymous' })
  }

  if (authState.status === 'restoring') {
    return (
      <main className="launch-screen" aria-label="正在恢复登录状态">
        <img src={coveIcon} alt="" />
        <strong>Cove</strong>
        <span className="launch-screen__progress" />
      </main>
    )
  }

  if (authState.status === 'anonymous') {
    return (
      <AuthScreen
        onAuthenticated={(session) => setAuthState({ status: 'authenticated', session })}
      />
    )
  }

  return <AuthenticatedApp session={authState.session} onLogout={handleLogout} />
}

export default App
