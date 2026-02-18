import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import { readFileSync } from 'fs'
import { resolve, dirname } from 'path'

// Preprocessor que permite <style src="./Component.css"> en archivos .svelte.
// Lee el archivo CSS externo y lo inyecta como si estuviera inline,
// para que Svelte aplique el scoping normal de clases.
function externalStyles() {
  return {
    style({ attributes, filename }) {
      if (!attributes.src) return
      const cssPath = resolve(dirname(filename), attributes.src)
      const code = readFileSync(cssPath, 'utf-8')
      return { code }
    },
  }
}

export default defineConfig({
  plugins: [
    svelte({
      preprocess: [externalStyles()],
    }),
  ],
})
