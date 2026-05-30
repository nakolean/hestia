import { useState } from 'preact/hooks'
import { useLocation } from 'preact-iso'
import { AddModal } from './AddModal'
import { Plus } from 'lucide-preact'

export function FloatingActionButton() {
  const { path } = useLocation()
  const [modalOpen, setModalOpen] = useState(false)

  const mode = path === '/shopping' ? 'item' : 'chore'

  return (
    <>
      <button class="app-fab" onClick={() => setModalOpen(true)}>
        <Plus size={24} />
      </button>
      {modalOpen && (
        <AddModal
          mode={mode}
          onClose={() => setModalOpen(false)}
        />
      )}
    </>
  )
}
