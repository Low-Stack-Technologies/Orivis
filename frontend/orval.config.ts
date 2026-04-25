import { defineConfig } from 'orval'

export default defineConfig({
  orivis: {
    input: {
      target: '../openapi/orivis.openapi.yaml'
    },
    output: {
      mode: 'split',
      target: './src/api/generated/orivis.ts',
      schemas: './src/api/generated/model',
      client: 'react-query',
      mock: false,
      override: {
        mutator: {
          path: './src/api/http-client.ts',
          name: 'customFetch'
        }
      }
    }
  }
})
