const DEFAULT_API_BASE = 'http://localhost:8080'
const input = document.getElementById('apiBase')
const saveBtn = document.getElementById('save')
const savedEl = document.getElementById('saved')

async function restore() {
    const { apiBase } = await chrome.storage.sync.get('apiBase')
    input.value = apiBase || DEFAULT_API_BASE
}

saveBtn.addEventListener('click', async () => {
    const value = (input.value || DEFAULT_API_BASE).replace(/\/+$/, '')
    await chrome.storage.sync.set({ apiBase: value })
    input.value = value
    savedEl.textContent = 'Saved'
    setTimeout(() => {
        savedEl.textContent = ''
    }, 1500)
})

restore()
