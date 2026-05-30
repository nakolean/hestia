import { useState, useEffect } from 'preact/hooks'
import { get, post, del } from '../api/client'

interface APIKey {
  id: number
  name: string
  permissions: string
  ip_whitelist: string
  expires_at: string | null
  last_used_at: string | null
  created_at: string
}

interface RawKeyResponse {
  raw_key: string
  key: APIKey
}

export function ApiKeyManager() {
  const [keys, setKeys] = useState<APIKey[]>([])
  const [newName, setNewName] = useState('')
  const [newPerm, setNewPerm] = useState<'read' | 'write' | 'full'>('read')
  const [rawKeyDisplay, setRawKeyDisplay] = useState<string | null>(null)
  const [error, setError] = useState('')

  const loadKeys = async () => {
    try {
      const data = await get('/admin/keys')
      setKeys(data.keys as APIKey[])
    } catch {
      // Silently fail — keys may not be available in public mode
    }
  }

  useEffect(() => { loadKeys() }, [])

  const handleCreate = async () => {
    if (!newName.trim()) return
    try {
      const resp: RawKeyResponse = await post('/admin/keys', { name: newName, permissions: newPerm })
      setRawKeyDisplay(resp.raw_key)
      setNewName('')
      await loadKeys()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create key')
    }
  }

  const handleDelete = async (id: number) => {
    try {
      await del(`/admin/keys/${id}`)
      await loadKeys()
    } catch {
      // Silently fail
    }
  }

  return (
    <div>
      <h3 class="text-lg font-semibold">API Keys</h3>
      <form onSubmit={e => { e.preventDefault(); handleCreate() }}>
        <div class="grid grid-cols-2 gap-2 mb-2">
          <label class="block">
            Name
            <input class="w-full border border-hestia-border rounded p-2" value={newName} onInput={e => setNewName((e.target as HTMLInputElement).value)} placeholder="My key" required />
          </label>
          <label class="block">
            Permissions
            <select class="w-full border border-hestia-border rounded p-2" value={newPerm} onChange={e => setNewPerm((e.target as HTMLSelectElement).value as 'read' | 'write' | 'full')}>
              <option value="read">read</option>
              <option value="write">write</option>
              <option value="full">full</option>
            </select>
          </label>
        </div>
        <button type="submit" class="px-4 py-2 bg-hestia-primary text-white rounded">Generate Key</button>
      </form>

      {rawKeyDisplay && (
        <div class="bg-hestia-primary/10 p-4 mt-4 rounded" style="background:rgba(1,114,173,0.1)">
          <strong>Save now — this key cannot be viewed again!</strong>
          <code class="block mt-2 break-all font-mono">{rawKeyDisplay}</code>
          <button class="mt-2 px-3 py-1 text-sm border border-current rounded bg-transparent hover:bg-hestia-primary hover:text-white" onClick={() => setRawKeyDisplay(null)}>Dismiss</button>
        </div>
      )}

      {error && <small class="text-red-500">{error}</small>}

      <h4 class="text-base font-semibold">Existing Keys</h4>
      {keys.length === 0 ? <p class="text-hestia-text-muted">No keys created yet.</p> : (
        keys.map(key => (
          <div key={key.id} class="mb-2 border border-hestia-border rounded p-3">
            <div class="flex justify-between items-center">
              <div>
                <strong>{key.name}</strong> <em class="text-hestia-text-muted" style="font-style:normal">({key.permissions})</em>
                <div class="text-xs text-hestia-text-muted">
                  {key.last_used_at ? `last used ${new Date(key.last_used_at).toLocaleString()}` : 'never used'}
                </div>
              </div>
              <button class="text-xs px-2 py-1 border border-current rounded bg-transparent hover:bg-hestia-primary hover:text-white" onClick={() => handleDelete(key.id)}>Revoke</button>
            </div>
          </div>
        ))
      )}
    </div>
  )
}
