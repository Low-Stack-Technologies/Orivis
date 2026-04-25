package server

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Low-Stack-Technologies/Orivis/backend/internal/apigen"
	"github.com/Low-Stack-Technologies/Orivis/backend/internal/db"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/argon2"
	"golang.org/x/oauth2"
)

type Server struct {
	apigen.Unimplemented

	pool       *pgxpool.Pool
	queries    *db.Queries
	cfg        Config
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	oauthGoogle *oauth2.Config
}

type accessClaims struct {
	Email    string   `json:"email,omitempty"`
	Groups   []string `json:"groups,omitempty"`
	IsAdmin  bool     `json:"is_admin,omitempty"`
	ClientID string   `json:"client_id,omitempty"`
	jwt.RegisteredClaims
}

type principal struct {
	UserID   string
	Email    string
	Username string
	Groups   []string
	IsAdmin  bool
}

func New(ctx context.Context, cfg Config) (*Server, error) {
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	privateKey, err := cfg.LoadOrCreatePrivateKey()
	if err != nil {
		return nil, err
	}

	s := &Server{
		pool:        pool,
		queries:     db.New(pool),
		cfg:         cfg,
		privateKey:  privateKey,
		publicKey:   &privateKey.PublicKey,
		oauthGoogle: cfg.GoogleOAuthConfig(),
	}

	if err := s.ensureDefaultOAuthClient(ctx); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Server) Close() {
	s.pool.Close()
}

func (s *Server) RegisterWithPassword(w http.ResponseWriter, r *http.Request) {
	var req apigen.RegisterRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	hash, err := hashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "password_hash_error", err.Error())
		return
	}

	user, err := s.queries.CreateUser(r.Context(), db.CreateUserParams{
		Email:    string(req.Email),
		Username: req.Username,
		PasswordHash: pgtype.Text{String: hash, Valid: true},
	})
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "user_exists", "email or username already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "create_user_failed", err.Error())
		return
	}

	_, _ = s.queries.CreateUserAuthMethod(r.Context(), db.CreateUserAuthMethodParams{
		UserID:          user.ID,
		MethodType:      string(apigen.AuthMethodTypePassword),
		ProviderSubject: pgtype.Text{},
		SecretRef:       pgtype.Text{},
		Metadata:        jsonBytes(map[string]any{"source": "local"}),
	})

	result, err := s.authenticateAndIssueSession(r.Context(), user.ID, user.Email, user.Username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "issue_session_failed", err.Error())
		return
	}

	s.setSessionCookie(w, result.Session.AccessToken)
	writeJSON(w, http.StatusCreated, result)
}

func (s *Server) LoginWithPassword(w http.ResponseWriter, r *http.Request) {
	var req apigen.PasswordLoginRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	user, err := s.queries.GetUserByIdentifier(r.Context(), req.Identifier)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "invalid identifier or password")
		return
	}

	if !user.PasswordHash.Valid || !verifyPassword(req.Password, user.PasswordHash.String) {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "invalid identifier or password")
		return
	}

	totpFactor, err := s.queries.GetTotpFactorByUserID(r.Context(), user.ID)
	if err == nil && totpFactor.Enabled {
		challenge, err := s.createChallenge(r.Context(), "totp_login", user.ID, map[string]any{"reason": "password_login"}, time.Now().Add(5*time.Minute))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "challenge_error", err.Error())
			return
		}

		resp := apigen.AuthResult{
			Status:            apigen.AuthResultStatusChallengeRequired,
			RequiredChallenge: ptr(apigen.AuthResultRequiredChallengeTotp),
			ChallengeId:       &challenge.ID,
		}
		writeJSON(w, http.StatusOK, resp)
		return
	}

	result, err := s.authenticateAndIssueSession(r.Context(), user.ID, user.Email, user.Username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "issue_session_failed", err.Error())
		return
	}

	s.setSessionCookie(w, result.Session.AccessToken)
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) VerifyTotpChallenge(w http.ResponseWriter, r *http.Request) {
	var req apigen.TotpChallengeRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	challenge, err := s.queries.GetAuthChallengeByID(r.Context(), req.ChallengeId)
	if err != nil {
		writeError(w, http.StatusNotFound, "challenge_not_found", "challenge does not exist")
		return
	}

	if challenge.ConsumedAt.Valid || challenge.ExpiresAt.Time.Before(time.Now()) || challenge.ChallengeType != "totp_login" {
		writeError(w, http.StatusBadRequest, "challenge_invalid", "challenge is expired or invalid")
		return
	}

	factor, err := s.queries.GetTotpFactorByUserID(r.Context(), challenge.UserID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "totp_not_configured", "totp not configured")
		return
	}

	if !totp.Validate(req.Code, factor.Secret) {
		writeError(w, http.StatusUnauthorized, "totp_invalid", "invalid TOTP code")
		return
	}

	_ = s.queries.ConsumeAuthChallenge(r.Context(), challenge.ID)

	user, err := s.queries.GetUserByID(r.Context(), challenge.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "user_not_found", "user not found")
		return
	}

	result, err := s.authenticateAndIssueSession(r.Context(), user.ID, user.Email, user.Username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "issue_session_failed", err.Error())
		return
	}

	s.setSessionCookie(w, result.Session.AccessToken)
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) GetWebAuthnChallengeOptions(w http.ResponseWriter, r *http.Request) {
	var req apigen.WebAuthnOptionsRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	user, err := s.queries.GetUserByIdentifier(r.Context(), req.Identifier)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "user_not_found", "user not found")
		return
	}

	challengeValue, err := randomURLToken(32)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "challenge_generation_failed", err.Error())
		return
	}

	challenge, err := s.createChallenge(r.Context(), "webauthn_login", user.ID, map[string]any{"challenge": challengeValue}, time.Now().Add(5*time.Minute))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "challenge_error", err.Error())
		return
	}

	_ = challenge
	resp := apigen.WebAuthnRequestOptions{
		Challenge:        challengeValue,
		RpId:             s.cfg.WebAuthnRPID,
		Timeout:          60000,
		UserVerification: "required",
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) VerifyWebAuthnChallenge(w http.ResponseWriter, r *http.Request) {
	var req apigen.WebAuthnVerifyRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	challenge, err := s.queries.GetAuthChallengeByID(r.Context(), req.ChallengeId)
	if err != nil {
		writeError(w, http.StatusNotFound, "challenge_not_found", "challenge not found")
		return
	}

	if challenge.ConsumedAt.Valid || challenge.ExpiresAt.Time.Before(time.Now()) || challenge.ChallengeType != "webauthn_login" {
		writeError(w, http.StatusBadRequest, "challenge_invalid", "challenge is expired or invalid")
		return
	}

	credentialID, _ := req.Credential["id"].(string)
	if credentialID == "" {
		writeError(w, http.StatusBadRequest, "credential_missing", "credential id is required")
		return
	}

	cred, err := s.queries.GetWebAuthnCredentialByCredentialID(r.Context(), credentialID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "credential_not_found", "passkey credential not found")
		return
	}

	if cred.UserID != challenge.UserID {
		writeError(w, http.StatusUnauthorized, "credential_mismatch", "credential does not belong to challenge subject")
		return
	}

	_ = s.queries.ConsumeAuthChallenge(r.Context(), challenge.ID)

	user, err := s.queries.GetUserByID(r.Context(), challenge.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "user_not_found", "user not found")
		return
	}

	result, err := s.authenticateAndIssueSession(r.Context(), user.ID, user.Email, user.Username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "issue_session_failed", err.Error())
		return
	}

	s.setSessionCookie(w, result.Session.AccessToken)
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) StartExternalProviderLogin(w http.ResponseWriter, r *http.Request, provider apigen.StartExternalProviderLoginParamsProvider) {
	if string(provider) != "google" {
		writeError(w, http.StatusBadRequest, "provider_not_supported", "only google is currently supported")
		return
	}
	if s.oauthGoogle == nil {
		writeError(w, http.StatusInternalServerError, "provider_not_configured", "google oauth is not configured")
		return
	}

	var req apigen.ExternalProviderStartRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	intent := string(apigen.ExternalProviderStartRequestIntentLogin)
	if req.Intent != nil {
		intent = string(*req.Intent)
	}

	challenge, err := s.createChallenge(r.Context(), "google_oauth", "", map[string]any{
		"redirectUri": req.RedirectUri,
		"intent":      intent,
	}, time.Now().Add(10*time.Minute))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "challenge_error", err.Error())
		return
	}

	url := s.oauthGoogle.AuthCodeURL(challenge.ID, oauth2.AccessTypeOffline)
	writeJSON(w, http.StatusOK, apigen.ExternalProviderStartResponse{
		AuthorizationUrl: url,
		State:            challenge.ID,
	})
}

