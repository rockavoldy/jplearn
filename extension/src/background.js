// Background service worker (Manifest V3). Clicking the toolbar icon scrapes
// the active page and POSTs the result to the JP Reader backend, showing
// success/error feedback on the icon badge.
const DEFAULT_API_BASE = 'http://localhost:8080'
const BADGE_CLEAR_MS = 4000

async function getSettings() {
    const { apiBase, apiKey } = await chrome.storage.sync.get(['apiBase', 'apiKey'])
    return {
        apiBase: (apiBase || DEFAULT_API_BASE).replace(/\/+$/, ''),
        apiKey: (apiKey || '').trim(),
    }
}

async function getActiveTab() {
    const [tab] = await chrome.tabs.query({ active: true, currentWindow: true })
    return tab
}

/** Shows a transient badge (e.g. "✓" green / "!" red) on the toolbar icon. */
function showBadge(text, color) {
    chrome.action.setBadgeBackgroundColor({ color })
    chrome.action.setBadgeText({ text })
    setTimeout(() => {
        chrome.action.setBadgeText({ text: '' })
    }, BADGE_CLEAR_MS)
}

/**
 * Extracts the article from the active tab and uploads it.
 * @returns {Promise<{ok: boolean, article?: object, error?: string}>}
 */
async function captureAndSend() {
    const tab = await getActiveTab()
    if (!tab?.id) {
        return { ok: false, error: 'No active tab found.' }
    }

    let response
    try {
        response = await chrome.tabs.sendMessage(tab.id, { type: 'JP_READER_EXTRACT' })
    } catch {
        // Content script not injected (e.g. restricted page). Inject on demand.
        try {
            await chrome.scripting.executeScript({
                target: { tabId: tab.id },
                files: ['content.js'],
            })
            response = await chrome.tabs.sendMessage(tab.id, { type: 'JP_READER_EXTRACT' })
        } catch (err) {
            return { ok: false, error: 'Cannot access this page: ' + String(err?.message || err) }
        }
    }

    if (!response?.ok) {
        return { ok: false, error: response?.error || 'Extraction failed.' }
    }

    const { apiBase, apiKey } = await getSettings()
    try {
        const res = await fetch(`${apiBase}/api/articles`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                ...(apiKey ? { 'X-API-Key': apiKey } : {}),
            },
            body: JSON.stringify(response.article),
        })
        if (res.status === 401) {
            return { ok: false, error: 'Unauthorized — set a valid API key in the extension options.' }
        }
        if (!res.ok) {
            let detail = res.statusText
            try {
                detail = (await res.json()).error || detail
            } catch {
                /* ignore */
            }
            return { ok: false, error: `Backend error ${res.status}: ${detail}` }
        }
        const article = await res.json()
        return { ok: true, article }
    } catch (err) {
        return { ok: false, error: 'Failed to reach backend: ' + String(err?.message || err) }
    }
}

// Clicking the toolbar icon scrapes the page immediately.
chrome.action.onClicked.addListener(async () => {
    chrome.action.setBadgeBackgroundColor({ color: '#6b7280' })
    chrome.action.setBadgeText({ text: '…' })

    const result = await captureAndSend()
    if (result?.ok) {
        showBadge('✓', '#15803d')
    } else {
        console.warn('JP Reader capture failed:', result?.error)
        showBadge('!', '#b91c1c')
    }
})

// Kept for programmatic triggers (e.g. context menus, future popup).
chrome.runtime.onMessage.addListener((message, _sender, sendResponse) => {
    if (message?.type === 'JP_READER_CAPTURE') {
        captureAndSend().then(sendResponse)
        return true // async response
    }
    return undefined
})
