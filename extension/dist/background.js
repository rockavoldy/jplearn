// Background service worker (Manifest V3). Orchestrates extraction on the
// active tab and POSTs the result to the JP Reader backend.
const DEFAULT_API_BASE = 'http://localhost:8080'

async function getApiBase() {
    const { apiBase } = await chrome.storage.sync.get('apiBase')
    return (apiBase || DEFAULT_API_BASE).replace(/\/+$/, '')
}

async function getActiveTab() {
    const [tab] = await chrome.tabs.query({ active: true, currentWindow: true })
    return tab
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

    const apiBase = await getApiBase()
    try {
        const res = await fetch(`${apiBase}/api/articles`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(response.article),
        })
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

chrome.runtime.onMessage.addListener((message, _sender, sendResponse) => {
    if (message?.type === 'JP_READER_CAPTURE') {
        captureAndSend().then(sendResponse)
        return true // async response
    }
    return undefined
})
