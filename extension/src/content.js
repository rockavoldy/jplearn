// Content script: clones the DOM, extracts the main article with Readability,
// and forwards the cleaned HTML to the background service worker.
import { Readability } from '@mozilla/readability'

const MAX_HTML_BYTES = 5 * 1024 * 1024 // 5MB scrape payload limit

function extractArticle() {
    // Clone the document so Readability can mutate it without affecting the page.
    const clone = document.cloneNode(true)
    const reader = new Readability(clone)
    const parsed = reader.parse()

    if (!parsed || !parsed.content) {
        return null
    }
    return {
        url: location.href,
        title: parsed.title || document.title,
        html: parsed.content,
    }
}

function utf8ByteLength(str) {
    return new Blob([str]).size
}

chrome.runtime.onMessage.addListener((message, _sender, sendResponse) => {
    if (message?.type !== 'JP_READER_EXTRACT') {
        return undefined
    }

    try {
        const article = extractArticle()
        if (!article) {
            sendResponse({ ok: false, error: 'Readability could not extract an article from this page.' })
            return true
        }
        if (utf8ByteLength(article.html) > MAX_HTML_BYTES) {
            sendResponse({ ok: false, error: 'Extracted article exceeds the 5MB payload limit.' })
            return true
        }
        sendResponse({ ok: true, article })
    } catch (err) {
        sendResponse({ ok: false, error: String(err?.message || err) })
    }
    return true
})
