import { useState } from 'preact/hooks'

async function tryLogin(username: string, password: string): Promise<boolean> {
  try {
    const res = await fetch('/api/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password }),
    })
    if (res.ok) {
      return true
    }
    return false
  } catch {
    return false
  }
}

export function LoginScreen() {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e: Event) => {
    e.preventDefault()
    setLoading(true)
    setError('')

    const ok = await tryLogin(username, password)
    if (ok) {
      window.location.reload()
    } else {
      setError('Invalid username or password')
      setLoading(false)
    }
  }

  return (
    <div class="flex flex-col items-center justify-center h-screen gap-4">
      <h1>Artemis</h1>
      <form onSubmit={handleSubmit} class="flex flex-col gap-3 w-[280px]">
        <input
          type="text"
          placeholder="Username"
          value={username}
          onChange={e => setUsername((e.target as HTMLInputElement).value)}
          required
          class="border border-artemis-border rounded p-2"
        />
        <input
          type="password"
          placeholder="Password"
          value={password}
          onChange={e => setPassword((e.target as HTMLInputElement).value)}
          required
          class="border border-artemis-border rounded p-2"
        />
        {error && <p class="text-red-500">{error}</p>}
        <button type="submit" disabled={loading} class="px-4 py-2 bg-artemis-primary text-white rounded">
          {loading ? 'Logging in...' : 'Login'}
        </button>
      </form>
    </div>
  )
}
