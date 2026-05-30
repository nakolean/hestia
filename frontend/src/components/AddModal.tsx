import { useState } from 'preact/hooks'
import { post } from '../api/client'
import { publish } from '../lib/events'

export interface AddModalProps {
  mode: 'chore' | 'item'
  onClose: () => void
}

export function AddModal({ mode, onClose }: AddModalProps) {
  const [name, setName] = useState('')
  const [desc, setDesc] = useState('')
  const [freqNum, setFreqNum] = useState(1)
  const [freqUnit, setFreqUnit] = useState('days')
  const [error, setError] = useState('')

  const handleSubmit = async (e: Event) => {
    e.preventDefault()
    if (!name.trim()) { setError('Name is required'); return }
    try {
      if (mode === 'chore') {
        await post('/chores', { name, description: desc, frequency_num: freqNum, frequency_unit: freqUnit })
        publish('chore-added')
      } else {
        await post('/items', { text: name })
        publish('item-added')
      }
      onClose()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save')
    }
  }

  return (
    <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={onClose}>
      <div class="max-w-[400px] mx-auto p-4" onClick={e => e.stopPropagation()}>
        <div>
          <h2>Add {mode === 'chore' ? 'Chore' : 'Item'}</h2>
          <form onSubmit={handleSubmit}>
            <label class="block mb-2">
              {mode === 'chore' ? 'Name' : 'Item'}
              <input autofocus class="w-full border border-artemis-border rounded p-2" value={name} onInput={e => setName((e.target as HTMLInputElement).value)} placeholder={mode === 'chore' ? 'e.g., Water plants' : 'e.g., Milk'} required />
            </label>
            {mode === 'chore' && (
              <>
                <label class="block mb-2">
                  Description
                  <input class="w-full border border-artemis-border rounded p-2" value={desc} onInput={e => setDesc((e.target as HTMLInputElement).value)} placeholder="Optional" />
                </label>
                <label class="block mb-2">
                  Repeat every
                  <input type="number" min="1" class="border border-artemis-border rounded p-2" value={freqNum} onInput={e => setFreqNum(parseInt((e.target as HTMLInputElement).value || '1'))} />
                  <select class="border border-artemis-border rounded p-2" value={freqUnit} onChange={e => setFreqUnit((e.target as HTMLSelectElement).value)}>
                    <option value="hours">hours</option>
                    <option value="days">days</option>
                    <option value="weeks">weeks</option>
                  </select>
                </label>
              </>
            )}
            {error && <small class="text-red-500">{error}</small>}
            <div class="grid grid-cols-2 gap-2">
              <button type="submit" class="px-4 py-2 bg-artemis-primary text-white rounded">Save</button>
              <button type="button" class="px-4 py-2 border border-current rounded bg-transparent hover:bg-artemis-primary hover:text-white" onClick={onClose}>Cancel</button>
            </div>
          </form>
        </div>
      </div>
    </div>
  )
}
