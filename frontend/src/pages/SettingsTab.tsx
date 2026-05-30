import { ApiKeyManager } from '../components/ApiKeyManager'

export interface SettingsModalProps {
  onClose: () => void
}

export function SettingsModal({ onClose }: SettingsModalProps) {
  return (
    <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={onClose}>
      <div class="max-w-[400px] mx-auto p-4" onClick={e => e.stopPropagation()}>
        <div>
          <h2>Settings</h2>
          <ApiKeyManager />
          <div class="grid grid-cols-2 gap-2">
            <button class="px-4 py-2 border border-current rounded bg-transparent hover:bg-artemis-primary hover:text-white" onClick={onClose}>Close</button>
          </div>
        </div>
      </div>
    </div>
  )
}
