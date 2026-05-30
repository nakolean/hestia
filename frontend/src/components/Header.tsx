import { Settings } from 'lucide-preact'

export function Header() {
  const openSettings = () => {
    window.dispatchEvent(new Event('open-settings-modal'))
  }

  return (
    <header class="app-header">
      <div class="container-wrapper">
        <h1 style="margin:0">Hestia</h1>
        <button class="p-2 border border-current rounded bg-transparent text-hestia-text hover:bg-hestia-primary hover:text-white" onClick={openSettings}>
          <Settings />
        </button>
      </div>
    </header>
  )
}