func (s *Server) CompleteExternalProviderLogin(w http.ResponseWriter, r *http.Request, provider apigen.CompleteExternalProviderLoginParamsProvider) {
	if string(provider) != "google" {
		writeError(w, http.StatusBadRequest, "provider_not_supported", "only google is currently supported")
		return
	}
	if s.oauthGoogle == nil {
		writeError(w, http.StatusInternalServerError, "provider_not_configured", "google oauth is not configured")
		return
	}

	var req apigen.ExternalProviderCallbackRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	challenge, err := s.queries.GetAuthChallengeByID(r.Context(), req.State)
	if err != nil || challenge.ChallengeType != "google_oauth" || challenge.ConsumedAt.Valid || challenge.ExpiresAt.Time.Before(time.Now()) {
		writeError(w, http.StatusBadRequest, "invalid_state", "oauth state is invalid")
		return
	}

	tok, err := s.oauthGoogle.Exchange(r.Context(), req.Code)
	if err != nil {
		writeError(w, http.StatusBadRequest, "provider_exchange_failed", err.Error())
		return
	}

	googleSub, googleEmail, err := s.fetchGoogleIdentity(r.Context(), tok.AccessToken)
	if err != nil {
		writeError(w, http.StatusBadRequest, "provider_identity_failed", err.Error())
		return
	}

	userID, username, err := s.resolveUserFromGoogleIdentity(r.Context(), googleSub, googleEmail)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "identity_link_failed", err.Error())
		return
	}

	_ = s.queries.ConsumeAuthChallenge(r.Context(), challenge.ID)

	result, err := s.authenticateAndIssueSession(r.Context(), userID, googleEmail, username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "issue_session_failed", err.Error())
		return
	}

	s.setSessionCookie(w, result.Session.AccessToken)
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	p, err := s.requirePrincipal(r.Context(), r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	writeJSON(w, http.StatusOK, apigen.User{
		Id:       p.UserID,
		Email:    openapi_types.Email(p.Email),
		Username: p.Username,
		Groups:   p.Groups,
	})
}

