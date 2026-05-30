import { useState, useEffect } from 'preact/hooks'
import { Settings } from 'lucide-preact'
import { get, patch, del } from '../api/client'
import { ShoppingItemRow, ShoppingItem } from '../components/ShoppingItem'
import { subscribe } from '../lib/events'

export function ShoppingTab({ path: _path }: { path?: string }) {
  const [items, setItems] = useState<ShoppingItem[]>([])
  const [loading, setLoading] = useState(true)

  const loadItems = async () => {
    try {
      const data = await get('/items')
      setItems(data.items as ShoppingItem[])
    } catch {
      // Handle gracefully
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { loadItems() }, [])

  useEffect(() => {
    return subscribe('item-added', loadItems)
  }, [])

  const togglePurchased = async (id: number, purchased: boolean) => {
    try {
      await patch(`/items/${id}`, { purchased })
      await loadItems()
    } catch {
      alert('Failed to update item')
    }
  }

  const deleteItem = async (id: number) => {
    await del(`/items/${id}`)
    await loadItems()
  }

  const active = items.filter(i => !i.purchased).sort((a, b) => a.id - b.id)
  const purchased = items.filter(i => i.purchased)

  if (loading) return <p>Loading...</p>

  return (
    <div>
      <div class="flex justify-between items-center p-2">
        <h2>Shopping List</h2>
        <button class="p-2 border border-current rounded bg-transparent text-artemis-text hover:bg-artemis-primary hover:text-white" onClick={() => window.dispatchEvent(new Event('open-settings-modal'))}>
          <Settings />
        </button>
      </div>
      <div>
        {active.map(item => (
          <ShoppingItemRow key={item.id} item={item} onToggle={togglePurchased} onDelete={deleteItem} />
        ))}
        {purchased.map(item => (
          <ShoppingItemRow key={item.id} item={item} onToggle={togglePurchased} onDelete={deleteItem} />
        ))}
      </div>
      {!items.length && <p class="text-artemis-text-muted">No items yet. Use the + button to add one.</p>}
    </div>
  )
}
