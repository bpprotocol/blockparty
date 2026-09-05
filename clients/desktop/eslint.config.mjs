import js from '@eslint/js'
import ts from 'typescript-eslint'
import vue from 'eslint-plugin-vue'
import prettier from 'eslint-config-prettier/flat'

export default ts.config(
  {
    ignores: [
      '.nuxt/**',
      '.output/**',
      'dist-electron/**',
      'release/**',
      'release-client-only/**',
      'node_modules/**',
      'electron/gen/**',
      'electron/testdata/**',
    ],
  },
  js.configs.recommended,
  ...ts.configs.recommended,
  ...vue.configs['flat/recommended'],
  // The packaging scripts (#48) are plain ESM run by Node, not by the renderer.
  {
    files: ['scripts/**/*.mjs'],
    languageOptions: {
      globals: { process: 'readonly', console: 'readonly' },
    },
  },
  {
    files: ['**/*.vue'],
    languageOptions: {
      parserOptions: {
        parser: ts.parser,
      },
    },
  },
  // Disable ESLint rules that conflict with Prettier (formatting is Prettier's job).
  prettier,
)
