# Policy Model

Orivis applies policy for both OAuth2 and forward-auth decisions.

## Policy Layers

1. Platform-level mode
2. Group-level override
3. User-level override

## Decisions

- `allow`
- `deny`
- `inherit`

`inherit` means continue evaluating lower-priority layers.

## Recommended Precedence

1. user override deny
2. user override allow
3. group override deny
4. group override allow
5. platform denylist
6. platform allowlist match
7. tenant default decision

If no rule allows access, the default is deny.

## Modes

- `allow_any`: all platforms allowed unless overridden
- `allowlist`: only listed platforms allowed
- `denylist`: all except listed denied platforms
