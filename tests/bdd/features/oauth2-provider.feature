Feature: OAuth2 provider mode
  As a third-party application
  I want to delegate user login to Orivis
  So I can use Orivis identities in my app

  Scenario: Authorization code flow with PKCE succeeds
    Given an OAuth client is registered and allowed
    And a user has an active Orivis session
    When the client sends the user to /oauth2/authorize with PKCE
    Then Orivis redirects with an authorization code
    When the client exchanges code at /oauth2/token
    Then Orivis returns an access token and optional refresh token

  Scenario: Authorization blocked by platform policy
    Given an OAuth client is in a platform denylist
    When the client starts an authorization request
    Then Orivis denies the request with a policy error

  Scenario: User-level deny beats platform allow
    Given platform policy allows the client
    And the user has a user override deny for that platform
    When the user attempts OAuth authorization
    Then Orivis denies the authorization request
