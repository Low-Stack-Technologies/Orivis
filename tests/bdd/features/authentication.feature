Feature: User authentication with multiple sign-in methods
  As an Orivis end user
  I want to sign in with different methods
  So I can access my account even if one method is unavailable

  Scenario: Sign in with password and TOTP
    Given a user exists with a password and TOTP enabled
    When the user submits a valid username and password
    Then Orivis returns a challenge_required response for TOTP
    When the user submits a valid TOTP challenge code
    Then Orivis returns an authenticated session

  Scenario: Sign in with a passkey
    Given a user has a registered passkey credential
    When the user requests WebAuthn challenge options
    And the user returns a valid passkey assertion
    Then Orivis returns an authenticated session

  Scenario: Link Google provider to existing user
    Given a user is authenticated with password
    When the user starts OAuth login with Google using link intent
    And Google callback succeeds for the same email identity
    Then the user has a linked Google sign-in method
