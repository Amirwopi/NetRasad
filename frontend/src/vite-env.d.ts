interface WailsRuntime {
  EventsOn(eventName: string, cb: (data: unknown) => void): () => void
  EventsOnce(eventName: string, cb: (data: unknown) => void): () => void
  EventsEmit(eventName: string, data?: unknown): void
  EventsOff(nameOrCallback: string | ((...args: unknown[]) => void)): void
  BrowserOpenURL(url: string): void
}

interface Window {
  runtime?: WailsRuntime
}