func (s *Server) ListCurrentUserAuthMethods(w http.ResponseWriter, r *http.Request) {
	p, err := s.requirePrincipal(r.Context(), r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	rows, err := s.queries.ListUserAuthMethods(r.Context(), p.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list_methods_failed", err.Error())
		return
	}

	items := make([]apigen.AuthMethod, 0, len(rows))
	for _, row := range rows {
		metadata := map[string]any{}
		_ = json.Unmarshal(row.Metadata, &metadata)
		items = append(items, apigen.AuthMethod{
			Id:        row.ID,
			Type:      apigen.AuthMethodType(row.MethodType),
			CreatedAt: row.CreatedAt.Time,
			Metadata:  &metadata,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) LinkCurrentUserAuthMethod(w http.ResponseWriter, r *http.Request) {
	p, err := s.requirePrincipal(r.Context(), r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	var req apigen.LinkAuthMethodRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	metadata := map[string]any{}
	if req.Payload != nil {
		metadata = *req.Payload
	}

	var providerSubject pgtype.Text
	var secretRef pgtype.Text

	switch req.Type {
	case apigen.LinkAuthMethodRequestTypeTotp:
		secret, err := totp.GenerateCodeCustom("JBSWY3DPEHPK3PXP", time.Now(), totp.ValidateOpts{Period: 30, Digits: 6, Skew: 1})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "totp_setup_failed", err.Error())
			return
		}
		metadata["secret"] = secret
		if err := s.queries.UpsertTotpFactor(r.Context(), db.UpsertTotpFactorParams{UserID: p.UserID, Secret: secret, Enabled: true}); err != nil {
			writeError(w, http.StatusInternalServerError, "totp_setup_failed", err.Error())
			return
		}
		secretRef = pgtype.Text{String: "totp", Valid: true}

	case apigen.LinkAuthMethodRequestTypePasskey:
		credentialID, _ := metadata["credentialId"].(string)
		if credentialID == "" {
			writeError(w, http.StatusBadRequest, "credential_missing", "payload.credentialId is required for passkey")
			return
		}
		_, err := s.queries.CreateWebAuthnCredential(r.Context(), db.CreateWebAuthnCredentialParams{
			UserID:       p.UserID,
			CredentialID: credentialID,
			PublicKey:    []byte{},
			SignCount:    0,
			Transports:   []string{},
			Metadata:     jsonBytes(metadata),
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "passkey_link_failed", err.Error())
			return
		}
		providerSubject = pgtype.Text{String: credentialID, Valid: true}

	case apigen.LinkAuthMethodRequestTypeOauthGoogle:
		providerSubjectString, _ := metadata["providerSubject"].(string)
		if providerSubjectString == "" {
			writeError(w, http.StatusBadRequest, "provider_subject_missing", "payload.providerSubject is required for oauth_google")
			return
		}
		providerSubject = pgtype.Text{String: providerSubjectString, Valid: true}

	default:
		writeError(w, http.StatusBadRequest, "method_type_invalid", "unsupported auth method type")
		return
	}

	row, err := s.queries.CreateUserAuthMethod(r.Context(), db.CreateUserAuthMethodParams{
		UserID:          p.UserID,
		MethodType:      string(req.Type),
		ProviderSubject: providerSubject,
		SecretRef:       secretRef,
		Metadata:        jsonBytes(metadata),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "link_method_failed", err.Error())
		return
	}

	respMetadata := metadata
	writeJSON(w, http.StatusCreated, apigen.AuthMethod{
		Id:        row.ID,
		Type:      apigen.AuthMethodType(row.MethodType),
		CreatedAt: row.CreatedAt.Time,
		Metadata:  &respMetadata,
	})
}

func (s *Server) UnlinkCurrentUserAuthMethod(w http.ResponseWriter, r *http.Request, methodId apigen.MethodIdPath) {
	p, err := s.requirePrincipal(r.Context(), r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	affected, err := s.queries.DeleteUserAuthMethod(r.Context(), db.DeleteUserAuthMethodParams{
		MethodID: string(methodId),
		UserID:   p.UserID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "unlink_method_failed", err.Error())
		return
	}
	if affected == 0 {
		writeError(w, http.StatusNotFound, "method_not_found", "auth method not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) GetOidcDiscoveryDocument(w http.ResponseWriter, r *http.Request) {
	responseTypes := []string{"code"}
	subjectTypes := []string{"public"}
	algs := []string{"RS256"}

	writeJSON(w, http.StatusOK, apigen.OidcDiscovery{
		Issuer:                           s.cfg.IssuerURL,
		AuthorizationEndpoint:            s.cfg.IssuerURL + "/oauth2/authorize",
		TokenEndpoint:                    s.cfg.IssuerURL + "/oauth2/token",
		UserinfoEndpoint:                 s.cfg.IssuerURL + "/oauth2/userinfo",
		JwksUri:                          s.cfg.IssuerURL + "/oauth2/jwks",
		ResponseTypesSupported:           &responseTypes,
		SubjectTypesSupported:            &subjectTypes,
		IdTokenSigningAlgValuesSupported: &algs,
	})
}

func (s *Server) AuthorizeOAuth2Client(w http.ResponseWriter, r *http.Request, params apigen.AuthorizeOAuth2ClientParams) {
	client, err := s.queries.GetOAuthClientByID(r.Context(), params.ClientId)
	if err != nil {
		writeError(w, http.StatusBadRequest, "client_invalid", "client not found")
		return
	}

	if !contains(client.RedirectUris, params.RedirectUri) {
		writeError(w, http.StatusBadRequest, "redirect_uri_invalid", "redirect_uri is not registered for client")
		return
	}

	principal, err := s.requirePrincipal(r.Context(), r)
	if err != nil {
		loginURL := s.cfg.FrontendURL + "/login"
		http.Redirect(w, r, loginURL, http.StatusFound)
		return
	}

	allowed, reason := s.evaluatePolicy(r.Context(), "oauth2", params.ClientId, principal.UserID, principal.Groups)
	if !allowed {
		writeError(w, http.StatusForbidden, "policy_denied", reason)
		return
	}

	if client.RequirePkce && params.CodeChallenge == nil {
		writeError(w, http.StatusBadRequest, "pkce_required", "code_challenge is required")
		return
	}

	rawCode, err := randomURLToken(32)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "code_generation_failed", err.Error())
		return
	}

	codeChallenge := pgtype.Text{}
	if params.CodeChallenge != nil {
		codeChallenge = pgtype.Text{String: string(*params.CodeChallenge), Valid: true}
	}

	codeChallengeMethod := pgtype.Text{}
	if params.CodeChallengeMethod != nil {
		codeChallengeMethod = pgtype.Text{String: string(*params.CodeChallengeMethod), Valid: true}
	}

	_, err = s.queries.CreateOAuthAuthorizationCode(r.Context(), db.CreateOAuthAuthorizationCodeParams{
		CodeHash:            hashToken(rawCode),
		ClientID:            client.ID,
		UserID:              principal.UserID,
		RedirectUri:         params.RedirectUri,
		Scope:               params.Scope,
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: codeChallengeMethod,
		ExpiresAt:           toPgTime(time.Now().Add(5 * time.Minute)),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "code_store_failed", err.Error())
		return
	}

	redirectURL, _ := url.Parse(params.RedirectUri)
	query := redirectURL.Query()
	query.Set("code", rawCode)
	query.Set("state", params.State)
	redirectURL.RawQuery = query.Encode()
	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}

func (s *Server) ExchangeOAuth2Token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_form", "invalid form body")
		return
	}

	grantType := r.PostFormValue("grant_type")
	client, clientSecret, ok := r.BasicAuth()
	if !ok || client == "" {
		writeError(w, http.StatusUnauthorized, "invalid_client", "client basic auth is required")
		return
	}

	clientRow, err := s.queries.GetOAuthClientByID(r.Context(), client)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid_client", "client does not exist")
		return
	}
	if clientRow.ClientSecretHash.Valid && !verifyPassword(clientSecret, clientRow.ClientSecretHash.String) {
		writeError(w, http.StatusUnauthorized, "invalid_client", "client secret mismatch")
		return
	}

	switch grantType {
	case string(apigen.OAuth2TokenRequestGrantTypeAuthorizationCode):
		s.exchangeAuthorizationCode(w, r, clientRow)
	case string(apigen.OAuth2TokenRequestGrantTypeRefreshToken):
		s.exchangeRefreshToken(w, r, clientRow)
	case string(apigen.OAuth2TokenRequestGrantTypeClientCredentials):
		s.exchangeClientCredentials(w, r, clientRow)
	default:
		writeError(w, http.StatusBadRequest, "unsupported_grant_type", "grant_type not supported")
	}
}

func (s *Server) RevokeOAuth2Token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_form", "invalid form body")
		return
	}

	token := r.PostFormValue("token")
	if token == "" {
		writeError(w, http.StatusBadRequest, "token_required", "token is required")
		return
	}

	tokenHash := hashToken(token)
	_ = s.queries.RevokeOAuthAccessTokenByHash(r.Context(), tokenHash)
	_ = s.queries.RevokeOAuthRefreshTokenByHash(r.Context(), tokenHash)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) IntrospectOAuth2Token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_form", "invalid form body")
		return
	}

	token := r.PostFormValue("token")
	if token == "" {
		writeError(w, http.StatusBadRequest, "token_required", "token is required")
		return
	}

	hash := hashToken(token)
	if access, err := s.queries.GetOAuthAccessTokenByHash(r.Context(), hash); err == nil {
		active := !access.RevokedAt.Valid && access.ExpiresAt.Time.After(time.Now())
		resp := apigen.OAuth2IntrospectResponse{Active: active}
		if active {
			resp.Sub = stringPtr(access.UserID)
			resp.ClientId = &access.ClientID
			resp.Scope = &access.Scope
			exp := int(access.ExpiresAt.Time.Unix())
			resp.Exp = &exp
			iat := int(access.CreatedAt.Time.Unix())
			resp.Iat = &iat
		}
		writeJSON(w, http.StatusOK, resp)
		return
	}

	writeJSON(w, http.StatusOK, apigen.OAuth2IntrospectResponse{Active: false})
}

