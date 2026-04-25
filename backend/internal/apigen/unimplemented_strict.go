package apigen

import (
	"context"
	"errors"
)

var ErrNotImplemented = errors.New("endpoint not implemented")

type UnimplementedStrictServer struct{}

func NewUnimplementedStrictServer() *UnimplementedStrictServer {
	return &UnimplementedStrictServer{}
}

var _ StrictServerInterface = (*UnimplementedStrictServer)(nil)

func (s *UnimplementedStrictServer) GetOidcDiscoveryDocument(ctx context.Context, request GetOidcDiscoveryDocumentRequestObject) (GetOidcDiscoveryDocumentResponseObject, error) {
	_ = ctx
	return nil, ErrNotImplemented
}

func (s *UnimplementedStrictServer) AuthorizeOAuth2Client(ctx context.Context, request AuthorizeOAuth2ClientRequestObject) (AuthorizeOAuth2ClientResponseObject, error) {
	_ = ctx
	return nil, ErrNotImplemented
}

func (s *UnimplementedStrictServer) IntrospectOAuth2Token(ctx context.Context, request IntrospectOAuth2TokenRequestObject) (IntrospectOAuth2TokenResponseObject, error) {
	_ = ctx
	return nil, ErrNotImplemented
}

func (s *UnimplementedStrictServer) GetOAuth2Jwks(ctx context.Context, request GetOAuth2JwksRequestObject) (GetOAuth2JwksResponseObject, error) {
	_ = ctx
	return nil, ErrNotImplemented
}

func (s *UnimplementedStrictServer) RevokeOAuth2Token(ctx context.Context, request RevokeOAuth2TokenRequestObject) (RevokeOAuth2TokenResponseObject, error) {
	_ = ctx
	return nil, ErrNotImplemented
}

func (s *UnimplementedStrictServer) ExchangeOAuth2Token(ctx context.Context, request ExchangeOAuth2TokenRequestObject) (ExchangeOAuth2TokenResponseObject, error) {
	_ = ctx
	return nil, ErrNotImplemented
}

func (s *UnimplementedStrictServer) GetOAuth2UserInfo(ctx context.Context, request GetOAuth2UserInfoRequestObject) (GetOAuth2UserInfoResponseObject, error) {
	_ = ctx
	return nil, ErrNotImplemented
}

func (s *UnimplementedStrictServer) ListAuditEvents(ctx context.Context, request ListAuditEventsRequestObject) (ListAuditEventsResponseObject, error) {
	_ = ctx
	return nil, ErrNotImplemented
}

func (s *UnimplementedStrictServer) GetForwardAuthGroupPlatformOverride(ctx context.Context, request GetForwardAuthGroupPlatformOverrideRequestObject) (GetForwardAuthGroupPlatformOverrideResponseObject, error) {
	_ = ctx
	return nil, ErrNotImplemented
}

func (s *UnimplementedStrictServer) PutForwardAuthGroupPlatformOverride(ctx context.Context, request PutForwardAuthGroupPlatformOverrideRequestObject) (PutForwardAuthGroupPlatformOverrideResponseObject, error) {
	_ = ctx
	return nil, ErrNotImplemented
}

func (s *UnimplementedStrictServer) GetForwardAuthPlatformPolicy(ctx context.Context, request GetForwardAuthPlatformPolicyRequestObject) (GetForwardAuthPlatformPolicyResponseObject, error) {
	_ = ctx
	return nil, ErrNotImplemented
}

func (s *UnimplementedStrictServer) PutForwardAuthPlatformPolicy(ctx context.Context, request PutForwardAuthPlatformPolicyRequestObject) (PutForwardAuthPlatformPolicyResponseObject, error) {
	_ = ctx
	return nil, ErrNotImplemented
}

func (s *UnimplementedStrictServer) GetForwardAuthUserPlatformOverride(ctx context.Context, request GetForwardAuthUserPlatformOverrideRequestObject) (GetForwardAuthUserPlatformOverrideResponseObject, error) {
	_ = ctx
	return nil, ErrNotImplemented
}

func (s *UnimplementedStrictServer) PutForwardAuthUserPlatformOverride(ctx context.Context, request PutForwardAuthUserPlatformOverrideRequestObject) (PutForwardAuthUserPlatformOverrideResponseObject, error) {
	_ = ctx
	return nil, ErrNotImplemented
}

func (s *UnimplementedStrictServer) GetOAuth2GroupPlatformOverride(ctx context.Context, request GetOAuth2GroupPlatformOverrideRequestObject) (GetOAuth2GroupPlatformOverrideResponseObject, error) {
	_ = ctx
	return nil, ErrNotImplemented
}

