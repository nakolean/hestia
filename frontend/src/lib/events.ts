type EventHandler = () => void

const listeners = new Map<string, Set<EventHandler>>()

export function subscribe(event: string, handler: EventHandler): () => void {
  if (!listeners.has(event)) {
    listeners.set(event, new Set())
  }
  listeners.get(event)!.add(handler)

  return () => {
    listeners.get(event)?.delete(handler)
  }
}

export function publish(event: string): void {
  listeners.get(event)?.forEach(handler => handler())
}