func (s *Server) GetOAuth2Jwks(w http.ResponseWriter, r *http.Request) {
	n := base64.RawURLEncoding.EncodeToString(s.publicKey.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(s.publicKey.E)).Bytes())

	writeJSON(w, http.StatusOK, apigen.JwksDocument{
		Keys: []map[string]any{{
			"kty": "RSA",
			"kid": s.cfg.SigningKeyID,
			"alg": "RS256",
			"use": "sig",
			"n":   n,
			"e":   e,
		}},
	})
}

func (s *Server) GetOAuth2UserInfo(w http.ResponseWriter, r *http.Request) {
	claims, err := s.parseAccessClaims(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "invalid bearer token")
		return
	}

	resp := apigen.UserInfo{
		Sub:               claims.Subject,
		Email:             openapi_types.Email(claims.Email),
		PreferredUsername: nil,
	}
	if len(claims.Groups) > 0 {
		resp.Groups = &claims.Groups
	}
	verified := true
	resp.EmailVerified = &verified

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) CheckForwardAuth(w http.ResponseWriter, r *http.Request, params apigen.CheckForwardAuthParams) {
	p, err := s.requirePrincipal(r.Context(), r)
	if err != nil {
		writeJSONWithStatus(w, http.StatusUnauthorized, apigen.ForwardAuthDecision{
			Decision:   apigen.ForwardAuthDecisionDecisionUnauthenticated,
			ReasonCode: apigen.ForwardAuthDecisionReasonCodeMissingSession,
		})
		return
	}

	allowed, reason := s.evaluatePolicy(r.Context(), "forward_auth", params.XForwardedHost, p.UserID, p.Groups)
	if !allowed {
		writeJSONWithStatus(w, http.StatusForbidden, apigen.ForwardAuthDecision{
			Decision:   apigen.ForwardAuthDecisionDecisionDeny,
			ReasonCode: apigen.ForwardAuthDecisionReasonCodePlatformDenylist,
		})
		return
	}

	w.Header().Set("X-Orivis-Subject", p.UserID)
	w.Header().Set("X-Orivis-Email", p.Email)
	w.Header().Set("X-Orivis-Groups", strings.Join(p.Groups, ","))
	w.Header().Set("X-Orivis-Decision-Reason", reason)

	decision := apigen.ForwardAuthDecision{
		Decision:   apigen.ForwardAuthDecisionDecisionAllow,
		ReasonCode: apigen.ForwardAuthDecisionReasonCodeAllowed,
		Subject: &struct {
			Email  *openapi_types.Email `json:"email,omitempty"`
			Groups *[]string            `json:"groups,omitempty"`
			UserId *string              `json:"userId,omitempty"`
		}{
			Email:  ptr(openapi_types.Email(p.Email)),
			Groups: &p.Groups,
			UserId: &p.UserID,
		},
	}

	writeJSONWithStatus(w, http.StatusOK, decision)
}

func (s *Server) GetOAuth2PlatformPolicy(w http.ResponseWriter, r *http.Request, platformId apigen.PlatformIdPath) {
	s.getPlatformPolicy(w, r, "oauth2", string(platformId))
}

func (s *Server) PutOAuth2PlatformPolicy(w http.ResponseWriter, r *http.Request, platformId apigen.PlatformIdPath) {
	s.putPlatformPolicy(w, r, "oauth2", string(platformId))
}

func (s *Server) GetForwardAuthPlatformPolicy(w http.ResponseWriter, r *http.Request, platformId apigen.PlatformIdPath) {
	s.getPlatformPolicy(w, r, "forward_auth", string(platformId))
}

func (s *Server) PutForwardAuthPlatformPolicy(w http.ResponseWriter, r *http.Request, platformId apigen.PlatformIdPath) {
	s.putPlatformPolicy(w, r, "forward_auth", string(platformId))
}

func (s *Server) GetOAuth2UserPlatformOverride(w http.ResponseWriter, r *http.Request, userId apigen.UserIdPath, platformId apigen.PlatformIdPath) {
	s.getOverride(w, r, "oauth2", "user", string(userId), string(platformId))
}

func (s *Server) PutOAuth2UserPlatformOverride(w http.ResponseWriter, r *http.Request, userId apigen.UserIdPath, platformId apigen.PlatformIdPath) {
	s.putOverride(w, r, "oauth2", "user", string(userId), string(platformId))
}

func (s *Server) GetOAuth2GroupPlatformOverride(w http.ResponseWriter, r *http.Request, groupId apigen.GroupIdPath, platformId apigen.PlatformIdPath) {
	s.getOverride(w, r, "oauth2", "group", string(groupId), string(platformId))
}

