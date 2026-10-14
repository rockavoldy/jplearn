const captureBtn = document.getElementById('capture')
const statusEl = document.getElementById('status')
const optionsLink = document.getElementById('options')

optionsLink.addEventListener('click', (e) => {
    e.preventDefault()
    chrome.runtime.openOptionsPage()
})

function setStatus(text, kind) {
    statusEl.textContent = text
    statusEl.className = 'status' + (kind ? ' ' + kind : '')
}

captureBtn.addEventListener('click', async () => {
    captureBtn.disabled = true
    setStatus('Extracting article…')
    try {
        const result = await chrome.runtime.sendMessage({ type: 'JP_READER_CAPTURE' })
        if (result?.ok) {
            setStatus(`Saved: ${result.article?.title || 'article'}`, 'ok')
        } else {
            setStatus(result?.error || 'Unknown error', 'err')
        }
    } catch (err) {
        setStatus(String(err?.message || err), 'err')
    } finally {
        captureBtn.disabled = false
    }
})
