Feature: Forward-auth decisions for reverse proxy integration
  As a reverse proxy
  I want Orivis to evaluate each request
  So I can allow or block traffic consistently

  Scenario: Authenticated request is allowed
    Given a user is authenticated in Orivis
    And platform policy allows the target host
    When the proxy calls /v1/forward-auth/check with forwarded headers
    Then Orivis returns status 200
    And the response includes user identity headers

  Scenario: Unauthenticated request is rejected
    Given no valid user session is present
    When the proxy calls /v1/forward-auth/check with forwarded headers
    Then Orivis returns status 401
    And decision is unauthenticated

  Scenario: Authenticated request is forbidden by policy
    Given a user is authenticated in Orivis
    And platform policy is denylist for the target host
    When the proxy calls /v1/forward-auth/check with forwarded headers
    Then Orivis returns status 403
    And decision reasonCode is platform denylist