func (s *Server) PutOAuth2GroupPlatformOverride(w http.ResponseWriter, r *http.Request, groupId apigen.GroupIdPath, platformId apigen.PlatformIdPath) {
	s.putOverride(w, r, "oauth2", "group", string(groupId), string(platformId))
}

func (s *Server) GetForwardAuthUserPlatformOverride(w http.ResponseWriter, r *http.Request, userId apigen.UserIdPath, platformId apigen.PlatformIdPath) {
	s.getOverride(w, r, "forward_auth", "user", string(userId), string(platformId))
}

func (s *Server) PutForwardAuthUserPlatformOverride(w http.ResponseWriter, r *http.Request, userId apigen.UserIdPath, platformId apigen.PlatformIdPath) {
	s.putOverride(w, r, "forward_auth", "user", string(userId), string(platformId))
}

func (s *Server) GetForwardAuthGroupPlatformOverride(w http.ResponseWriter, r *http.Request, groupId apigen.GroupIdPath, platformId apigen.PlatformIdPath) {
	s.getOverride(w, r, "forward_auth", "group", string(groupId), string(platformId))
}

func (s *Server) PutForwardAuthGroupPlatformOverride(w http.ResponseWriter, r *http.Request, groupId apigen.GroupIdPath, platformId apigen.PlatformIdPath) {
	s.putOverride(w, r, "forward_auth", "group", string(groupId), string(platformId))
}

func (s *Server) ListAuditEvents(w http.ResponseWriter, r *http.Request, params apigen.ListAuditEventsParams) {
	if _, err := s.requireAdmin(r.Context(), r); err != nil {
		writeError(w, http.StatusUnauthorized, "admin_required", "admin bearer token is required")
		return
	}

	page := 1
	if params.Page != nil && *params.Page > 0 {
		page = *params.Page
	}

	pageSize := 50
	if params.PageSize != nil && *params.PageSize > 0 {
		pageSize = *params.PageSize
	}

	offset := (page - 1) * pageSize
	rows, err := s.queries.ListAuditEvents(r.Context(), db.ListAuditEventsParams{Limit: int32(pageSize), Offset: int32(offset)})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "audit_query_failed", err.Error())
		return
	}

	total, _ := s.queries.CountAuditEvents(r.Context())
	items := make([]apigen.AuditEvent, 0, len(rows))
	for _, row := range rows {
		metadata := map[string]any{}
		_ = json.Unmarshal(row.Metadata, &metadata)
		item := apigen.AuditEvent{
			Id:        fmt.Sprintf("%d", row.ID),
			ActorType: apigen.AuditEventActorType(row.ActorType),
			ActorId:   row.ActorID,
			Action:    row.Action,
			Metadata:  &metadata,
			CreatedAt: row.CreatedAt.Time,
		}
		if row.TargetType.Valid {
			item.TargetType = &row.TargetType.String
		}
		if row.TargetID.Valid {
			item.TargetId = &row.TargetID.String
		}
		items = append(items, item)
	}

	writeJSON(w, http.StatusOK, apigen.AuditEventPage{
		Items:    items,
		Page:     page,
		PageSize: pageSize,
		Total:    int(total),
	})
}

func (s *Server) getPlatformPolicy(w http.ResponseWriter, r *http.Request, surface string, platformID string) {
	if _, err := s.requireAdmin(r.Context(), r); err != nil {
		writeError(w, http.StatusUnauthorized, "admin_required", "admin bearer token is required")
		return
	}

	row, err := s.queries.GetPlatformPolicy(r.Context(), db.GetPlatformPolicyParams{AuthSurface: surface, PlatformID: platformID})
	if err != nil {
		writeError(w, http.StatusNotFound, "policy_not_found", "platform policy not found")
		return
	}

	writeJSON(w, http.StatusOK, apigen.PlatformPolicy{
		PlatformId: row.PlatformID,
		Mode:       apigen.PolicyMode(row.Mode),
		Entries:    row.Entries,
		UpdatedAt:  row.UpdatedAt.Time,
	})
}

func (s *Server) putPlatformPolicy(w http.ResponseWriter, r *http.Request, surface string, platformID string) {
	admin, err := s.requireAdmin(r.Context(), r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "admin_required", "admin bearer token is required")
		return
	}

	var req apigen.PlatformPolicy
	if !decodeJSON(w, r, &req) {
		return
	}

	row, err := s.queries.UpsertPlatformPolicy(r.Context(), db.UpsertPlatformPolicyParams{
		AuthSurface: surface,
		PlatformID:  platformID,
		Mode:        string(req.Mode),
		Entries:     req.Entries,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, "policy_upsert_failed", err.Error())
		return
	}

	s.audit(r.Context(), "admin", admin.UserID, "platform_policy_upsert", "platform", platformID, map[string]any{"surface": surface})

	writeJSON(w, http.StatusOK, apigen.PlatformPolicy{
		PlatformId: row.PlatformID,
		Mode:       apigen.PolicyMode(row.Mode),
		Entries:    row.Entries,
		UpdatedAt:  row.UpdatedAt.Time,
	})
}

