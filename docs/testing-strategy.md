# Testing Strategy (TDD)

Orivis starts with contract and behavior tests before implementation.

## Contract Tests

Location: `tests/contract/openapi.contract.test.ts`

Coverage includes:

- OpenAPI validity
- Required endpoints
- Required security on admin endpoints
- Forward-auth response contract
- OAuth2 grant type support in schema

## BDD Specifications

Location: `tests/bdd/features/`

Feature sets:

- Authentication and method linking
- OAuth2 provider behavior
- Forward-auth behavior
- Admin policy precedence behavior

## Commands

```bash
npm run lint:openapi
npm run test:contract
npm run test:bdd
```
