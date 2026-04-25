function App() {
  return (
    <main className="min-h-screen bg-mist text-ink">
      <section className="mx-auto max-w-4xl px-6 py-16">
        <p className="text-sm uppercase tracking-[0.18em] text-pine">Orivis</p>
        <h1 className="mt-3 text-4xl font-semibold leading-tight md:text-5xl">
          Build your own authentication stack.
        </h1>
        <p className="mt-5 max-w-2xl text-lg text-slate-700">
          This dashboard will manage OAuth2 provider apps, forward-auth rules, and user sign-in methods from one place.
        </p>
        <div className="mt-8 rounded-2xl border border-glacier/40 bg-white p-6 shadow-sm">
          <p className="text-sm text-slate-600">API client status</p>
          <p className="mt-2 font-medium">Orval + React Query wiring is configured.</p>
        </div>
      </section>
    </main>
  )
}

export default App
