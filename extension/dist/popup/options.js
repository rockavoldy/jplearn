const DEFAULT_API_BASE = 'http://localhost:8080'
const apiBaseInput = document.getElementById('apiBase')
const apiKeyInput = document.getElementById('apiKey')
const saveBtn = document.getElementById('save')
const savedEl = document.getElementById('saved')

async function restore() {
    const { apiBase, apiKey } = await chrome.storage.sync.get(['apiBase', 'apiKey'])
    apiBaseInput.value = apiBase || DEFAULT_API_BASE
    apiKeyInput.value = apiKey || ''
}

saveBtn.addEventListener('click', async () => {
    const apiBase = (apiBaseInput.value || DEFAULT_API_BASE).replace(/\/+$/, '')
    const apiKey = (apiKeyInput.value || '').trim()
    await chrome.storage.sync.set({ apiBase, apiKey })
    apiBaseInput.value = apiBase
    apiKeyInput.value = apiKey
    savedEl.textContent = 'Saved'
    setTimeout(() => {
        savedEl.textContent = ''
    }, 1500)
})

restore()
