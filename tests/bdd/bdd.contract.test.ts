import { readFileSync } from 'node:fs'
import path from 'node:path'
import { describe, expect, it } from 'vitest'

const features = [
  'tests/bdd/features/authentication.feature',
  'tests/bdd/features/oauth2-provider.feature',
  'tests/bdd/features/forward-auth.feature',
  'tests/bdd/features/policy-controls.feature'
]

describe('BDD specification coverage', () => {
  it('contains required feature files', () => {
    for (const featurePath of features) {
      const fullPath = path.resolve(process.cwd(), featurePath)
      const body = readFileSync(fullPath, 'utf8')
      expect(body.length).toBeGreaterThan(0)
      expect(body).toContain('Feature:')
      expect(body).toContain('Scenario:')
    }
  })

  it('captures multi-method auth and account-linking scenarios', () => {
    const auth = readFileSync(path.resolve(process.cwd(), features[0]), 'utf8')
    expect(auth).toContain('password')
    expect(auth).toContain('TOTP')
    expect(auth).toContain('passkey')
    expect(auth).toContain('Google')
  })

  it('captures policy precedence behavior for user and group overrides', () => {
    const policy = readFileSync(path.resolve(process.cwd(), features[3]), 'utf8')
    expect(policy).toContain('user override deny')
    expect(policy).toContain('group override allow')
    expect(policy).toContain('platform denylist')
  })
})
