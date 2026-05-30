export interface ShoppingItem {
  id: number
  text: string
  purchased: boolean
  created_at: string
  purchased_at: string | null
}

interface Props {
  item: ShoppingItem
  onToggle: (id: number, purchased: boolean) => void
  onDelete: (id: number) => void
}

export function ShoppingItemRow({ item, onToggle, onDelete }: Props) {
  return (
    <div class="grid grid-cols-2 gap-2 p-3 border border-hestia-border mb-1" style={{
      textDecoration: item.purchased ? 'line-through' : 'none',
      color: item.purchased ? 'var(--hestia-text-muted)' : 'inherit',
    }}>
      <label class="col-span-1 flex items-center gap-2 cursor-pointer">
        <input type="checkbox" checked={item.purchased} onChange={() => onToggle(item.id, !item.purchased)} />
        <span>{item.text}</span>
      </label>
      <button class="text-xs px-2 py-1 border border-current rounded bg-transparent hover:bg-hestia-primary hover:text-white" onClick={() => onDelete(item.id)}>✕</button>
    </div>
  )
}
