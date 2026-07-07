declare global {
  interface Window {
    __RUNTIME_CONFIG__?: {
      portalUrl?: string
    }
  }
}

export function getPortalUrl(): string {
  return (
    window.__RUNTIME_CONFIG__?.portalUrl?.trim()
    || import.meta.env.VITE_PORTAL_URL?.trim()
    || 'http://localhost:5174'
  )
}
