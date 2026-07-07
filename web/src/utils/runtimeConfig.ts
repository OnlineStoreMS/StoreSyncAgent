declare global {
  interface Window {
    __RUNTIME_CONFIG__?: {
      portalUrl?: string
    }
  }
}

/** 部署时由 runtime-config.js + PUBLIC_HOST / VITE_PORTAL_URL 覆盖 */
function defaultPortalUrl(): string {
  if (typeof window !== 'undefined' && window.location?.hostname) {
    const { protocol, hostname } = window.location
    if (hostname !== 'localhost' && hostname !== '127.0.0.1') {
      return `${protocol}//${hostname}:5174`
    }
  }
  return 'http://localhost:5174'
}

export function getPortalUrl(): string {
  return (
    window.__RUNTIME_CONFIG__?.portalUrl?.trim()
    || import.meta.env.VITE_PORTAL_URL?.trim()
    || defaultPortalUrl()
  )
}