func (s *Server) getOverride(w http.ResponseWriter, r *http.Request, surface, subjectType, subjectID, platformID string) {
	if _, err := s.requireAdmin(r.Context(), r); err != nil {
		writeError(w, http.StatusUnauthorized, "admin_required", "admin bearer token is required")
		return
	}

	row, err := s.queries.GetSubjectPolicyOverride(r.Context(), db.GetSubjectPolicyOverrideParams{
		AuthSurface: surface,
		SubjectType: subjectType,
		SubjectID:   subjectID,
		PlatformID:  platformID,
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "override_not_found", "subject override not found")
		return
	}

	resp := apigen.SubjectPolicyOverride{
		SubjectId:   row.SubjectID,
		SubjectType: apigen.SubjectPolicyOverrideSubjectType(row.SubjectType),
		PlatformId:  row.PlatformID,
		Decision:    apigen.PolicyDecision(row.Decision),
	}
	if row.Reason.Valid {
		resp.Reason = &row.Reason.String
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) putOverride(w http.ResponseWriter, r *http.Request, surface, subjectType, subjectID, platformID string) {
	admin, err := s.requireAdmin(r.Context(), r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "admin_required", "admin bearer token is required")
		return
	}

	var req apigen.SubjectPolicyOverride
	if !decodeJSON(w, r, &req) {
		return
	}

	reason := pgtype.Text{}
	if req.Reason != nil {
		reason = pgtype.Text{String: *req.Reason, Valid: true}
	}

	row, err := s.queries.UpsertSubjectPolicyOverride(r.Context(), db.UpsertSubjectPolicyOverrideParams{
		AuthSurface: surface,
		SubjectType: subjectType,
		SubjectID:   subjectID,
		PlatformID:  platformID,
		Decision:    string(req.Decision),
		Reason:      reason,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, "override_upsert_failed", err.Error())
		return
	}

	s.audit(r.Context(), "admin", admin.UserID, "subject_override_upsert", subjectType, subjectID, map[string]any{"surface": surface, "platform": platformID})

	resp := apigen.SubjectPolicyOverride{
		SubjectId:   row.SubjectID,
		SubjectType: apigen.SubjectPolicyOverrideSubjectType(row.SubjectType),
		PlatformId:  row.PlatformID,
		Decision:    apigen.PolicyDecision(row.Decision),
	}
	if row.Reason.Valid {
		resp.Reason = &row.Reason.String
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) evaluatePolicy(ctx context.Context, surface, platformID, userID string, groups []string) (bool, string) {
	if row, err := s.queries.GetSubjectPolicyOverride(ctx, db.GetSubjectPolicyOverrideParams{AuthSurface: surface, SubjectType: "user", SubjectID: userID, PlatformID: platformID}); err == nil {
		if row.Decision == "deny" {
			return false, "user_override_deny"
		}
		if row.Decision == "allow" {
			return true, "user_override_allow"
		}
	}

	for _, groupID := range groups {
		if row, err := s.queries.GetSubjectPolicyOverride(ctx, db.GetSubjectPolicyOverrideParams{AuthSurface: surface, SubjectType: "group", SubjectID: groupID, PlatformID: platformID}); err == nil {
			if row.Decision == "deny" {
				return false, "group_override_deny"
			}
			if row.Decision == "allow" {
				return true, "group_override_allow"
			}
		}
	}

	platformPolicy, err := s.queries.GetPlatformPolicy(ctx, db.GetPlatformPolicyParams{AuthSurface: surface, PlatformID: platformID})
	if err != nil {
		return false, "tenant_default_deny"
	}

	switch platformPolicy.Mode {
	case "allow_any":
		return true, "allowed"
	case "allowlist":
		if contains(platformPolicy.Entries, platformID) {
			return true, "allowed"
		}
		return false, "platform_allowlist_missing"
	case "denylist":
		if contains(platformPolicy.Entries, platformID) {
			return false, "platform_denylist"
		}
		return true, "allowed"
	default:
		return false, "tenant_default_deny"
	}
}

func (s *Server) exchangeAuthorizationCode(w http.ResponseWriter, r *http.Request, client db.OauthClient) {
	code := r.PostFormValue("code")
	redirectURI := r.PostFormValue("redirect_uri")
	codeVerifier := r.PostFormValue("code_verifier")
	if code == "" || redirectURI == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "code and redirect_uri are required")
		return
	}

	stored, err := s.queries.GetOAuthAuthorizationCodeByHash(r.Context(), hashToken(code))
	if err != nil || stored.ClientID != client.ID || stored.RedirectUri != redirectURI {
		writeError(w, http.StatusBadRequest, "invalid_grant", "authorization code is invalid")
		return
	}

	if stored.ConsumedAt.Valid || stored.ExpiresAt.Time.Before(time.Now()) {
		writeError(w, http.StatusBadRequest, "invalid_grant", "authorization code expired or already used")
		return
	}

	if client.RequirePkce {
		if !stored.CodeChallenge.Valid || codeVerifier == "" {
			writeError(w, http.StatusBadRequest, "invalid_grant", "pkce code_verifier required")
			return
		}
		computed := pkceS256(codeVerifier)
		if subtle.ConstantTimeCompare([]byte(computed), []byte(stored.CodeChallenge.String)) != 1 {
			writeError(w, http.StatusBadRequest, "invalid_grant", "pkce verification failed")
			return
		}
	}

	_ = s.queries.ConsumeOAuthAuthorizationCode(r.Context(), stored.ID)

	accessToken, jti, exp, err := s.issueJWT(stored.UserID, client.ID, stored.Scope)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token_issue_failed", err.Error())
		return
	}

	accessRow, err := s.queries.CreateOAuthAccessToken(r.Context(), db.CreateOAuthAccessTokenParams{
		TokenHash: hashToken(accessToken),
		TokenJti:  jti,
		ClientID:  client.ID,
		UserID:    pgUUID(stored.UserID),
		Scope:     stored.Scope,
		ExpiresAt: toPgTime(exp),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token_store_failed", err.Error())
		return
	}

	rawRefresh, err := randomURLToken(48)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token_issue_failed", err.Error())
		return
	}

	refreshExpiry := time.Now().Add(time.Duration(s.cfg.RefreshTokenTTLHours) * time.Hour)
	_, err = s.queries.CreateOAuthRefreshToken(r.Context(), db.CreateOAuthRefreshTokenParams{
		TokenHash:     hashToken(rawRefresh),
		AccessTokenID: accessRow.ID,
		ClientID:      client.ID,
		UserID:        pgUUID(stored.UserID),
		Scope:         stored.Scope,
		ExpiresAt:     toPgTime(refreshExpiry),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token_store_failed", err.Error())
		return
	}

	scope := stored.Scope
	writeJSON(w, http.StatusOK, apigen.OAuth2TokenResponse{
		TokenType:    apigen.OAuth2TokenResponseTokenTypeBearer,
		AccessToken:  accessToken,
		ExpiresIn:    int(time.Until(exp).Seconds()),
		RefreshToken: &rawRefresh,
		Scope:        &scope,
	})
}

