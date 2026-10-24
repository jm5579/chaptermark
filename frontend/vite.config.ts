import {defineConfig, type Plugin} from 'vite'
import react from '@vitejs/plugin-react'
import {writeFileSync} from 'node:fs'
import {resolve} from 'node:path'

// Go's //go:embed needs frontend/dist to contain a file even before the first
// frontend build (e.g. on a fresh clone), so keep a committed placeholder alive.
const keepDist = (): Plugin => ({
  name: 'keep-dist-placeholder',
  closeBundle() {
    writeFileSync(resolve(__dirname, 'dist/gitkeep'), '')
  },
})

export default defineConfig({
  plugins: [react(), keepDist()],
})
