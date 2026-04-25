import { useCallback, useEffect, useMemo, useState } from 'react'
import type { FormEvent, ReactNode } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  BrowserRouter,
  Link,
  Navigate,
  Outlet,
  Route,
  Routes,
  useLocation,
  useNavigate
} from 'react-router-dom'
import {
  completeExternalProviderLogin,
  getForwardAuthGroupPlatformOverride,
  getForwardAuthPlatformPolicy,
  getForwardAuthUserPlatformOverride,
  getOAuth2GroupPlatformOverride,
  getOAuth2PlatformPolicy,
  getOAuth2UserPlatformOverride,
  linkCurrentUserAuthMethod,
  listAuditEvents,
  putForwardAuthGroupPlatformOverride,
  putForwardAuthPlatformPolicy,
  putForwardAuthUserPlatformOverride,
  putOAuth2GroupPlatformOverride,
  putOAuth2PlatformPolicy,
  putOAuth2UserPlatformOverride,
  registerWithPassword,
  startExternalProviderLogin,
  unlinkCurrentUserAuthMethod,
  useGetCurrentUser,
  useListCurrentUserAuthMethods,
  useLoginWithPassword,
  useVerifyTotpChallenge
} from './api/generated/orivis'
import { PolicyDecision, PolicyMode } from './api/generated/model'
import { clearAccessToken, getAccessToken, setAccessToken } from './auth/session'

type AuthChallengeState = {
  challengeId: string
  requiredChallenge: 'totp'
}

type PolicySurface = 'oauth2' | 'forward_auth'
type SubjectType = 'user' | 'group'

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<GuestOnly />}>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/register" element={<RegisterPage />} />
          <Route path="/oauth/callback" element={<ExternalProviderCallbackPage />} />
        </Route>

        <Route element={<RequireAuth />}>
          <Route path="/dashboard" element={<DashboardLayout />}>
            <Route index element={<DashboardHomePage />} />
            <Route path="admin" element={<AdminPoliciesPage />} />
            <Route path="audit" element={<AuditEventsPage />} />
          </Route>
        </Route>

        <Route path="*" element={<Navigate to="/dashboard" replace />} />
      </Routes>
    </BrowserRouter>
  )
}

function GuestOnly() {
  const me = useGetCurrentUser({ query: { retry: false } })

  if (me.isLoading) {
    return <PageStatus label="Checking existing session..." />
  }

  if (me.isSuccess) {
    return <Navigate to="/dashboard" replace />
  }

  return <Outlet />
}

function RequireAuth() {
  const me = useGetCurrentUser({ query: { retry: false } })
  const location = useLocation()

  if (me.isLoading) {
    return <PageStatus label="Loading session..." />
  }

  if (me.isError) {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />
  }

  return <Outlet />
}

function LoginPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [identifier, setIdentifier] = useState('')
  const [password, setPassword] = useState('')
  const [totpCode, setTotpCode] = useState('')
  const [challenge, setChallenge] = useState<AuthChallengeState | null>(null)
  const [generalError, setGeneralError] = useState('')
  const [showPasskeyHint, setShowPasskeyHint] = useState(false)

  const loginMutation = useLoginWithPassword()
  const verifyTotpMutation = useVerifyTotpChallenge()
  const googleStartMutation = useMutation({
    mutationFn: () =>
      startExternalProviderLogin('google', {
        redirectUri: `${window.location.origin}/oauth/callback`,
        intent: 'login'
      })
  })

  const onAuthenticated = useCallback(
    (token: string) => {
      setAccessToken(token)
      queryClient.invalidateQueries({ queryKey: ['/v1/me'] })
      queryClient.invalidateQueries({ queryKey: ['/v1/me/methods'] })
      navigate('/dashboard', { replace: true })
    },
    [navigate, queryClient]
  )

  const submitPasswordLogin = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setGeneralError('')

    try {
      const result = await loginMutation.mutateAsync({ data: { identifier, password } })

      if (result.status === 'challenge_required' && result.challengeId && result.requiredChallenge === 'totp') {
        setChallenge({ challengeId: result.challengeId, requiredChallenge: 'totp' })
        return
      }

      if (result.session?.accessToken) {
        onAuthenticated(result.session.accessToken)
        return
      }

      setGeneralError('Login response did not contain a session token.')
    } catch (error) {
      setGeneralError(getErrorMessage(error))
    }
  }

  const submitTotp = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!challenge) {
      return
    }

    setGeneralError('')

    try {
      const result = await verifyTotpMutation.mutateAsync({
        data: {
          challengeId: challenge.challengeId,
          code: totpCode
        }
      })

      if (result.session?.accessToken) {
        onAuthenticated(result.session.accessToken)
        return
      }

      setGeneralError('TOTP verification did not return a session token.')
    } catch (error) {
      setGeneralError(getErrorMessage(error))
    }
  }

  const startGoogleLogin = async () => {
    setGeneralError('')

    try {
      const result = await googleStartMutation.mutateAsync()
      window.location.assign(result.authorizationUrl)
    } catch (error) {
      setGeneralError(getErrorMessage(error))
    }
  }

  return (
    <AuthShell
      title="Welcome back"
      subtitle="Sign in to manage OAuth2, forward-auth policy, and account security methods."
      footer={
        <p className="text-sm text-slate-600">
          No account yet?{' '}
          <Link to="/register" className="font-semibold text-pine hover:text-ink">
            Create one
          </Link>
        </p>
      }
    >
      {challenge ? (
        <form className="space-y-4" onSubmit={submitTotp}>
          <div>
            <label className="mb-1 block text-sm font-medium text-slate-700" htmlFor="totpCode">
              TOTP code
            </label>
            <input
              id="totpCode"
              autoComplete="one-time-code"
              className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm focus:border-pine focus:outline-none"
              maxLength={6}
              minLength={6}
              pattern="[0-9]{6}"
              value={totpCode}
              onChange={(event) => setTotpCode(event.target.value)}
              required
            />
          </div>

          <button
            className="w-full rounded-lg bg-pine px-4 py-2 text-sm font-semibold text-white transition hover:bg-ink disabled:cursor-not-allowed disabled:opacity-50"
            disabled={verifyTotpMutation.isPending}
            type="submit"
          >
            {verifyTotpMutation.isPending ? 'Verifying...' : 'Verify challenge'}
          </button>

          <button
            className="w-full rounded-lg border border-slate-300 px-4 py-2 text-sm font-semibold text-slate-700 transition hover:bg-slate-50"
            onClick={() => {
              setChallenge(null)
              setTotpCode('')
              setGeneralError('')
            }}
            type="button"
          >
            Back to password login
          </button>
        </form>
      ) : (
        <>
          <form className="space-y-4" onSubmit={submitPasswordLogin}>
            <div>
              <label className="mb-1 block text-sm font-medium text-slate-700" htmlFor="identifier">
                Email or username
              </label>
              <input
                id="identifier"
                autoComplete="username"
                className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm focus:border-pine focus:outline-none"
                value={identifier}
                onChange={(event) => setIdentifier(event.target.value)}
                required
              />
            </div>

            <div>
              <label className="mb-1 block text-sm font-medium text-slate-700" htmlFor="password">
                Password
              </label>
              <input
                id="password"
                autoComplete="current-password"
                className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm focus:border-pine focus:outline-none"
                type="password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                required
              />
            </div>

            <button
              className="w-full rounded-lg bg-pine px-4 py-2 text-sm font-semibold text-white transition hover:bg-ink disabled:cursor-not-allowed disabled:opacity-50"
              disabled={loginMutation.isPending}
              type="submit"
            >
              {loginMutation.isPending ? 'Signing in...' : 'Sign in'}
            </button>
          </form>

          <div className="mt-4 grid gap-2">
            <button
              className="rounded-lg border border-glacier/80 bg-white px-4 py-2 text-sm font-semibold text-ink transition hover:border-pine"
              disabled={googleStartMutation.isPending}
              onClick={startGoogleLogin}
              type="button"
            >
              {googleStartMutation.isPending ? 'Redirecting...' : 'Continue with Google'}
            </button>

            <button
              className="rounded-lg border border-slate-200 px-4 py-2 text-sm text-slate-700 transition hover:bg-slate-50"
              onClick={() => setShowPasskeyHint((value) => !value)}
              type="button"
            >
              Passkey sign-in help
            </button>
          </div>

          {showPasskeyHint ? (
            <p className="mt-3 text-xs text-slate-600">
              Passkey verification endpoint is available, but this UI currently uses password + TOTP and Google OAuth for complete sign-in.
            </p>
          ) : null}
        </>
      )}

      {generalError ? <ErrorNotice message={generalError} /> : null}
    </AuthShell>
  )
}

function RegisterPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [email, setEmail] = useState('')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [errorMessage, setErrorMessage] = useState('')

  const registerMutation = useMutation({
    mutationFn: (data: { email: string; username: string; password: string }) => registerWithPassword(data)
  })

  const submitRegistration = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setErrorMessage('')

    try {
      const result = await registerMutation.mutateAsync({ email, username, password })

      if (!result.session?.accessToken) {
        setErrorMessage('Registration completed, but no session token was returned.')
        return
      }

      setAccessToken(result.session.accessToken)
      queryClient.invalidateQueries({ queryKey: ['/v1/me'] })
      queryClient.invalidateQueries({ queryKey: ['/v1/me/methods'] })
      navigate('/dashboard', { replace: true })
    } catch (error) {
      setErrorMessage(getErrorMessage(error))
    }
  }

  return (
    <AuthShell
      title="Create your Orivis account"
      subtitle="Get started with local credentials, then add passkey, TOTP, or Google from the dashboard."
      footer={
        <p className="text-sm text-slate-600">
          Already signed up?{' '}
          <Link to="/login" className="font-semibold text-pine hover:text-ink">
            Sign in
          </Link>
        </p>
      }
    >
      <form className="space-y-4" onSubmit={submitRegistration}>
        <div>
          <label className="mb-1 block text-sm font-medium text-slate-700" htmlFor="email">
            Email
          </label>
          <input
            id="email"
            autoComplete="email"
            className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm focus:border-pine focus:outline-none"
            type="email"
            value={email}
            onChange={(event) => setEmail(event.target.value)}
            required
          />
        </div>

        <div>
          <label className="mb-1 block text-sm font-medium text-slate-700" htmlFor="username">
            Username
          </label>
          <input
            id="username"
            autoComplete="username"
            className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm focus:border-pine focus:outline-none"
            minLength={3}
            value={username}
            onChange={(event) => setUsername(event.target.value)}
            required
          />
        </div>

        <div>
          <label className="mb-1 block text-sm font-medium text-slate-700" htmlFor="newPassword">
            Password
          </label>
          <input
            id="newPassword"
            autoComplete="new-password"
            className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm focus:border-pine focus:outline-none"
            minLength={12}
            type="password"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            required
          />
        </div>

        <button
          className="w-full rounded-lg bg-pine px-4 py-2 text-sm font-semibold text-white transition hover:bg-ink disabled:cursor-not-allowed disabled:opacity-50"
          disabled={registerMutation.isPending}
          type="submit"
        >
          {registerMutation.isPending ? 'Creating account...' : 'Create account'}
        </button>
      </form>

      {errorMessage ? <ErrorNotice message={errorMessage} /> : null}
    </AuthShell>
  )
}

function ExternalProviderCallbackPage() {
  const navigate = useNavigate()
  const location = useLocation()
  const queryClient = useQueryClient()
  const [errorMessage, setErrorMessage] = useState('')

  const callback = useMutation({
    mutationFn: (payload: { provider: 'google'; data: { code: string; state: string } }) =>
      completeExternalProviderLogin(payload.provider, payload.data)
  })

  const searchParams = useMemo(() => new URLSearchParams(location.search), [location.search])

  const code = searchParams.get('code') ?? ''
  const state = searchParams.get('state') ?? ''

  useEffect(() => {
    if (!code || !state || callback.isPending || callback.isSuccess) {
      return
    }

    callback
      .mutateAsync({ provider: 'google', data: { code, state } })
      .then((result) => {
        if (!result.session?.accessToken) {
          setErrorMessage('Provider callback succeeded but did not create a session token.')
          return
        }

        setAccessToken(result.session.accessToken)
        queryClient.invalidateQueries({ queryKey: ['/v1/me'] })
        queryClient.invalidateQueries({ queryKey: ['/v1/me/methods'] })
        navigate('/dashboard', { replace: true })
      })
      .catch((error: unknown) => {
        setErrorMessage(getErrorMessage(error))
      })
  }, [callback, code, navigate, queryClient, state])

  if (!code || !state) {
    return (
      <AuthShell
        title="Provider callback is incomplete"
        subtitle="Missing authorization code or state value from Google callback."
      >
        <Link className="text-sm font-semibold text-pine hover:text-ink" to="/login">
          Back to login
        </Link>
      </AuthShell>
    )
  }

  return (
    <AuthShell title="Completing Google sign-in" subtitle="Finishing provider callback with Orivis.">
      {callback.isPending ? <p className="text-sm text-slate-700">Please wait while we verify your provider identity...</p> : null}
      {errorMessage ? (
        <div className="space-y-3">
          <ErrorNotice message={errorMessage} />
          <Link className="text-sm font-semibold text-pine hover:text-ink" to="/login">
            Return to login
          </Link>
        </div>
      ) : null}
    </AuthShell>
  )
}

function DashboardLayout() {
  const navigate = useNavigate()
  const me = useGetCurrentUser({ query: { retry: false } })

  if (me.isLoading || !me.data) {
    return <PageStatus label="Loading dashboard..." />
  }

  const clearBearerToken = () => {
    clearAccessToken()
    navigate('/dashboard', { replace: true })
  }

  const hasToken = Boolean(getAccessToken())

  return (
    <main className="min-h-screen bg-gradient-to-b from-mist to-white text-ink">
      <div className="mx-auto grid max-w-7xl gap-6 px-4 py-6 md:grid-cols-[220px_1fr] md:px-6">
        <aside className="rounded-2xl border border-glacier/50 bg-white p-4 shadow-sm">
          <p className="text-xs uppercase tracking-[0.16em] text-pine">Orivis Control</p>
          <p className="mt-2 text-sm font-medium text-slate-700">{me.data.email}</p>

          <nav className="mt-6 grid gap-2 text-sm">
            <Link className="rounded-md px-3 py-2 transition hover:bg-mist" to="/dashboard">
              Overview
            </Link>
            <Link className="rounded-md px-3 py-2 transition hover:bg-mist" to="/dashboard/admin">
              Admin policies
            </Link>
            <Link className="rounded-md px-3 py-2 transition hover:bg-mist" to="/dashboard/audit">
              Audit events
            </Link>
          </nav>

          <div className="mt-6 rounded-lg border border-slate-200 bg-slate-50 p-3">
            <p className="text-xs font-medium uppercase tracking-wide text-slate-500">Admin bearer token</p>
            <p className="mt-1 text-xs text-slate-600">{hasToken ? 'Attached to API requests' : 'Not stored in this browser session'}</p>
            <button
              className="mt-2 w-full rounded-md border border-slate-300 px-3 py-2 text-xs font-semibold text-slate-700 transition hover:bg-white"
              onClick={clearBearerToken}
              type="button"
            >
              Clear stored token
            </button>
          </div>
        </aside>

        <section className="space-y-6">
          <Outlet />
        </section>
      </div>
    </main>
  )
}