func (s *Server) exchangeRefreshToken(w http.ResponseWriter, r *http.Request, client db.OauthClient) {
	rawRefresh := r.PostFormValue("refresh_token")
	if rawRefresh == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "refresh_token is required")
		return
	}

	stored, err := s.queries.GetOAuthRefreshTokenByHash(r.Context(), hashToken(rawRefresh))
	if err != nil || stored.ClientID != client.ID || stored.RevokedAt.Valid || stored.ExpiresAt.Time.Before(time.Now()) {
		writeError(w, http.StatusBadRequest, "invalid_grant", "refresh token is invalid")
		return
	}

	_ = s.queries.RevokeOAuthRefreshTokenByHash(r.Context(), hashToken(rawRefresh))

	accessToken, jti, exp, err := s.issueJWT(stored.UserID, client.ID, stored.Scope)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token_issue_failed", err.Error())
		return
	}

	accessRow, err := s.queries.CreateOAuthAccessToken(r.Context(), db.CreateOAuthAccessTokenParams{
		TokenHash: hashToken(accessToken),
		TokenJti:  jti,
		ClientID:  client.ID,
		UserID:    pgUUID(stored.UserID),
		Scope:     stored.Scope,
		ExpiresAt: toPgTime(exp),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token_store_failed", err.Error())
		return
	}

	newRefresh, err := randomURLToken(48)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token_issue_failed", err.Error())
		return
	}
	_, err = s.queries.CreateOAuthRefreshToken(r.Context(), db.CreateOAuthRefreshTokenParams{
		TokenHash:     hashToken(newRefresh),
		AccessTokenID: accessRow.ID,
		ClientID:      client.ID,
		UserID:        pgUUID(stored.UserID),
		Scope:         stored.Scope,
		ExpiresAt:     toPgTime(time.Now().Add(time.Duration(s.cfg.RefreshTokenTTLHours) * time.Hour)),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token_store_failed", err.Error())
		return
	}

	scope := stored.Scope
	writeJSON(w, http.StatusOK, apigen.OAuth2TokenResponse{
		TokenType:    apigen.OAuth2TokenResponseTokenTypeBearer,
		AccessToken:  accessToken,
		ExpiresIn:    int(time.Until(exp).Seconds()),
		RefreshToken: &newRefresh,
		Scope:        &scope,
	})
}

func (s *Server) exchangeClientCredentials(w http.ResponseWriter, r *http.Request, client db.OauthClient) {
	scope := r.PostFormValue("scope")
	if scope == "" {
		scope = strings.Join(client.Scopes, " ")
	}

	accessToken, jti, exp, err := s.issueJWT("", client.ID, scope)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token_issue_failed", err.Error())
		return
	}

	_, err = s.queries.CreateOAuthAccessToken(r.Context(), db.CreateOAuthAccessTokenParams{
		TokenHash: hashToken(accessToken),
		TokenJti:  jti,
		ClientID:  client.ID,
		UserID:    pgtype.UUID{},
		Scope:     scope,
		ExpiresAt: toPgTime(exp),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token_store_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, apigen.OAuth2TokenResponse{
		TokenType:   apigen.OAuth2TokenResponseTokenTypeBearer,
		AccessToken: accessToken,
		ExpiresIn:   int(time.Until(exp).Seconds()),
		Scope:       &scope,
	})
}

func (s *Server) issueJWT(userID, clientID, scope string) (token string, jti string, expiresAt time.Time, err error) {
	now := time.Now()
	expiresAt = now.Add(time.Duration(s.cfg.AccessTokenTTLSeconds) * time.Second)
	jti = uuid.NewString()

	claims := accessClaims{
		ClientID: clientID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.cfg.IssuerURL,
			Subject:   userID,
			Audience:  []string{clientID},
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        jti,
		},
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	jwtToken.Header["kid"] = s.cfg.SigningKeyID

	token, err = jwtToken.SignedString(s.privateKey)
	if err != nil {
		return "", "", time.Time{}, err
	}

	_ = scope
	return token, jti, expiresAt, nil
}

