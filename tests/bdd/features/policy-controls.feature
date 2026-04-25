Feature: Administrator policy controls
  As an Orivis administrator
  I want to control platform access globally and by subject
  So I can enforce security requirements for OAuth2 and forward-auth

  Scenario: Configure platform allowlist mode
    Given an admin is authenticated with JWT bearer token
    When the admin sets platform policy mode to allowlist
    And the admin adds approved platform identifiers
    Then only listed platforms can authenticate via Orivis

  Scenario: user override deny takes precedence
    Given platform mode is allow_any
    And a user has user override deny for a platform
    When the user attempts OAuth2 or forward-auth access
    Then Orivis denies access with reason user override deny

  Scenario: group override allow applies when no user override deny exists
    Given platform mode is denylist
    And the user belongs to a group with group override allow
    And no user override deny exists
    When the user attempts forward-auth access
    Then Orivis allows access with reason group override allow

  Scenario: platform denylist applies without subject override
    Given platform mode is denylist
    And a platform entry is denied
    And no user or group override exists
    When a user attempts OAuth2 authorization
    Then Orivis denies access with reason platform denylist
