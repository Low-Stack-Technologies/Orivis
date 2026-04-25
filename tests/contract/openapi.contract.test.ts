import { readFileSync } from 'node:fs'
import path from 'node:path'
import SwaggerParser from '@apidevtools/swagger-parser'
import yaml from 'js-yaml'
import { describe, expect, it } from 'vitest'

const specPath = path.resolve(process.cwd(), 'openapi/orivis.openapi.yaml')
const raw = readFileSync(specPath, 'utf8')
const spec = yaml.load(raw) as Record<string, any>

describe('OpenAPI contract', () => {
  it('is valid OpenAPI', async () => {
    await expect(SwaggerParser.validate(spec as any)).resolves.toBeDefined()
  })

  it('uses OpenAPI 3.1', () => {
    expect(spec.openapi).toBe('3.1.0')
  })

  it('contains key auth and policy endpoints', () => {
    const expectedPaths = [
      '/v1/auth/login/password',
      '/v1/auth/challenge/totp',
      '/v1/auth/challenge/webauthn/options',
      '/oauth2/authorize',
      '/oauth2/token',
      '/v1/forward-auth/check',
      '/v1/admin/policies/oauth2/platforms/{platformId}',
      '/v1/admin/policies/forward-auth/platforms/{platformId}'
    ]

    for (const endpoint of expectedPaths) {
      expect(spec.paths?.[endpoint], `missing path ${endpoint}`).toBeDefined()
    }
  })

  it('protects all admin endpoints with bearer auth', () => {
    const adminPaths = Object.entries(spec.paths || {}).filter(([route]) =>
      route.startsWith('/v1/admin/')
    )

    for (const [route, operations] of adminPaths) {
      for (const [method, operation] of Object.entries(operations as Record<string, any>)) {
        const security = (operation as any).security
        expect(security, `${method.toUpperCase()} ${route} missing security`).toBeDefined()
        expect(
          security.some((item: Record<string, unknown>) => Object.prototype.hasOwnProperty.call(item, 'bearerAuth')),
          `${method.toUpperCase()} ${route} should require bearerAuth`
        ).toBe(true)
      }
    }
  })

  it('defines Traefik-style forward-auth responses', () => {
    const operation = spec.paths?.['/v1/forward-auth/check']?.get
    expect(operation).toBeDefined()
    expect(operation.responses?.['200']).toBeDefined()
    expect(operation.responses?.['401']).toBeDefined()
    expect(operation.responses?.['403']).toBeDefined()
  })

  it('supports OAuth2 token grant types for AS flows', () => {
    const tokenRequest =
      spec.components?.schemas?.OAuth2TokenRequest?.properties?.grant_type?.enum ?? []

    expect(tokenRequest).toContain('authorization_code')
    expect(tokenRequest).toContain('refresh_token')
    expect(tokenRequest).toContain('client_credentials')
  })
})
