// Bundles the content script (which imports @mozilla/readability) into the
// dist/ folder. The service worker and popup are plain ES modules and are
// copied as-is.
import { build, context } from 'esbuild'
import { cp, mkdir, rm } from 'node:fs/promises'

const watch = process.argv.includes('--watch')

async function prepare() {
    await rm('dist', { recursive: true, force: true })
    await mkdir('dist/popup', { recursive: true })
    await cp('manifest.json', 'dist/manifest.json')
    await cp('src/background.js', 'dist/background.js')
    await cp('popup', 'dist/popup', { recursive: true })
}

const buildOptions = {
    entryPoints: ['src/content.js'],
    bundle: true,
    format: 'iife',
    target: ['chrome110'],
    outfile: 'dist/content.js',
    logLevel: 'info',
}

await prepare()

if (watch) {
    const ctx = await context(buildOptions)
    await ctx.watch()
    console.log('watching extension sources…')
} else {
    await build(buildOptions)
    console.log('extension built into dist/')
}