function DashboardHomePage() {
  const queryClient = useQueryClient()
  const methodsQuery = useListCurrentUserAuthMethods()
  const userQuery = useGetCurrentUser({ query: { retry: false } })
  const [passkeyCredentialId, setPasskeyCredentialId] = useState('')
  const [googleSubject, setGoogleSubject] = useState('')
  const [linkMessage, setLinkMessage] = useState('')
  const [linkError, setLinkError] = useState('')

  const linkMutation = useMutation({ mutationFn: (data: { type: 'totp' | 'passkey' | 'oauth_google'; payload: Record<string, unknown> }) => linkCurrentUserAuthMethod(data) })
  const unlinkMutation = useMutation({ mutationFn: ({ methodId }: { methodId: string }) => unlinkCurrentUserAuthMethod(methodId) })

  const refreshMethods = () => {
    queryClient.invalidateQueries({ queryKey: ['/v1/me/methods'] })
  }

  const linkTotp = async () => {
    setLinkMessage('')
    setLinkError('')

    try {
      const created = await linkMutation.mutateAsync({ type: 'totp', payload: {} })
      const metadata = created.metadata ?? {}
      const secret = String((metadata as Record<string, unknown>).secret ?? '')
      const otpauthUrl = String((metadata as Record<string, unknown>).otpauthUrl ?? '')
      setLinkMessage(`TOTP linked. Secret: ${secret || 'n/a'} | otpauth URL: ${otpauthUrl || 'n/a'}`)
      refreshMethods()
    } catch (error) {
      setLinkError(getErrorMessage(error))
    }
  }

  const linkPasskey = async () => {
    setLinkMessage('')
    setLinkError('')

    if (!passkeyCredentialId) {
      setLinkError('Credential ID is required to link a passkey in the current API contract.')
      return
    }

    try {
      await linkMutation.mutateAsync({ type: 'passkey', payload: { credentialId: passkeyCredentialId } })
      setLinkMessage('Passkey method linked.')
      setPasskeyCredentialId('')
      refreshMethods()
    } catch (error) {
      setLinkError(getErrorMessage(error))
    }
  }

  const linkGoogleMethod = async () => {
    setLinkMessage('')
    setLinkError('')

    if (!googleSubject) {
      setLinkError('Provider subject is required for oauth_google link in the current backend implementation.')
      return
    }

    try {
      await linkMutation.mutateAsync({ type: 'oauth_google', payload: { providerSubject: googleSubject } })
      setLinkMessage('Google auth method linked.')
      setGoogleSubject('')
      refreshMethods()
    } catch (error) {
      setLinkError(getErrorMessage(error))
    }
  }

  const unlinkMethod = async (methodId: string) => {
    setLinkMessage('')
    setLinkError('')

    try {
      await unlinkMutation.mutateAsync({ methodId })
      refreshMethods()
    } catch (error) {
      setLinkError(getErrorMessage(error))
    }
  }

  return (
    <>
      <article className="rounded-2xl border border-glacier/40 bg-white p-5 shadow-sm">
        <p className="text-xs uppercase tracking-[0.16em] text-pine">Profile</p>
        {userQuery.data ? (
          <div className="mt-3 grid gap-1 text-sm text-slate-700">
            <p>
              <span className="font-semibold text-ink">ID:</span> {userQuery.data.id}
            </p>
            <p>
              <span className="font-semibold text-ink">Email:</span> {userQuery.data.email}
            </p>
            <p>
              <span className="font-semibold text-ink">Username:</span> {userQuery.data.username}
            </p>
            <p>
              <span className="font-semibold text-ink">Groups:</span> {userQuery.data.groups.join(', ') || 'none'}
            </p>
          </div>
        ) : (
          <p className="mt-2 text-sm text-slate-600">Unable to load profile.</p>
        )}
      </article>

      <article className="rounded-2xl border border-glacier/40 bg-white p-5 shadow-sm">
        <p className="text-xs uppercase tracking-[0.16em] text-pine">Authentication methods</p>

        {methodsQuery.isLoading ? <p className="mt-3 text-sm text-slate-600">Loading methods...</p> : null}

        {methodsQuery.isError ? <ErrorNotice message={getErrorMessage(methodsQuery.error)} /> : null}

        {methodsQuery.data?.items?.length ? (
          <div className="mt-3 overflow-x-auto">
            <table className="w-full min-w-[640px] border-separate border-spacing-y-2 text-sm">
              <thead>
                <tr className="text-left text-xs uppercase tracking-wide text-slate-500">
                  <th className="px-2">Type</th>
                  <th className="px-2">Created</th>
                  <th className="px-2">Metadata</th>
                  <th className="px-2">Action</th>
                </tr>
              </thead>
              <tbody>
                {methodsQuery.data.items.map((method) => (
                  <tr className="rounded-lg bg-slate-50" key={method.id}>
                    <td className="px-2 py-2 font-medium text-ink">{method.type}</td>
                    <td className="px-2 py-2 text-slate-700">{new Date(method.createdAt).toLocaleString()}</td>
                    <td className="max-w-[360px] truncate px-2 py-2 text-xs text-slate-600">{JSON.stringify(method.metadata ?? {})}</td>
                    <td className="px-2 py-2">
                      <button
                        className="rounded-md border border-slate-300 px-3 py-1 text-xs font-semibold text-slate-700 transition hover:bg-white disabled:opacity-50"
                        disabled={unlinkMutation.isPending}
                        onClick={() => unlinkMethod(method.id)}
                        type="button"
                      >
                        Unlink
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : null}

        {!methodsQuery.data?.items?.length && !methodsQuery.isLoading ? (
          <p className="mt-3 text-sm text-slate-600">No linked auth methods found.</p>
        ) : null}

        <div className="mt-6 grid gap-4 md:grid-cols-3">
          <div className="rounded-xl border border-slate-200 p-3">
            <p className="text-sm font-semibold text-ink">Link TOTP</p>
            <p className="mt-1 text-xs text-slate-600">Generates secret and otpauth URL immediately.</p>
            <button
              className="mt-3 rounded-md bg-pine px-3 py-2 text-xs font-semibold text-white transition hover:bg-ink disabled:opacity-50"
              disabled={linkMutation.isPending}
              onClick={linkTotp}
              type="button"
            >
              Link TOTP
            </button>
          </div>

          <div className="rounded-xl border border-slate-200 p-3">
            <p className="text-sm font-semibold text-ink">Link Passkey</p>
            <input
              className="mt-2 w-full rounded-md border border-slate-300 px-2 py-1 text-xs focus:border-pine focus:outline-none"
              onChange={(event) => setPasskeyCredentialId(event.target.value)}
              placeholder="credentialId"
              value={passkeyCredentialId}
            />
            <button
              className="mt-3 rounded-md bg-pine px-3 py-2 text-xs font-semibold text-white transition hover:bg-ink disabled:opacity-50"
              disabled={linkMutation.isPending}
              onClick={linkPasskey}
              type="button"
            >
              Link passkey
            </button>
          </div>

          <div className="rounded-xl border border-slate-200 p-3">
            <p className="text-sm font-semibold text-ink">Link Google Method</p>
            <input
              className="mt-2 w-full rounded-md border border-slate-300 px-2 py-1 text-xs focus:border-pine focus:outline-none"
              onChange={(event) => setGoogleSubject(event.target.value)}
              placeholder="providerSubject"
              value={googleSubject}
            />
            <button
              className="mt-3 rounded-md bg-pine px-3 py-2 text-xs font-semibold text-white transition hover:bg-ink disabled:opacity-50"
              disabled={linkMutation.isPending}
              onClick={linkGoogleMethod}
              type="button"
            >
              Link Google
            </button>
          </div>
        </div>

        {linkMessage ? <SuccessNotice message={linkMessage} /> : null}
        {linkError ? <ErrorNotice message={linkError} /> : null}
      </article>
    </>
  )
}

function AdminPoliciesPage() {
  return (
    <div className="grid gap-6">
      <PlatformPolicyEditor surface="oauth2" title="OAuth2 platform policy" />
      <PlatformPolicyEditor surface="forward_auth" title="Forward-auth platform policy" />
      <SubjectOverrideEditor surface="oauth2" subjectType="user" title="OAuth2 user override" />
      <SubjectOverrideEditor surface="oauth2" subjectType="group" title="OAuth2 group override" />
      <SubjectOverrideEditor surface="forward_auth" subjectType="user" title="Forward-auth user override" />
      <SubjectOverrideEditor surface="forward_auth" subjectType="group" title="Forward-auth group override" />
    </div>
  )
}

function PlatformPolicyEditor({ surface, title }: { surface: PolicySurface; title: string }) {
  const [platformId, setPlatformId] = useState('')
  const [mode, setMode] = useState<keyof typeof PolicyMode>('allow_any')
  const [entriesText, setEntriesText] = useState('')
  const [statusText, setStatusText] = useState('')
  const [errorText, setErrorText] = useState('')

  const policyQuery = useQuery({
    queryKey: ['admin', 'platform-policy', surface, platformId],
    enabled: Boolean(platformId),
    queryFn: () =>
      surface === 'oauth2' ? getOAuth2PlatformPolicy(platformId) : getForwardAuthPlatformPolicy(platformId),
    retry: false
  })

  const saveMutation = useMutation({
    mutationFn: (payload: { platformId: string; mode: keyof typeof PolicyMode; entries: string[] }) => {
      const body = {
        platformId: payload.platformId,
        mode: PolicyMode[payload.mode],
        entries: payload.entries,
        updatedAt: new Date().toISOString()
      }

      return surface === 'oauth2'
        ? putOAuth2PlatformPolicy(payload.platformId, body)
        : putForwardAuthPlatformPolicy(payload.platformId, body)
    }
  })

  const loadPolicy = () => {
    setStatusText('')
    setErrorText('')
    policyQuery
      .refetch()
      .then((result) => {
        if (!result.data) {
          return
        }

        setMode(result.data.mode)
        setEntriesText(result.data.entries.join('\n'))
        setStatusText('Policy loaded from API.')
      })
      .catch((error: unknown) => {
        setErrorText(getErrorMessage(error))
      })
  }

  const savePolicy = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setStatusText('')
    setErrorText('')

    const entries = entriesText
      .split('\n')
      .map((value) => value.trim())
      .filter(Boolean)

    try {
      await saveMutation.mutateAsync({ platformId, mode, entries })
      setStatusText('Policy saved.')
    } catch (error) {
      setErrorText(getErrorMessage(error))
    }
  }

  return (
    <article className="rounded-2xl border border-glacier/40 bg-white p-5 shadow-sm">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-sm font-semibold text-ink">{title}</p>
        <button
          className="rounded-md border border-slate-300 px-3 py-1 text-xs font-semibold text-slate-700 transition hover:bg-slate-50"
          disabled={!platformId || policyQuery.isFetching}
          onClick={loadPolicy}
          type="button"
        >
          {policyQuery.isFetching ? 'Loading...' : 'Load current policy'}
        </button>
      </div>

      <form className="mt-4 grid gap-3" onSubmit={savePolicy}>
        <input
          className="rounded-lg border border-slate-300 px-3 py-2 text-sm focus:border-pine focus:outline-none"
          onChange={(event) => setPlatformId(event.target.value)}
          placeholder="platformId"
          value={platformId}
          required
        />

        <select
          className="rounded-lg border border-slate-300 px-3 py-2 text-sm focus:border-pine focus:outline-none"
          onChange={(event) => setMode(event.target.value as keyof typeof PolicyMode)}
          value={mode}
        >
          <option value="allow_any">allow_any</option>
          <option value="allowlist">allowlist</option>
          <option value="denylist">denylist</option>
        </select>

        <textarea
          className="min-h-24 rounded-lg border border-slate-300 px-3 py-2 text-sm focus:border-pine focus:outline-none"
          onChange={(event) => setEntriesText(event.target.value)}
          placeholder="Entries (one per line)"
          value={entriesText}
        />

        <button
          className="rounded-lg bg-pine px-4 py-2 text-sm font-semibold text-white transition hover:bg-ink disabled:opacity-50"
          disabled={saveMutation.isPending || !platformId}
          type="submit"
        >
          {saveMutation.isPending ? 'Saving...' : 'Save policy'}
        </button>
      </form>

      {statusText ? <SuccessNotice message={statusText} /> : null}
      {errorText ? <ErrorNotice message={errorText} /> : null}
      {policyQuery.error ? <ErrorNotice message={getErrorMessage(policyQuery.error)} /> : null}
    </article>
  )
}

function SubjectOverrideEditor({
  surface,
  subjectType,
  title
}: {
  surface: PolicySurface
  subjectType: SubjectType
  title: string
}) {
  const [subjectId, setSubjectId] = useState('')
  const [platformId, setPlatformId] = useState('')
  const [decision, setDecision] = useState<keyof typeof PolicyDecision>('inherit')
  const [reason, setReason] = useState('')
  const [statusText, setStatusText] = useState('')
  const [errorText, setErrorText] = useState('')

  const overrideQuery = useQuery({
    queryKey: ['admin', 'override', surface, subjectType, subjectId, platformId],
    enabled: Boolean(subjectId && platformId),
    queryFn: () => {
      if (surface === 'oauth2' && subjectType === 'user') {
        return getOAuth2UserPlatformOverride(subjectId, platformId)
      }
      if (surface === 'oauth2' && subjectType === 'group') {
        return getOAuth2GroupPlatformOverride(subjectId, platformId)
      }
      if (surface === 'forward_auth' && subjectType === 'user') {
        return getForwardAuthUserPlatformOverride(subjectId, platformId)
      }
      return getForwardAuthGroupPlatformOverride(subjectId, platformId)
    },
    retry: false
  })

  const saveMutation = useMutation({
    mutationFn: (payload: {
      subjectId: string
      platformId: string
      decision: keyof typeof PolicyDecision
      reason: string
    }) => {
      const body = {
        subjectId: payload.subjectId,
        subjectType,
        platformId: payload.platformId,
        decision: PolicyDecision[payload.decision],
        reason: payload.reason || undefined
      }

      if (surface === 'oauth2' && subjectType === 'user') {
        return putOAuth2UserPlatformOverride(payload.subjectId, payload.platformId, body)
      }
      if (surface === 'oauth2' && subjectType === 'group') {
        return putOAuth2GroupPlatformOverride(payload.subjectId, payload.platformId, body)
      }
      if (surface === 'forward_auth' && subjectType === 'user') {
        return putForwardAuthUserPlatformOverride(payload.subjectId, payload.platformId, body)
      }
      return putForwardAuthGroupPlatformOverride(payload.subjectId, payload.platformId, body)
    }
  })

  const loadOverride = () => {
    setStatusText('')
    setErrorText('')
    overrideQuery
      .refetch()
      .then((result) => {
        if (!result.data) {
          return
        }

        setDecision(result.data.decision)
        setReason(result.data.reason ?? '')
        setStatusText('Override loaded from API.')
      })
      .catch((error: unknown) => {
        setErrorText(getErrorMessage(error))
      })
  }

  const saveOverride = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setStatusText('')
    setErrorText('')

    try {
      await saveMutation.mutateAsync({ subjectId, platformId, decision, reason })
      setStatusText('Override saved.')
    } catch (error) {
      setErrorText(getErrorMessage(error))
    }
  }

  return (
    <article className="rounded-2xl border border-glacier/40 bg-white p-5 shadow-sm">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-sm font-semibold text-ink">{title}</p>
        <button
          className="rounded-md border border-slate-300 px-3 py-1 text-xs font-semibold text-slate-700 transition hover:bg-slate-50"
          disabled={!subjectId || !platformId || overrideQuery.isFetching}
          onClick={loadOverride}
          type="button"
        >
          {overrideQuery.isFetching ? 'Loading...' : 'Load current override'}
        </button>
      </div>

      <form className="mt-4 grid gap-3 md:grid-cols-2" onSubmit={saveOverride}>
        <input
          className="rounded-lg border border-slate-300 px-3 py-2 text-sm focus:border-pine focus:outline-none"
          onChange={(event) => setSubjectId(event.target.value)}
          placeholder={subjectType === 'user' ? 'userId' : 'groupId'}
          value={subjectId}
          required
        />
        <input
          className="rounded-lg border border-slate-300 px-3 py-2 text-sm focus:border-pine focus:outline-none"
          onChange={(event) => setPlatformId(event.target.value)}
          placeholder="platformId"
          value={platformId}
          required
        />

        <select
          className="rounded-lg border border-slate-300 px-3 py-2 text-sm focus:border-pine focus:outline-none"
          onChange={(event) => setDecision(event.target.value as keyof typeof PolicyDecision)}
          value={decision}
        >
          <option value="inherit">inherit</option>
          <option value="allow">allow</option>
          <option value="deny">deny</option>
        </select>

        <input
          className="rounded-lg border border-slate-300 px-3 py-2 text-sm focus:border-pine focus:outline-none"
          onChange={(event) => setReason(event.target.value)}
          placeholder="reason (optional)"
          value={reason}
        />

        <button
          className="rounded-lg bg-pine px-4 py-2 text-sm font-semibold text-white transition hover:bg-ink disabled:opacity-50 md:col-span-2"
          disabled={saveMutation.isPending || !subjectId || !platformId}
          type="submit"
        >
          {saveMutation.isPending ? 'Saving...' : 'Save override'}
        </button>
      </form>

      {statusText ? <SuccessNotice message={statusText} /> : null}
      {errorText ? <ErrorNotice message={errorText} /> : null}
      {overrideQuery.error ? <ErrorNotice message={getErrorMessage(overrideQuery.error)} /> : null}
    </article>
  )
}

function AuditEventsPage() {
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(25)

  const auditQuery = useQuery({
    queryKey: ['admin', 'audit-events', page, pageSize],
    queryFn: () => listAuditEvents({ page, pageSize }),
    retry: false
  })

  const total = auditQuery.data?.total ?? 0
  const maxPage = Math.max(1, Math.ceil(total / pageSize))

  return (
    <article className="rounded-2xl border border-glacier/40 bg-white p-5 shadow-sm">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-sm font-semibold text-ink">Audit events</p>
        <div className="flex items-center gap-2">
          <label className="text-xs text-slate-600" htmlFor="pageSize">
            Page size
          </label>
          <select
            className="rounded-md border border-slate-300 px-2 py-1 text-xs"
            id="pageSize"
            onChange={(event) => {
              setPageSize(Number(event.target.value))
              setPage(1)
            }}
            value={pageSize}
          >
            <option value={10}>10</option>
            <option value={25}>25</option>
            <option value={50}>50</option>
          </select>
        </div>
      </div>

      {auditQuery.isLoading ? <p className="mt-3 text-sm text-slate-600">Loading events...</p> : null}
      {auditQuery.error ? <ErrorNotice message={getErrorMessage(auditQuery.error)} /> : null}

      {auditQuery.data?.items?.length ? (
        <div className="mt-4 overflow-x-auto">
          <table className="w-full min-w-[760px] text-sm">
            <thead>
              <tr className="border-b border-slate-200 text-left text-xs uppercase tracking-wide text-slate-500">
                <th className="py-2 pr-3">When</th>
                <th className="py-2 pr-3">Actor</th>
                <th className="py-2 pr-3">Action</th>
                <th className="py-2 pr-3">Target</th>
                <th className="py-2">Metadata</th>
              </tr>
            </thead>
            <tbody>
              {auditQuery.data.items.map((item) => (
                <tr className="border-b border-slate-100 align-top" key={item.id}>
                  <td className="py-2 pr-3 text-xs text-slate-600">{new Date(item.createdAt).toLocaleString()}</td>
                  <td className="py-2 pr-3">
                    <p className="font-medium text-ink">{item.actorType}</p>
                    <p className="text-xs text-slate-600">{item.actorId}</p>
                  </td>
                  <td className="py-2 pr-3 text-slate-700">{item.action}</td>
                  <td className="py-2 pr-3 text-xs text-slate-600">{item.targetType || '-'} / {item.targetId || '-'}</td>
                  <td className="py-2 text-xs text-slate-600">{JSON.stringify(item.metadata ?? {})}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}

      {!auditQuery.data?.items?.length && !auditQuery.isLoading ? (
        <p className="mt-3 text-sm text-slate-600">No audit entries found.</p>
      ) : null}

      <div className="mt-4 flex items-center justify-between">
        <p className="text-xs text-slate-600">
          Page {page} of {maxPage} ({total} events)
        </p>
        <div className="flex gap-2">
          <button
            className="rounded-md border border-slate-300 px-3 py-1 text-xs font-semibold text-slate-700 transition hover:bg-slate-50 disabled:opacity-50"
            disabled={page <= 1 || auditQuery.isFetching}
            onClick={() => setPage((value) => Math.max(1, value - 1))}
            type="button"
          >
            Previous
          </button>
          <button
            className="rounded-md border border-slate-300 px-3 py-1 text-xs font-semibold text-slate-700 transition hover:bg-slate-50 disabled:opacity-50"
            disabled={page >= maxPage || auditQuery.isFetching}
            onClick={() => setPage((value) => Math.min(maxPage, value + 1))}
            type="button"
          >
            Next
          </button>
        </div>
      </div>
    </article>
  )
}

function AuthShell({
  title,
  subtitle,
  children,
  footer
}: {
  title: string
  subtitle: string
  children: ReactNode
  footer?: ReactNode
}) {
  return (
    <main className="min-h-screen bg-gradient-to-b from-mist via-[#f7fbfe] to-white px-4 py-8 text-ink">
      <section className="mx-auto grid max-w-5xl gap-6 md:grid-cols-[1.2fr_1fr]">
        <article className="rounded-3xl bg-gradient-to-br from-pine via-[#2b6f70] to-[#205356] p-8 text-white shadow-xl">
          <p className="text-xs uppercase tracking-[0.2em] text-amber">Orivis Identity Plane</p>
          <h1 className="mt-3 text-3xl font-semibold leading-tight">Secure auth UX for OAuth2 and forward-auth control</h1>
          <p className="mt-4 max-w-md text-sm text-white/80">
            This frontend runs through Vite dev proxy, so calls to /v1 are forwarded to your local Go API at localhost:8080.
          </p>
        </article>

        <article className="rounded-3xl border border-glacier/50 bg-white p-6 shadow-sm">
          <h2 className="text-xl font-semibold text-ink">{title}</h2>
          <p className="mt-2 text-sm text-slate-600">{subtitle}</p>
          <div className="mt-5">{children}</div>
          {footer ? <div className="mt-5 border-t border-slate-100 pt-4">{footer}</div> : null}
        </article>
      </section>
    </main>
  )
}

function ErrorNotice({ message }: { message: string }) {
  return <p className="mt-4 rounded-lg border border-rose-200 bg-rose-50 px-3 py-2 text-sm text-rose-700">{message}</p>
}

function SuccessNotice({ message }: { message: string }) {
  return <p className="mt-4 rounded-lg border border-emerald-200 bg-emerald-50 px-3 py-2 text-sm text-emerald-700">{message}</p>
}

function PageStatus({ label }: { label: string }) {
  return (
    <main className="grid min-h-screen place-items-center bg-mist px-4 text-center text-slate-700">
      <p className="text-sm">{label}</p>
    </main>
  )
}

function getErrorMessage(error: unknown): string {
  if (error instanceof Error && error.message) {
    return error.message
  }

  return 'Unexpected API error'
}

export default App