func (s *UnimplementedStrictServer) PutOAuth2GroupPlatformOverride(ctx context.Context, request PutOAuth2GroupPlatformOverrideRequestObject) (PutOAuth2GroupPlatformOverrideResponseObject, error) {
	_ = ctx
	return nil, ErrNotImplemented
}

func (s *UnimplementedStrictServer) GetOAuth2PlatformPolicy(ctx context.Context, request GetOAuth2PlatformPolicyRequestObject) (GetOAuth2PlatformPolicyResponseObject, error) {
	_ = ctx
	return nil, ErrNotImplemented
}

func (s *UnimplementedStrictServer) PutOAuth2PlatformPolicy(ctx context.Context, request PutOAuth2PlatformPolicyRequestObject) (PutOAuth2PlatformPolicyResponseObject, error) {
	_ = ctx
	return nil, ErrNotImplemented
}

func (s *UnimplementedStrictServer) GetOAuth2UserPlatformOverride(ctx context.Context, request GetOAuth2UserPlatformOverrideRequestObject) (GetOAuth2UserPlatformOverrideResponseObject, error) {
	_ = ctx
	return nil, ErrNotImplemented
}

func (s *UnimplementedStrictServer) PutOAuth2UserPlatformOverride(ctx context.Context, request PutOAuth2UserPlatformOverrideRequestObject) (PutOAuth2UserPlatformOverrideResponseObject, error) {
	_ = ctx
	return nil, ErrNotImplemented
}

func (s *UnimplementedStrictServer) VerifyTotpChallenge(ctx context.Context, request VerifyTotpChallengeRequestObject) (VerifyTotpChallengeResponseObject, error) {
	_ = ctx
	return nil, ErrNotImplemented
}

func (s *UnimplementedStrictServer) GetWebAuthnChallengeOptions(ctx context.Context, request GetWebAuthnChallengeOptionsRequestObject) (GetWebAuthnChallengeOptionsResponseObject, error) {
	_ = ctx
	return nil, ErrNotImplemented
}

func (s *UnimplementedStrictServer) VerifyWebAuthnChallenge(ctx context.Context, request VerifyWebAuthnChallengeRequestObject) (VerifyWebAuthnChallengeResponseObject, error) {
	_ = ctx
	return nil, ErrNotImplemented
}

func (s *UnimplementedStrictServer) LoginWithPassword(ctx context.Context, request LoginWithPasswordRequestObject) (LoginWithPasswordResponseObject, error) {
	_ = ctx
	return nil, ErrNotImplemented
}

func (s *UnimplementedStrictServer) CompleteExternalProviderLogin(ctx context.Context, request CompleteExternalProviderLoginRequestObject) (CompleteExternalProviderLoginResponseObject, error) {
	_ = ctx
	return nil, ErrNotImplemented
}

func (s *UnimplementedStrictServer) StartExternalProviderLogin(ctx context.Context, request StartExternalProviderLoginRequestObject) (StartExternalProviderLoginResponseObject, error) {
	_ = ctx
	return nil, ErrNotImplemented
}

func (s *UnimplementedStrictServer) RegisterWithPassword(ctx context.Context, request RegisterWithPasswordRequestObject) (RegisterWithPasswordResponseObject, error) {
	_ = ctx
	return nil, ErrNotImplemented
}

func (s *UnimplementedStrictServer) CheckForwardAuth(ctx context.Context, request CheckForwardAuthRequestObject) (CheckForwardAuthResponseObject, error) {
	_ = ctx
	return nil, ErrNotImplemented
}

func (s *UnimplementedStrictServer) GetCurrentUser(ctx context.Context, request GetCurrentUserRequestObject) (GetCurrentUserResponseObject, error) {
	_ = ctx
	return nil, ErrNotImplemented
}

func (s *UnimplementedStrictServer) ListCurrentUserAuthMethods(ctx context.Context, request ListCurrentUserAuthMethodsRequestObject) (ListCurrentUserAuthMethodsResponseObject, error) {
	_ = ctx
	return nil, ErrNotImplemented
}

func (s *UnimplementedStrictServer) LinkCurrentUserAuthMethod(ctx context.Context, request LinkCurrentUserAuthMethodRequestObject) (LinkCurrentUserAuthMethodResponseObject, error) {
	_ = ctx
	return nil, ErrNotImplemented
}

func (s *UnimplementedStrictServer) UnlinkCurrentUserAuthMethod(ctx context.Context, request UnlinkCurrentUserAuthMethodRequestObject) (UnlinkCurrentUserAuthMethodResponseObject, error) {
	_ = ctx
	return nil, ErrNotImplemented
}