func (s *Server) parseAccessClaims(r *http.Request) (*accessClaims, error) {
	token := bearerToken(r)
	if token == "" {
		if cookie, err := r.Cookie("orivis_session"); err == nil {
			token = cookie.Value
		}
	}
	if token == "" {
		return nil, errors.New("missing token")
	}

	parsed, err := jwt.ParseWithClaims(token, &accessClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodRS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		return s.publicKey, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := parsed.Claims.(*accessClaims)
	if !ok || !parsed.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

func (s *Server) requirePrincipal(ctx context.Context, r *http.Request) (*principal, error) {
	claims, err := s.parseAccessClaims(r)
	if err != nil {
		return nil, err
	}

	if claims.Subject == "" {
		return nil, errors.New("token has no subject")
	}

	user, err := s.queries.GetUserByID(ctx, claims.Subject)
	if err != nil {
		return nil, err
	}

	groupsRows, _ := s.queries.ListUserGroups(ctx, user.ID)
	groups := make([]string, 0, len(groupsRows))
	for _, g := range groupsRows {
		groups = append(groups, g.ID)
	}

	isAdmin := claims.IsAdmin || claims.Subject == s.cfg.BootstrapAdminSub || strings.EqualFold(user.Email, s.cfg.BootstrapAdminEmail)

	return &principal{
		UserID:   user.ID,
		Email:    user.Email,
		Username: user.Username,
		Groups:   groups,
		IsAdmin:  isAdmin,
	}, nil
}

func (s *Server) requireAdmin(ctx context.Context, r *http.Request) (*principal, error) {
	p, err := s.requirePrincipal(ctx, r)
	if err != nil {
		return nil, err
	}
	if !p.IsAdmin {
		return nil, errors.New("admin required")
	}
	return p, nil
}

func (s *Server) authenticateAndIssueSession(ctx context.Context, userID, email, username string) (apigen.AuthResult, error) {
	groupsRows, _ := s.queries.ListUserGroups(ctx, userID)
	groups := make([]string, 0, len(groupsRows))
	for _, g := range groupsRows {
		groups = append(groups, g.ID)
	}

	isAdmin := userID == s.cfg.BootstrapAdminSub || strings.EqualFold(email, s.cfg.BootstrapAdminEmail)
	claims := accessClaims{
		Email:   email,
		Groups:  groups,
		IsAdmin: isAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.cfg.IssuerURL,
			Subject:   userID,
			Audience:  []string{"orivis"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(s.cfg.AccessTokenTTLSeconds) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.NewString(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = s.cfg.SigningKeyID
	accessToken, err := token.SignedString(s.privateKey)
	if err != nil {
		return apigen.AuthResult{}, err
	}

	rawRefresh, err := randomURLToken(48)
	if err != nil {
		return apigen.AuthResult{}, err
	}

	_, err = s.queries.CreateSession(ctx, db.CreateSessionParams{
		UserID:           userID,
		RefreshTokenHash: hashToken(rawRefresh),
		ExpiresAt:        toPgTime(time.Now().Add(time.Duration(s.cfg.RefreshTokenTTLHours) * time.Hour)),
	})
	if err != nil {
		return apigen.AuthResult{}, err
	}

	return apigen.AuthResult{
		Status: apigen.AuthResultStatusAuthenticated,
		Session: &apigen.Session{
			AccessToken:  accessToken,
			RefreshToken: rawRefresh,
			ExpiresIn:    s.cfg.AccessTokenTTLSeconds,
		},
	}, nil
}

func (s *Server) createChallenge(ctx context.Context, challengeType string, userID string, data map[string]any, expiresAt time.Time) (db.CreateAuthChallengeRow, error) {
	arg := db.CreateAuthChallengeParams{
		ChallengeType: challengeType,
		UserID:        pgUUID(userID),
		Data:          jsonBytes(data),
		ExpiresAt:     toPgTime(expiresAt),
	}
	return s.queries.CreateAuthChallenge(ctx, arg)
}

func (s *Server) fetchGoogleIdentity(ctx context.Context, accessToken string) (subject string, email string, err error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.googleapis.com/oauth2/v3/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("google userinfo failed: %s", string(body))
	}

	var payload struct {
		Sub   string `json:"sub"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", "", err
	}

	if payload.Sub == "" || payload.Email == "" {
		return "", "", errors.New("google response missing sub/email")
	}

	return payload.Sub, payload.Email, nil
}

func (s *Server) resolveUserFromGoogleIdentity(ctx context.Context, googleSub, googleEmail string) (userID, username string, err error) {
	id, err := s.queries.GetExternalIdentityByProviderSubject(ctx, db.GetExternalIdentityByProviderSubjectParams{Provider: "google", ProviderSubject: googleSub})
	if err == nil {
		user, err := s.queries.GetUserByID(ctx, id.UserID)
		if err != nil {
			return "", "", err
		}
		return user.ID, user.Username, nil
	}

	user, err := s.queries.GetUserByEmail(ctx, googleEmail)
	if err != nil {
		uname := strings.SplitN(googleEmail, "@", 2)[0]
		uname = fmt.Sprintf("%s-%s", uname, strings.ToLower(uuid.NewString()[:6]))
		created, createErr := s.queries.CreateUser(ctx, db.CreateUserParams{
			Email:        googleEmail,
			Username:     uname,
			PasswordHash: pgtype.Text{},
		})
		if createErr != nil {
			return "", "", createErr
		}
		user = db.GetUserByEmailRow{ID: created.ID, Email: created.Email, Username: created.Username, PasswordHash: created.PasswordHash}
	}

	_, _ = s.queries.CreateExternalIdentity(ctx, db.CreateExternalIdentityParams{
		UserID:          user.ID,
		Provider:        "google",
		ProviderSubject: googleSub,
		Email:           pgtype.Text{String: googleEmail, Valid: true},
		Metadata:        jsonBytes(map[string]any{"provider": "google"}),
	})

	_, _ = s.queries.CreateUserAuthMethod(ctx, db.CreateUserAuthMethodParams{
		UserID:          user.ID,
		MethodType:      string(apigen.AuthMethodTypeOauthGoogle),
		ProviderSubject: pgtype.Text{String: googleSub, Valid: true},
		SecretRef:       pgtype.Text{},
		Metadata:        jsonBytes(map[string]any{"provider": "google"}),
	})

	return user.ID, user.Username, nil
}

func (s *Server) audit(ctx context.Context, actorType, actorID, action, targetType, targetID string, metadata map[string]any) {
	_, _ = s.queries.CreateAuditEvent(ctx, db.CreateAuditEventParams{
		ActorType:  actorType,
		ActorID:    actorID,
		Action:     action,
		TargetType: pgtype.Text{String: targetType, Valid: targetType != ""},
		TargetID:   pgtype.Text{String: targetID, Valid: targetID != ""},
		Metadata:   jsonBytes(metadata),
	})
}

func (s *Server) setSessionCookie(w http.ResponseWriter, accessToken string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "orivis_session",
		Value:    accessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   strings.HasPrefix(s.cfg.IssuerURL, "https://"),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   s.cfg.AccessTokenTTLSeconds,
	})
}

func (s *Server) ensureDefaultOAuthClient(ctx context.Context) error {
	const defaultClientID = "orivis-dashboard"
	if _, err := s.queries.GetOAuthClientByID(ctx, defaultClientID); err == nil {
		return nil
	}

	secretHash, err := hashPassword("orivis-dashboard-secret")
	if err != nil {
		return err
	}

	_, err = s.queries.CreateOAuthClient(ctx, db.CreateOAuthClientParams{
		ID:               defaultClientID,
		Name:             "Orivis Dashboard",
		RedirectUris:     []string{s.cfg.FrontendURL + "/oauth/callback"},
		Scopes:           []string{"openid", "profile", "email"},
		Confidential:     true,
		ClientSecretHash: pgtype.Text{String: secretHash, Valid: true},
		RequirePkce:      true,
	})
	if err != nil && !isUniqueViolation(err) {
		return err
	}

	return nil
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	writeJSONWithStatus(w, status, data)
}

func writeJSONWithStatus(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSONWithStatus(w, status, apigen.Error{Code: code, Message: message})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return false
	}
	return true
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func randomURLToken(size int) (string, error) {
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func toPgTime(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t.UTC(), Valid: true}
}

func pgUUID(id string) pgtype.UUID {
	if id == "" {
		return pgtype.UUID{}
	}
	var out pgtype.UUID
	_ = out.Scan(id)
	return out
}

func bearerToken(r *http.Request) string {
	value := strings.TrimSpace(r.Header.Get("Authorization"))
	if value == "" {
		return ""
	}
	parts := strings.SplitN(value, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, 3, 64*1024, 1, 32)
	return fmt.Sprintf("argon2id$v=19$m=65536,t=3,p=1$%s$%s",
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

func verifyPassword(password, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 5 {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return false
	}
	decodedHash, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}

	computed := argon2.IDKey([]byte(password), salt, 3, 64*1024, 1, uint32(len(decodedHash)))
	return subtle.ConstantTimeCompare(decodedHash, computed) == 1
}

func pkceS256(verifier string) string {
	s := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(s[:])
}

func contains(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}

func jsonBytes(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func ptr[T any](v T) *T {
	return &v
}

func stringPtr(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

func isUniqueViolation(err error) bool {
	var pgConnErr *pgconn.PgError
	if errors.As(err, &pgConnErr) {
		return pgConnErr.Code == "23505"
	}
	return false
}
